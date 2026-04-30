// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

// ConfigFromEnv builds a Config from OIDC_* env vars, with GOOGLE_* as
// deprecated aliases. When GOOGLE_* vars are set and OIDC_* equivalents are
// empty, they are mapped to Google's OIDC issuer with a deprecation warning.
func ConfigFromEnv() Config {
	issuerURL := os.Getenv("OIDC_ISSUER_URL")
	clientID := os.Getenv("OIDC_CLIENT_ID")
	clientSecret := os.Getenv("OIDC_CLIENT_SECRET")
	callbackURL := envOr("OIDC_CALLBACK_URL", "http://localhost:8080/auth/callback")

	if clientID == "" && os.Getenv("GOOGLE_CLIENT_ID") != "" {
		slog.Warn("GOOGLE_* auth vars are deprecated — migrate to OIDC_* (removal planned for v2.0)",
			"hint", "set OIDC_ISSUER_URL=https://accounts.google.com and rename GOOGLE_CLIENT_ID/SECRET to OIDC_CLIENT_ID/SECRET")
		clientID = os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
		if issuerURL == "" {
			issuerURL = "https://accounts.google.com"
		}
		if cb := os.Getenv("GOOGLE_CALLBACK_URL"); cb != "" && callbackURL == "http://localhost:8080/auth/callback" {
			callbackURL = cb
		}
	}

	var bootstrapEmails []string
	if raw := os.Getenv("OIDC_BOOTSTRAP_EMAILS"); raw != "" {
		bootstrapEmails = splitEnvComma(raw)
	}

	var provider *OIDCProvider
	if issuerURL != "" {
		provider = &OIDCProvider{IssuerURL: issuerURL}
	}

	return Config{
		ClientID:        clientID,
		ClientSecret:    clientSecret,
		CallbackURL:     callbackURL,
		Provider:        provider,
		Scopes:          envOr("OIDC_SCOPES", "openid email profile"),
		RolesClaim:      os.Getenv("OIDC_ROLES_CLAIM"),
		BootstrapEmails: bootstrapEmails,
	}
}

// ParseDiscoveryRefreshInterval parses OIDC_DISCOVERY_REFRESH (default 24h).
func ParseDiscoveryRefreshInterval() time.Duration {
	raw := os.Getenv("OIDC_DISCOVERY_REFRESH")
	if raw == "" {
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		slog.Warn("invalid OIDC_DISCOVERY_REFRESH, using 24h", "value", raw, "error", err)
		return 24 * time.Hour
	}
	return d
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitEnvComma(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
