// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// OIDC Discovery tests
// ---------------------------------------------------------------------------

func mockDiscoveryServer(t *testing.T, statusCode int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

func validDiscoveryBody(issuer string) string {
	return fmt.Sprintf(`{
		"issuer": %q,
		"authorization_endpoint": %q,
		"token_endpoint": %q,
		"userinfo_endpoint": %q,
		"jwks_uri": %q
	}`,
		issuer,
		issuer+"/auth",
		issuer+"/token",
		issuer+"/userinfo",
		issuer+"/jwks",
	)
}

func TestDiscover_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validDiscoveryBody("http://fake-issuer")))
	}))
	t.Cleanup(srv.Close)

	provider, err := Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	if provider.AuthURL == "" {
		t.Error("AuthURL should be populated")
	}
	if provider.TokenURL == "" {
		t.Error("TokenURL should be populated")
	}
	if provider.JWKSURL == "" {
		t.Error("JWKSURL should be populated")
	}
}

func TestDiscover_MissingRequiredField(t *testing.T) {
	srv := mockDiscoveryServer(t, http.StatusOK, `{"issuer":"http://x","authorization_endpoint":"http://x/auth"}`)
	t.Cleanup(srv.Close)

	_, err := Discover(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for missing token_endpoint and jwks_uri")
	}
}

func TestDiscover_Non200(t *testing.T) {
	srv := mockDiscoveryServer(t, http.StatusServiceUnavailable, "unavailable")
	t.Cleanup(srv.Close)

	_, err := Discover(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for 503")
	}
}

func TestDiscoverWithRetry_SucceedsAfterTransientFailure(t *testing.T) {
	attempts := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validDiscoveryBody(srv.URL)))
	}))
	t.Cleanup(srv.Close)

	// Speed up backoff so the test completes in milliseconds not seconds.
	orig := retryBaseDelay
	retryBaseDelay = 5 * time.Millisecond
	t.Cleanup(func() { retryBaseDelay = orig })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := DiscoverWithRetry(ctx, srv.URL)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if provider == nil {
		t.Fatal("provider should not be nil")
	}
	if attempts < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts)
	}
}

func TestDiscoverWithRetry_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	orig := retryBaseDelay
	retryBaseDelay = 5 * time.Millisecond
	t.Cleanup(func() { retryBaseDelay = orig })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := DiscoverWithRetry(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

func TestStartRefreshLoop_CallsOnRefresh(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validDiscoveryBody(srv.URL)))
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := make(chan *OIDCProvider, 1)
	StartRefreshLoop(ctx, srv.URL, 10*time.Millisecond, func(p *OIDCProvider) {
		select {
		case called <- p:
		default:
		}
	})

	select {
	case p := <-called:
		if p == nil {
			t.Fatal("onRefresh received nil provider")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("onRefresh was not called within 2s")
	}
}

func TestStartRefreshLoop_StopsOnContextCancel(t *testing.T) {
	callCount := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validDiscoveryBody(srv.URL)))
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	StartRefreshLoop(ctx, srv.URL, 10*time.Millisecond, func(p *OIDCProvider) {
		callCount++
		if callCount == 1 {
			cancel()
			close(done)
		}
	})

	<-done
	time.Sleep(50 * time.Millisecond) // give loop time to exit
	snapshot := callCount
	time.Sleep(50 * time.Millisecond) // ensure no further calls arrive
	if callCount != snapshot {
		t.Errorf("loop continued after context cancel: callCount changed from %d to %d", snapshot, callCount)
	}
}

// ---------------------------------------------------------------------------
// PKCE tests
// ---------------------------------------------------------------------------

func TestPKCE_CodeChallenge(t *testing.T) {
	verifier := generateCodeVerifier()
	if verifier == "" {
		t.Fatal("code_verifier should not be empty")
	}

	challenge := pkceChallenge(verifier)
	if challenge == "" {
		t.Fatal("code_challenge should not be empty")
	}

	// Verify S256: challenge == base64url(sha256(verifier))
	h := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(h[:])
	if challenge != expected {
		t.Errorf("pkceChallenge = %q, want %q", challenge, expected)
	}
}

func TestPKCE_DifferentVerifiersProduceDifferentChallenges(t *testing.T) {
	v1 := generateCodeVerifier()
	v2 := generateCodeVerifier()
	if v1 == v2 {
		t.Error("two generated verifiers should not be equal")
	}
	if pkceChallenge(v1) == pkceChallenge(v2) {
		t.Error("different verifiers should produce different challenges")
	}
}

// ---------------------------------------------------------------------------
// JWT / JWKS tests
// ---------------------------------------------------------------------------

