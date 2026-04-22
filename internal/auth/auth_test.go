// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func testHandler(t *testing.T) *Handler {
	t.Helper()
	h, err := NewHandler(Config{}, testKey(t), NewMemorySessionStore())
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func createSession(t *testing.T, h *Handler, sess ServerSession) *http.Cookie {
	t.Helper()
	sid := generateSessionID()
	if err := h.store.Put(nil, sid, sess); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if err := h.setSessionCookie(rec, req, sid); err != nil {
		t.Fatal(err)
	}
	return rec.Result().Cookies()[0]
}

func TestNewHandler_ValidKey(t *testing.T) {
	h, err := NewHandler(Config{}, testKey(t), NewMemorySessionStore())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("handler is nil")
	}
}

func TestNewHandler_ShortKey(t *testing.T) {
	_, err := NewHandler(Config{}, []byte("too-short"), NewMemorySessionStore())
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestNewHandler_NilKey(t *testing.T) {
	_, err := NewHandler(Config{}, nil, NewMemorySessionStore())
	if err == nil {
		t.Fatal("expected error for nil key")
	}
}

func TestSessionCookieRoundTrip(t *testing.T) {
	h := testHandler(t)

	original := ServerSession{
		AccessToken: "ya29.test123",
		Login:       "user@example.com",
		Name:        "Test User",
		AvatarURL:   "https://lh3.googleusercontent.com/photo.jpg",
		Email:       "user@example.com",
		ExpiresAt:   time.Now().Add(sessionMaxAge).Unix(),
	}
	cookie := createSession(t, h, original)

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

	sid, err := h.sessionIDFromCookie(readReq)
	if err != nil {
		t.Fatalf("sessionIDFromCookie: %v", err)
	}
	decoded, err := h.store.Get(readReq.Context(), sid)
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	if decoded.AccessToken != original.AccessToken {
		t.Errorf("token = %q, want %q", decoded.AccessToken, original.AccessToken)
	}
	if decoded.Login != original.Login {
		t.Errorf("login = %q, want %q", decoded.Login, original.Login)
	}
	if decoded.Email != original.Email {
		t.Errorf("email = %q, want %q", decoded.Email, original.Email)
	}
}

func TestSessionCookieRoundTrip_WrongKey(t *testing.T) {
	h1 := testHandler(t)
	h2 := testHandler(t)

	sess := ServerSession{Login: "octocat", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h1, sess)

	readReq := httptest.NewRequest(http.MethodGet, "/", nil)
	readReq.AddCookie(cookie)

	_, err := h2.sessionIDFromCookie(readReq)
	if err == nil {
		t.Fatal("expected decryption error with wrong key")
	}
}

func TestSessionIDFromCookie_MalformedValue(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "not-base64-!!!"})
	_, err := h.sessionIDFromCookie(req)
	if err == nil {
		t.Fatal("expected error for malformed cookie")
	}
}

func TestTokenFromRequest(t *testing.T) {
	h := testHandler(t)

	sess := ServerSession{AccessToken: "ya29.abc", Login: "user@example.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	readReq := httptest.NewRequest(http.MethodGet, "/api/a2a/agent", nil)
	readReq.AddCookie(cookie)

	token, ok := h.TokenFromRequest(readReq)
	if !ok || token != "ya29.abc" {
		t.Fatalf("TokenFromRequest = (%q, %v), want (ya29.abc, true)", token, ok)
	}
}

func TestTokenFromRequest_NoCookie(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/a2a/agent", nil)
	_, ok := h.TokenFromRequest(req)
	if ok {
		t.Fatal("expected false when no cookie present")
	}
}

func TestTokenFromRequest_NotInCookie(t *testing.T) {
	h := testHandler(t)
	sid := generateSessionID()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_ = h.setSessionCookie(rec, req, sid)

	readReq := httptest.NewRequest(http.MethodGet, "/api/a2a/agent", nil)
	readReq.AddCookie(rec.Result().Cookies()[0])

	token, ok := h.TokenFromRequest(readReq)
	if ok {
		t.Fatalf("expected false for missing session, got token %q", token)
	}
}

func TestMiddleware_BlocksUnauthenticatedAPI(t *testing.T) {
	h := testHandler(t)
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
	h := testHandler(t)
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
	h := testHandler(t)
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
	h := testHandler(t)

	var gotSession bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s, ok := SessionFrom(r.Context()); ok && s.Login == "testuser" {
			gotSession = true
		}
		w.WriteHeader(http.StatusOK)
	})

	sess := ServerSession{Login: "testuser", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	handler := h.Middleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.AddCookie(cookie)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !gotSession {
		t.Fatal("session not injected into context")
	}
}

