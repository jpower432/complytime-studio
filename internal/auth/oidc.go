// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// OIDCProvider holds endpoints discovered from an OIDC issuer's
// well-known configuration document.
type OIDCProvider struct {
	IssuerURL   string
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	JWKSURL     string
}

// discoveryDoc maps the fields in /.well-known/openid-configuration.
type discoveryDoc struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

// Discover fetches and parses the OIDC discovery document for issuerURL.
// It validates required fields and performs issuer normalization per the OIDC spec.
func Discover(ctx context.Context, issuerURL string) (*OIDCProvider, error) {
	issuerURL = strings.TrimRight(issuerURL, "/")
	wellKnown := issuerURL + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, fmt.Errorf("oidc discover: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc discover %s: %w", wellKnown, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("oidc discover: %s returned %d: %s", wellKnown, resp.StatusCode, string(b))
	}

	const maxDiscoveryBody = 1 << 20 // 1 MiB
	var doc discoveryDoc
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxDiscoveryBody)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("oidc discover: decode: %w", err)
	}

	for _, pair := range []struct{ name, val string }{
		{"authorization_endpoint", doc.AuthorizationEndpoint},
		{"token_endpoint", doc.TokenEndpoint},
		{"jwks_uri", doc.JwksURI},
	} {
		if pair.val == "" {
			return nil, fmt.Errorf("oidc discover: missing required field %q", pair.name)
		}
	}

	// OIDC spec requires issuer to match exactly (modulo trailing slash).
	if doc.Issuer != "" && strings.TrimRight(doc.Issuer, "/") != issuerURL {
		slog.Warn("oidc: discovered issuer does not match configured issuerURL — using configured value",
			"configured", issuerURL, "discovered", doc.Issuer)
	}

	oidcDiscoverySuccess.Add(1)
	return &OIDCProvider{
		IssuerURL:   issuerURL,
		AuthURL:     doc.AuthorizationEndpoint,
		TokenURL:    doc.TokenEndpoint,
		UserInfoURL: doc.UserinfoEndpoint,
		JWKSURL:     doc.JwksURI,
	}, nil
}

// retryBaseDelay is the initial backoff for DiscoverWithRetry.
// Overridable in tests to avoid multi-second waits.
var retryBaseDelay = 2 * time.Second

// DiscoverWithRetry calls Discover with bounded exponential backoff.
// Use at startup where the IdP may not be reachable immediately.
// base: 2s, cap: 30s, maxTotal: 5min per the design spec.
func DiscoverWithRetry(ctx context.Context, issuerURL string) (*OIDCProvider, error) {
	const (
		backCap  = 30 * time.Second
		maxTotal = 5 * time.Minute
	)
	deadline := time.Now().Add(maxTotal)
	delay := retryBaseDelay

	for {
		provider, err := Discover(ctx, issuerURL)
		if err == nil {
			return provider, nil
		}
		oidcDiscoveryFailure.Add(1)

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("oidc discovery failed after %s: %w", maxTotal, err)
		}

		slog.Warn("oidc: discovery failed, retrying", "error", err, "retryIn", delay)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > backCap {
			delay = backCap
		}
	}
}

// StartRefreshLoop starts a background goroutine that periodically re-discovers
// OIDC configuration and calls onRefresh with the updated provider.
// Refresh failures are logged but do not stop the loop.
// interval defaults to 24h if zero.
func StartRefreshLoop(ctx context.Context, issuerURL string, interval time.Duration, onRefresh func(*OIDCProvider)) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				provider, err := Discover(ctx, issuerURL)
				if err != nil {
					oidcDiscoveryFailure.Add(1)
					slog.Warn("oidc: periodic refresh failed", "error", err)
					continue
				}
				onRefresh(provider)
				slog.Info("oidc: discovery refreshed", "issuer", issuerURL)
			}
		}
	}()
}