// testRSAKey generates a 2048-bit RSA key for tests.
func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// buildTestJWT constructs a signed RS256 JWT with the given claims.
func buildTestJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	hdr := map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"}
	hdrJSON, _ := json.Marshal(hdr)
	claimsJSON, _ := json.Marshal(claims)

	hdrB64 := base64.RawURLEncoding.EncodeToString(hdrJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := hdrB64 + "." + claimsB64

	h := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// jwkFromKey builds a JWK from an RSA public key for use in a test JWKS cache.
func jwkFromKey(key *rsa.PublicKey, kid string) jwk {
	n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
	return jwk{Kty: "RSA", Kid: kid, Alg: "RS256", Use: "sig", N: n, E: e}
}

// cacheWithKey creates a JWKSCache pre-populated with the given key.
func cacheWithKey(t *testing.T, key *rsa.PrivateKey, kid string) *JWKSCache {
	t.Helper()
	c := newJWKSCache("http://unused", time.Hour)
	c.keys[kid] = jwkFromKey(&key.PublicKey, kid)
	c.fetchedAt = time.Now()
	return c
}

func validClaims(issuer, clientID string) map[string]any {
	return map[string]any{
		"iss":            issuer,
		"aud":            clientID,
		"sub":            "user-123",
		"email":          "user@example.com",
		"email_verified": true,
		"name":           "Test User",
		"exp":            float64(time.Now().Add(time.Hour).Unix()),
	}
}

func TestVerifyIDToken_ValidSignature(t *testing.T) {
	key := testRSAKey(t)
	cache := cacheWithKey(t, key, "key1")

	claims := validClaims("https://issuer.example.com", "client-abc")
	token := buildTestJWT(t, key, "key1", claims)

	got, err := cache.VerifyIDToken(context.Background(), token, "https://issuer.example.com", "client-abc")
	if err != nil {
		t.Fatalf("VerifyIDToken: unexpected error: %v", err)
	}
	if got.Subject != "user-123" {
		t.Errorf("Subject = %q, want user-123", got.Subject)
	}
	if got.Email != "user@example.com" {
		t.Errorf("Email = %q, want user@example.com", got.Email)
	}
	if !got.EmailVerified {
		t.Error("EmailVerified should be true")
	}
}

func TestVerifyIDToken_InvalidSignature(t *testing.T) {
	key1 := testRSAKey(t)
	key2 := testRSAKey(t)
	// Cache has key2 public key, but token is signed with key1.
	cache := cacheWithKey(t, key2, "key1")

	claims := validClaims("https://issuer.example.com", "client-abc")
	token := buildTestJWT(t, key1, "key1", claims)

	_, err := cache.VerifyIDToken(context.Background(), token, "https://issuer.example.com", "client-abc")
	if err == nil {
		t.Fatal("expected signature verification error")
	}
}

func TestVerifyIDToken_ExpiredToken(t *testing.T) {
	key := testRSAKey(t)
	cache := cacheWithKey(t, key, "key1")

	claims := validClaims("https://issuer.example.com", "client-abc")
	claims["exp"] = float64(time.Now().Add(-2 * time.Hour).Unix()) // expired + past clock skew
	token := buildTestJWT(t, key, "key1", claims)

	_, err := cache.VerifyIDToken(context.Background(), token, "https://issuer.example.com", "client-abc")
	if err == nil {
		t.Fatal("expected expired token error")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("error should mention expiry, got: %v", err)
	}
}

func TestVerifyIDToken_WrongAudience(t *testing.T) {
	key := testRSAKey(t)
	cache := cacheWithKey(t, key, "key1")

	claims := validClaims("https://issuer.example.com", "other-client")
	token := buildTestJWT(t, key, "key1", claims)

	_, err := cache.VerifyIDToken(context.Background(), token, "https://issuer.example.com", "client-abc")
	if err == nil {
		t.Fatal("expected audience mismatch error")
	}
}

func TestVerifyIDToken_WrongIssuer(t *testing.T) {
	key := testRSAKey(t)
	cache := cacheWithKey(t, key, "key1")

	claims := validClaims("https://evil.example.com", "client-abc")
	token := buildTestJWT(t, key, "key1", claims)

	_, err := cache.VerifyIDToken(context.Background(), token, "https://issuer.example.com", "client-abc")
	if err == nil {
		t.Fatal("expected issuer mismatch error")
	}
}

// ---------------------------------------------------------------------------
// Role seeding tests
// ---------------------------------------------------------------------------

// mockUserStore is a minimal in-memory UserStore for handler tests.
type mockUserStore struct {
	users       map[string]*auth_User // keyed by email
	usersBySub  map[string]*auth_User // keyed by "sub|issuer"
	roleChanges []RoleChange
	adminCount  int
}

// auth_User is an alias to avoid import cycle (same package).
type auth_User = User

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		users:      make(map[string]*User),
		usersBySub: make(map[string]*User),
	}
}