func TestMiddleware_RejectsExpiredSession(t *testing.T) {
	h := testHandler(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sess := ServerSession{Login: "expired", ExpiresAt: time.Now().Add(-time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	handler := h.Middleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.AddCookie(cookie)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for expired session", rec.Code)
	}
}

func TestHandleMe_ValidSession(t *testing.T) {
	h := testHandler(t)

	sess := ServerSession{
		Login: "octocat", Name: "Octo Cat",
		AvatarURL: "https://example.com/avatar.png", Email: "a@b.com",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	cookie := createSession(t, h, sess)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.AddCookie(cookie)
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
	h := testHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	h.handleMe(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandleLogout_DeletesSession(t *testing.T) {
	h := testHandler(t)
	store := h.store.(*MemorySessionStore)

	sess := ServerSession{Login: "user", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	if store.Len() != 1 {
		t.Fatalf("store should have 1 session before logout, got %d", store.Len())
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(cookie)
	h.handleLogout(rec, req)

	if store.Len() != 0 {
		t.Fatalf("store should have 0 sessions after logout, got %d", store.Len())
	}
}

func TestRoleForEmail_EmptyAdmins(t *testing.T) {
	role := RoleForEmail("anyone@example.com", map[string]bool{})
	if role != "admin" {
		t.Fatalf("role = %q, want admin (empty allowlist)", role)
	}
}

func TestRoleForEmail_InList(t *testing.T) {
	admins := map[string]bool{"admin@co.com": true}
	if got := RoleForEmail("admin@co.com", admins); got != "admin" {
		t.Fatalf("role = %q, want admin", got)
	}
}

func TestRoleForEmail_NotInList(t *testing.T) {
	admins := map[string]bool{"admin@co.com": true}
	if got := RoleForEmail("viewer@co.com", admins); got != "viewer" {
		t.Fatalf("role = %q, want viewer", got)
	}
}

func TestRoleForEmail_CaseInsensitive(t *testing.T) {
	admins := map[string]bool{"admin@co.com": true}
	if got := RoleForEmail("Admin@CO.com", admins); got != "admin" {
		t.Fatalf("role = %q, want admin (case-insensitive)", got)
	}
}

func TestRequireAdmin_Allows(t *testing.T) {
	h := testHandler(t)
	admins := map[string]bool{"admin@co.com": true}
	guard := RequireAdmin(admins)

	sess := ServerSession{Login: "admin@co.com", Email: "admin@co.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guarded := h.Middleware(guard(inner))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/policies/import", nil)
	req.AddCookie(cookie)
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for admin", rec.Code)
	}
}

func TestRequireAdmin_Blocks(t *testing.T) {
	h := testHandler(t)
	admins := map[string]bool{"admin@co.com": true}
	guard := RequireAdmin(admins)

	sess := ServerSession{Login: "viewer@co.com", Email: "viewer@co.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guarded := h.Middleware(guard(inner))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/policies/import", nil)
	req.AddCookie(cookie)
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for viewer", rec.Code)
	}
}

func TestHandleMe_ReturnsRole(t *testing.T) {
	h := testHandler(t)
	admins := map[string]bool{"admin@co.com": true}
	h.SetAdmins(admins)

	sess := ServerSession{
		Login: "viewer@co.com", Email: "viewer@co.com",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	cookie := createSession(t, h, sess)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.AddCookie(cookie)
	h.handleMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var info UserInfo
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.Role != "viewer" {
		t.Errorf("role = %q, want viewer", info.Role)
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

func TestChatStore_PutGet(t *testing.T) {
	store := NewMemorySessionStore()
	chat := ChatSession{Messages: json.RawMessage(`[{"role":"user","text":"hello"}]`), TaskID: "t1"}
	if err := store.PutChat(nil, "a@b.com", chat); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetChat(nil, "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskID != "t1" {
		t.Fatalf("taskId = %q, want t1", got.TaskID)
	}
}

func TestChatStore_NotFound(t *testing.T) {
	store := NewMemorySessionStore()
	_, err := store.GetChat(nil, "missing@b.com")
	if err != ErrSessionNotFound {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestChatStore_Delete(t *testing.T) {
	store := NewMemorySessionStore()
	_ = store.PutChat(nil, "a@b.com", ChatSession{TaskID: "t1"})
	_ = store.DeleteChat(nil, "a@b.com")
	_, err := store.GetChat(nil, "a@b.com")
	if err != ErrSessionNotFound {
		t.Fatalf("err = %v, want ErrSessionNotFound after delete", err)
	}
}

func TestGetChatHistory_Empty(t *testing.T) {
	h := testHandler(t)
	sess := ServerSession{Login: "u@co.com", Email: "u@co.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	mux := http.NewServeMux()
	h.RegisterChatHistory(mux, h.store.(*MemorySessionStore))
	handler := h.Middleware(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/history", nil)
	req.AddCookie(cookie)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		Messages json.RawMessage
		TaskID   *string
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if string(resp.Messages) != "[]" {
		t.Fatalf("messages = %s, want []", resp.Messages)
	}
}

func TestPutGetChatHistory(t *testing.T) {
	h := testHandler(t)
	store := h.store.(*MemorySessionStore)
	sess := ServerSession{Login: "u@co.com", Email: "u@co.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	mux := http.NewServeMux()
	h.RegisterChatHistory(mux, store)
	handler := h.Middleware(mux)

	// PUT
	body := strings.NewReader(`{"messages":[{"role":"user","text":"hi"}],"taskId":"t123"}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/chat/history", body)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.AddCookie(cookie)
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d, want 204", putRec.Code)
	}

	// GET
	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/chat/history", nil)
	getReq.AddCookie(cookie)
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", getRec.Code)
	}
	var resp struct {
		Messages json.RawMessage
		TaskID   *string
	}
	json.NewDecoder(getRec.Body).Decode(&resp)
	if resp.TaskID == nil || *resp.TaskID != "t123" {
		t.Fatalf("taskId = %v, want t123", resp.TaskID)
	}
}
