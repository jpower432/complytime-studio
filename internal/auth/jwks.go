// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const clockSkew = 60 * time.Second

// IDTokenClaims holds the verified, structured claims from an OIDC ID token.
type IDTokenClaims struct {
	Subject       string   `json:"sub"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Name          string   `json:"name"`
	Picture       string   `json:"picture"`
	Groups        []string `json:"groups,omitempty"`
	raw           map[string]any
}

// ExtractRolesClaim traverses a dot-separated path in the raw claims and
// returns the string slice at that path. Returns nil if not found or mismatched.
// Example: "realm_access.roles" or "roles".
func (c *IDTokenClaims) ExtractRolesClaim(dotPath string) []string {
	if dotPath == "" || c.raw == nil {
		return nil
	}
	parts := strings.Split(dotPath, ".")
	var cur any = c.raw
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[part]
	}
	switch v := cur.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				out = append(out, str)
			}
		}
		return out
	case string:
		return []string{v}
	}
	return nil
}

// jwk represents a single JSON Web Key.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwkSet represents a JWKS response body.
type jwkSet struct {
	Keys []jwk `json:"keys"`
}

// jwtHeader holds the fields we need from a JWT header.
type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

// JWKSCache caches JWKS keys with a configurable TTL.
// A key-miss beyond the TTL triggers an on-demand refetch.
type JWKSCache struct {
	mu        sync.RWMutex
	keys      map[string]jwk
	fetchedAt time.Time
	ttl       time.Duration
	jwksURL   string
}

// newJWKSCache creates a cache for the given JWKS URL. Default TTL is 1 hour.
func newJWKSCache(jwksURL string, ttl time.Duration) *JWKSCache {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &JWKSCache{
		keys:    make(map[string]jwk),
		ttl:     ttl,
		jwksURL: jwksURL,
	}
}

// Refresh unconditionally fetches fresh keys from the JWKS endpoint.
func (c *JWKSCache) Refresh(ctx context.Context) error {
	set, err := fetchJWKS(ctx, c.jwksURL)
	if err != nil {
		jwksRefreshTotal.Add("error", 1)
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = make(map[string]jwk, len(set.Keys))
	for _, k := range set.Keys {
		c.keys[k.Kid] = k
	}
	c.fetchedAt = time.Now()
	jwksRefreshTotal.Add("success", 1)
	return nil
}

// get returns the JWK for kid. Refetches if the cache is stale or kid is unknown.
func (c *JWKSCache) get(ctx context.Context, kid string) (jwk, error) {
	c.mu.RLock()
	k, ok := c.keys[kid]
	stale := time.Since(c.fetchedAt) > c.ttl
	c.mu.RUnlock()

	if ok && !stale {
		return k, nil
	}

	// kid-miss or TTL expired: refetch.
	if err := c.Refresh(ctx); err != nil {
		slog.Warn("jwks: refetch failed, serving stale key if available", "error", err)
		if ok {
			return k, nil
		}
		return jwk{}, fmt.Errorf("jwks: key %q not found and refetch failed: %w", kid, err)
	}

	c.mu.RLock()
	k, ok = c.keys[kid]
	c.mu.RUnlock()
	if !ok {
		return jwk{}, fmt.Errorf("jwks: key %q not found after refresh", kid)
	}
	return k, nil
}

// fetchJWKS fetches the JWKS document from the given URL.
func fetchJWKS(ctx context.Context, url string) (*jwkSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jwks fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("jwks endpoint %d: %s", resp.StatusCode, string(b))
	}
	var set jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, fmt.Errorf("jwks decode: %w", err)
	}
	return &set, nil
}

// rsaPublicKeyFromJWK constructs an *rsa.PublicKey from a JWK entry.
func rsaPublicKeyFromJWK(k jwk) (*rsa.PublicKey, error) {
	if k.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported JWK key type %q (only RSA supported)", k.Kty)
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode JWK n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode JWK e: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() {
		return nil, fmt.Errorf("JWK e value too large")
	}
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

// VerifyIDToken performs full cryptographic verification of an OIDC ID token.
// It checks: signature (RS256/384/512 via JWKS), iss, aud, exp, nbf.
func (c *JWKSCache) VerifyIDToken(ctx context.Context, idToken, wantIssuer, wantClientID string) (*IDTokenClaims, error) {
	parts := strings.SplitN(idToken, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed JWT: expected 3 dot-separated parts")
	}

	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("JWT header decode: %w", err)
	}
	var hdr jwtHeader
	if err := json.Unmarshal(headerRaw, &hdr); err != nil {
		return nil, fmt.Errorf("JWT header parse: %w", err)
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("JWT payload decode: %w", err)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("JWT signature decode: %w", err)
	}

	key, err := c.get(ctx, hdr.Kid)
	if err != nil {
		return nil, fmt.Errorf("JWT key lookup: %w", err)
	}
	rsaKey, err := rsaPublicKeyFromJWK(key)
	if err != nil {
		return nil, fmt.Errorf("JWT: build RSA key: %w", err)
	}

	// Select hash function based on algorithm.
	signingInput := parts[0] + "." + parts[1]
	var h hash.Hash
	var hashID crypto.Hash
	switch hdr.Alg {
	case "RS256":
		h = sha256.New()
		hashID = crypto.SHA256
	case "RS384":
		h = sha512.New384()
		hashID = crypto.SHA384
	case "RS512":
		h = sha512.New()
		hashID = crypto.SHA512
	default:
		return nil, fmt.Errorf("unsupported JWT algorithm %q", hdr.Alg)
	}
	_, _ = h.Write([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(rsaKey, hashID, h.Sum(nil), sig); err != nil {
		return nil, fmt.Errorf("JWT signature invalid: %w", err)
	}

	var rawClaims map[string]any
	if err := json.Unmarshal(payloadRaw, &rawClaims); err != nil {
		return nil, fmt.Errorf("JWT claims parse: %w", err)
	}

	// Validate issuer.
	iss, _ := rawClaims["iss"].(string)
	if strings.TrimRight(iss, "/") != strings.TrimRight(wantIssuer, "/") {
		return nil, fmt.Errorf("JWT: issuer mismatch: got %q want %q", iss, wantIssuer)
	}

	// Validate audience.
	switch aud := rawClaims["aud"].(type) {
	case string:
		if aud != wantClientID {
			return nil, fmt.Errorf("JWT: audience mismatch: got %q want %q", aud, wantClientID)
		}
	case []any:
		found := false
		for _, a := range aud {
			if s, ok := a.(string); ok && s == wantClientID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("JWT: client_id %q not in audience", wantClientID)
		}
	default:
		return nil, fmt.Errorf("JWT: audience claim missing or invalid")
	}

	// Validate exp (required).
	now := time.Now()
	exp, ok := rawClaims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("JWT: missing exp claim")
	}
	if now.After(time.Unix(int64(exp), 0).Add(clockSkew)) {
		return nil, fmt.Errorf("JWT: token expired")
	}

	// Validate nbf if present.
	if nbf, ok := rawClaims["nbf"].(float64); ok {
		if now.Before(time.Unix(int64(nbf), 0).Add(-clockSkew)) {
			return nil, fmt.Errorf("JWT: token not yet valid (nbf)")
		}
	}

	claims := &IDTokenClaims{raw: rawClaims}
	claims.Subject, _ = rawClaims["sub"].(string)
	claims.Email, _ = rawClaims["email"].(string)
	claims.EmailVerified, _ = rawClaims["email_verified"].(bool)
	claims.Name, _ = rawClaims["name"].(string)
	claims.Picture, _ = rawClaims["picture"].(string)
	if groups, ok := rawClaims["groups"].([]any); ok {
		for _, g := range groups {
			if s, ok := g.(string); ok {
				claims.Groups = append(claims.Groups, s)
			}
		}
	}
	return claims, nil
}