func (m *mockUserStore) UpsertUser(_ context.Context, sub, issuer, email, name, avatarURL string) error {
	u := &User{Sub: sub, Issuer: issuer, Email: email, Name: name, AvatarURL: avatarURL, Role: "reviewer"}
	if existing, ok := m.users[email]; ok {
		u.Role = existing.Role
	}
	m.users[email] = u
	m.usersBySub[sub+"|"+issuer] = u
	return nil
}

func (m *mockUserStore) GetUser(_ context.Context, email string) (*User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserStore) GetUserBySub(_ context.Context, sub, issuer string) (*User, error) {
	u, ok := m.usersBySub[sub+"|"+issuer]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserStore) ListUsers(_ context.Context) ([]User, error)         { return nil, nil }
func (m *mockUserStore) CountUsers(_ context.Context) (int, error)           { return len(m.users), nil }
func (m *mockUserStore) ListRoleChanges(_ context.Context) ([]RoleChange, error) {
	return m.roleChanges, nil
}
func (m *mockUserStore) InsertRoleChange(_ context.Context, rc RoleChange) error {
	m.roleChanges = append(m.roleChanges, rc)
	return nil
}
func (m *mockUserStore) CountAdmins(_ context.Context) (int, error) { return m.adminCount, nil }
func (m *mockUserStore) SetRole(_ context.Context, email, role string) (string, error) {
	u, ok := m.users[email]
	if !ok {
		return "", ErrUserNotFound
	}
	old := u.Role
	u.Role = role
	if role == "admin" {
		m.adminCount++
	}
	return old, nil
}

func testHandlerWithStore(t *testing.T, cfg Config, store *mockUserStore) *Handler {
	t.Helper()
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	h, err := NewHandler(cfg, key, NewMemorySessionStore())
	if err != nil {
		t.Fatal(err)
	}
	h.users = store
	return h
}

func TestCallback_RoleSeedFromJWT(t *testing.T) {
	store := newMockUserStore()
	cfg := Config{
		Provider:   &OIDCProvider{IssuerURL: "https://issuer.example.com"},
		RolesClaim: "roles",
	}
	h := testHandlerWithStore(t, cfg, store)

	claims := &IDTokenClaims{
		Subject:       "sub-001",
		Email:         "admin@example.com",
		EmailVerified: true,
		raw:           map[string]any{"roles": []any{"admin", "user"}},
	}
	user := &oidcUser{Email: "admin@example.com", Name: "Admin User"}

	h.seedUserRole(context.Background(), claims, user)

	u, _ := store.GetUser(context.Background(), "admin@example.com")
	if u == nil || u.Role != "admin" {
		t.Errorf("role = %v, want admin", u)
	}
	if len(store.roleChanges) != 1 || store.roleChanges[0].ChangedBy != "jwt-seed" {
		t.Error("expected jwt-seed role change record")
	}
}

func TestCallback_UnverifiedEmailNoAdmin(t *testing.T) {
	store := newMockUserStore()
	cfg := Config{
		Provider:   &OIDCProvider{IssuerURL: "https://issuer.example.com"},
		RolesClaim: "roles",
	}
	h := testHandlerWithStore(t, cfg, store)

	claims := &IDTokenClaims{
		Subject:       "sub-002",
		Email:         "unverified@example.com",
		EmailVerified: false, // unverified
		raw:           map[string]any{"roles": []any{"admin"}},
	}
	user := &oidcUser{Email: "unverified@example.com", Name: "Unverified"}

	h.seedUserRole(context.Background(), claims, user)

	u, _ := store.GetUser(context.Background(), "unverified@example.com")
	if u != nil && u.Role == "admin" {
		t.Error("unverified email should not be seeded as admin")
	}
}

func TestCallback_BootstrapEmailAllowlist(t *testing.T) {
	store := newMockUserStore()
	cfg := Config{
		Provider:        &OIDCProvider{IssuerURL: "https://issuer.example.com"},
		BootstrapEmails: []string{"allowed@example.com"},
	}
	h := testHandlerWithStore(t, cfg, store)

	// Non-allowlisted user — even with admin JWT role — should stay reviewer.
	claims := &IDTokenClaims{
		Subject:       "sub-003",
		Email:         "notallowed@example.com",
		EmailVerified: true,
		raw:           map[string]any{"roles": []any{"admin"}},
	}
	user := &oidcUser{Email: "notallowed@example.com", Name: "Not Allowed"}
	h.seedUserRole(context.Background(), claims, user)

	u, _ := store.GetUser(context.Background(), "notallowed@example.com")
	if u != nil && u.Role == "admin" {
		t.Error("non-allowlisted email should not become admin")
	}
}

