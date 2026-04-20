// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestNewHandler_ValidKey(t *testing.T) {
	h, err := NewHandler(Config{}, testKey(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("handler is nil")
	}
}

func TestNewHandler_ShortKey(t *testing.T) {
	_, err := NewHandler(Config{}, []byte("too-short"))
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestNewHandler_NilKey(t *testing.T) {
	_, err := NewHandler(Config{}, nil)
	if err == nil {
		t.Fatal("expected error for nil key")
	}
}

func TestSessionCookieRoundTrip(t *testing.T) {
	h, err := NewHandler(Config{}, testKey(t))
	if err != nil {
		t.Fatal(err)
	}

	original := &Session{
		GitHubToken: "ghp_test123",
		Login:       "octocat",
		Name:        "Octo Cat",
		AvatarURL:   "https://github.com/octocat.png",
		Email:       "octo@example.com",
		ExpiresAt:   time.Now().Add(sessionMaxAge).Unix(),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if err := h.setSessionCookie(rec, req, original); err != nil {
		t.Fatalf("setSessionCookie: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, sessionCookieName)
	}
	if !cookie.HttpOnly {
		t.Fatal("cookie should be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("cookie should be SameSite=Lax")
	}

	readReq := httptest.NewRequest(http.MethodGet, "/", nil)
	readReq.AddCookie(cookie)

	decoded, err := h.sessionFromCookie(readReq)
	if err != nil {
		t.Fatalf("sessionFromCookie: %v", err)
	}
	if decoded.GitHubToken != original.GitHubToken {
		t.Errorf("token = %q, want %q", decoded.GitHubToken, original.GitHubToken)
	}
	if decoded.Login != original.Login {
		t.Errorf("login = %q, want %q", decoded.Login, original.Login)
	}
	if decoded.Email != original.Email {
		t.Errorf("email = %q, want %q", decoded.Email, original.Email)
	}
}

func TestSessionCookieRoundTrip_WrongKey(t *testing.T) {
	h1, _ := NewHandler(Config{}, testKey(t))
	h2, _ := NewHandler(Config{}, testKey(t))

	sess := &Session{Login: "octocat", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_ = h1.setSessionCookie(rec, req, sess)

	readReq := httptest.NewRequest(http.MethodGet, "/", nil)
	readReq.AddCookie(rec.Result().Cookies()[0])

	_, err := h2.sessionFromCookie(readReq)
	if err == nil {
		t.Fatal("expected decryption error with wrong key")
	}
}

func TestSessionFromCookie_MalformedValue(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "not-base64-!!!"})
	_, err := h.sessionFromCookie(req)
	if err == nil {
		t.Fatal("expected error for malformed cookie")
	}
}

func TestTokenFromRequest(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))

	sess := &Session{GitHubToken: "ghp_abc", Login: "user", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_ = h.setSessionCookie(rec, req, sess)

	readReq := httptest.NewRequest(http.MethodGet, "/api/a2a/agent", nil)
	readReq.AddCookie(rec.Result().Cookies()[0])

	token, ok := h.TokenFromRequest(readReq)
	if !ok || token != "ghp_abc" {
		t.Fatalf("TokenFromRequest = (%q, %v), want (ghp_abc, true)", token, ok)
	}
}

func TestTokenFromRequest_NoCookie(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))
	req := httptest.NewRequest(http.MethodGet, "/api/a2a/agent", nil)
	_, ok := h.TokenFromRequest(req)
	if ok {
		t.Fatal("expected false when no cookie present")
	}
}

func TestMiddleware_BlocksUnauthenticatedAPI(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := h.Middleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestMiddleware_AllowsConfigWithoutAuth(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := h.Middleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for /api/config", rec.Code)
	}
}

func TestMiddleware_AllowsNonAPIPaths(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := h.Middleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for non-/api/ path", rec.Code)
	}
}

func TestMiddleware_AllowsValidSession(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))

	var gotSession bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s, ok := SessionFrom(r.Context()); ok && s.Login == "testuser" {
			gotSession = true
		}
		w.WriteHeader(http.StatusOK)
	})

	sess := &Session{Login: "testuser", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookieRec := httptest.NewRecorder()
	cookieReq := httptest.NewRequest(http.MethodGet, "/", nil)
	_ = h.setSessionCookie(cookieRec, cookieReq, sess)

	handler := h.Middleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.AddCookie(cookieRec.Result().Cookies()[0])
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !gotSession {
		t.Fatal("session not injected into context")
	}
}

func TestMiddleware_RejectsExpiredSession(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sess := &Session{Login: "expired", ExpiresAt: time.Now().Add(-time.Hour).Unix()}
	cookieRec := httptest.NewRecorder()
	cookieReq := httptest.NewRequest(http.MethodGet, "/", nil)
	_ = h.setSessionCookie(cookieRec, cookieReq, sess)

	handler := h.Middleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.AddCookie(cookieRec.Result().Cookies()[0])
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for expired session", rec.Code)
	}
}

func TestHandleMe_ValidSession(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))

	sess := &Session{
		Login: "octocat", Name: "Octo Cat",
		AvatarURL: "https://example.com/avatar.png", Email: "a@b.com",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	cookieRec := httptest.NewRecorder()
	cookieReq := httptest.NewRequest(http.MethodGet, "/", nil)
	_ = h.setSessionCookie(cookieRec, cookieReq, sess)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.AddCookie(cookieRec.Result().Cookies()[0])
	h.handleMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var info UserInfo
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.Login != "octocat" {
		t.Errorf("login = %q, want octocat", info.Login)
	}
}

func TestHandleMe_NoSession(t *testing.T) {
	h, _ := NewHandler(Config{}, testKey(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	h.handleMe(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestIsSecureRequest(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{"no header, no TLS", "", false},
		{"X-Forwarded-Proto https", "https", true},
		{"X-Forwarded-Proto HTTPS", "HTTPS", true},
		{"X-Forwarded-Proto http", "http", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("X-Forwarded-Proto", tt.header)
			}
			got := isSecureRequest(req)
			if got != tt.want {
				t.Fatalf("isSecureRequest = %v, want %v", got, tt.want)
			}
		})
	}
}