func TestCallback_ExistingUserNoOverride(t *testing.T) {
	store := newMockUserStore()
	// Pre-seed user as reviewer.
	_ = store.UpsertUser(context.Background(), "sub-004", "https://issuer.example.com", "returning@example.com", "Returning", "")

	cfg := Config{
		Provider:   &OIDCProvider{IssuerURL: "https://issuer.example.com"},
		RolesClaim: "roles",
	}
	h := testHandlerWithStore(t, cfg, store)

	// Returning user with admin JWT claim — role should NOT change.
	claims := &IDTokenClaims{
		Subject:       "sub-004",
		Email:         "returning@example.com",
		EmailVerified: true,
		raw:           map[string]any{"roles": []any{"admin"}},
	}
	user := &oidcUser{Email: "returning@example.com", Name: "Returning"}
	h.seedUserRole(context.Background(), claims, user)

	u, _ := store.GetUser(context.Background(), "returning@example.com")
	if u.Role == "admin" {
		t.Error("existing user's role should not be overridden by JWT claim")
	}
}

func TestCallback_FirstAdminPromotion(t *testing.T) {
	store := newMockUserStore()
	cfg := Config{
		Provider: &OIDCProvider{IssuerURL: "https://issuer.example.com"},
	}
	h := testHandlerWithStore(t, cfg, store)

	claims := &IDTokenClaims{
		Subject:       "sub-005",
		Email:         "first@example.com",
		EmailVerified: true,
		raw:           map[string]any{},
	}
	user := &oidcUser{Email: "first@example.com", Name: "First User"}
	h.seedUserRole(context.Background(), claims, user)

	u, _ := store.GetUser(context.Background(), "first@example.com")
	if u == nil || u.Role != "admin" {
		t.Errorf("first user should be promoted to admin, got role %v", u)
	}
}

// ---------------------------------------------------------------------------
// Env alias tests
// ---------------------------------------------------------------------------

func TestEnvAliases_GoogleDeprecated(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "google-client-123")
	t.Setenv("GOOGLE_CLIENT_SECRET", "google-secret-456")
	t.Setenv("OIDC_CLIENT_ID", "")
	t.Setenv("OIDC_ISSUER_URL", "")

	cfg := ConfigFromEnv()

	if cfg.ClientID != "google-client-123" {
		t.Errorf("ClientID = %q, want google-client-123", cfg.ClientID)
	}
	if cfg.Provider == nil || cfg.Provider.IssuerURL != "https://accounts.google.com" {
		t.Errorf("Provider.IssuerURL = %v, want https://accounts.google.com", cfg.Provider)
	}
}

func TestEnvAliases_OIDCTakesPrecedence(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "google-client-123")
	t.Setenv("OIDC_CLIENT_ID", "oidc-client-456")
	t.Setenv("OIDC_ISSUER_URL", "https://keycloak.example.com/realms/myrealm")

	cfg := ConfigFromEnv()

	if cfg.ClientID != "oidc-client-456" {
		t.Errorf("OIDC_CLIENT_ID should take precedence, got %q", cfg.ClientID)
	}
	if cfg.Provider == nil || cfg.Provider.IssuerURL != "https://keycloak.example.com/realms/myrealm" {
		t.Errorf("IssuerURL = %v, want keycloak URL", cfg.Provider)
	}
}

// ---------------------------------------------------------------------------
// IDTokenClaims.ExtractRolesClaim tests
// ---------------------------------------------------------------------------

func TestExtractRolesClaim_DotPath(t *testing.T) {
	claims := &IDTokenClaims{
		raw: map[string]any{
			"realm_access": map[string]any{
				"roles": []any{"admin", "user"},
			},
		},
	}
	roles := claims.ExtractRolesClaim("realm_access.roles")
	if len(roles) != 2 || roles[0] != "admin" {
		t.Errorf("ExtractRolesClaim = %v, want [admin user]", roles)
	}
}

func TestExtractRolesClaim_TopLevel(t *testing.T) {
	claims := &IDTokenClaims{
		raw: map[string]any{"roles": []any{"reviewer"}},
	}
	roles := claims.ExtractRolesClaim("roles")
	if len(roles) != 1 || roles[0] != "reviewer" {
		t.Errorf("ExtractRolesClaim = %v, want [reviewer]", roles)
	}
}

func TestExtractRolesClaim_Missing(t *testing.T) {
	claims := &IDTokenClaims{raw: map[string]any{}}
	roles := claims.ExtractRolesClaim("nonexistent")
	if roles != nil {
		t.Errorf("expected nil for missing claim, got %v", roles)
	}
}

func TestExtractRolesClaim_Empty(t *testing.T) {
	claims := &IDTokenClaims{raw: map[string]any{}}
	roles := claims.ExtractRolesClaim("")
	if roles != nil {
		t.Errorf("expected nil for empty path, got %v", roles)
	}
}
