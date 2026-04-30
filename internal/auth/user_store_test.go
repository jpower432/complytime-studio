// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/complytime/complytime-studio/internal/consts"
)

type memoryUserStore struct {
	mu      sync.RWMutex
	users   map[string]*User
	changes []RoleChange
}

func newMemoryUserStore() *memoryUserStore {
	return &memoryUserStore{users: make(map[string]*User)}
}

func (m *memoryUserStore) UpsertUser(_ context.Context, sub, issuer, email, name, avatarURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[email]; ok {
		u.Sub = sub
		u.Issuer = issuer
		u.Name = name
		u.AvatarURL = avatarURL
		return nil
	}
	m.users[email] = &User{
		Sub: sub, Issuer: issuer,
		Email: email, Name: name, AvatarURL: avatarURL,
		Role: consts.RoleReviewer, CreatedAt: time.Now(),
	}
	return nil
}

func (m *memoryUserStore) GetUserBySub(_ context.Context, sub, issuer string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Sub == sub && u.Issuer == issuer {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *memoryUserStore) GetUser(_ context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *memoryUserStore) ListUsers(_ context.Context) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []User
	for _, u := range m.users {
		out = append(out, *u)
	}
	return out, nil
}

func (m *memoryUserStore) SetRole(_ context.Context, email, role string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[email]
	if !ok {
		return "", ErrUserNotFound
	}
	old := u.Role
	u.Role = role
	return old, nil
}

func (m *memoryUserStore) CountUsers(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users), nil
}

func (m *memoryUserStore) CountAdmins(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, u := range m.users {
		if u.Role == consts.RoleAdmin {
			count++
		}
	}
	return count, nil
}

func (m *memoryUserStore) InsertRoleChange(_ context.Context, change RoleChange) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	change.ChangedAt = time.Now()
	m.changes = append(m.changes, change)
	return nil
}

func (m *memoryUserStore) ListRoleChanges(_ context.Context) ([]RoleChange, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.changes, nil
}

func TestRequireAdmin_WithUserStore(t *testing.T) {
	h := testHandler(t)
	us := newMemoryUserStore()
	_ = us.UpsertUser(context.TODO(), "sub-admin", "https://issuer", "admin@co.com", "Admin", "")
	_, _ = us.SetRole(context.TODO(), "admin@co.com", consts.RoleAdmin)
	_ = us.UpsertUser(context.TODO(), "sub-viewer", "https://issuer", "viewer@co.com", "Viewer", "")

	guard := RequireAdmin(us)

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("admin passes", func(t *testing.T) {
		sess := ServerSession{Email: "admin@co.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
		cookie := createSession(t, h, sess)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/policies/import", nil)
		req.AddCookie(cookie)
		h.Middleware(guard(inner)).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("reviewer blocked", func(t *testing.T) {
		sess := ServerSession{Email: "viewer@co.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
		cookie := createSession(t, h, sess)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/policies/import", nil)
		req.AddCookie(cookie)
		h.Middleware(guard(inner)).ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})
}

func TestHandleMe_WithUserStore(t *testing.T) {
	h := testHandler(t)
	us := newMemoryUserStore()
	h.SetUserStore(us)

	_ = us.UpsertUser(context.TODO(), "sub-test", "https://issuer", "test@co.com", "Test", "")
	_, _ = us.SetRole(context.TODO(), "test@co.com", consts.RoleAdmin)

	sess := ServerSession{
		Login: "test@co.com", Email: "test@co.com", Name: "Test",
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
	_ = json.NewDecoder(rec.Body).Decode(&info)
	if info.Role != consts.RoleAdmin {
		t.Fatalf("role = %q, want admin (from user store)", info.Role)
	}
}

func TestHandleSetRole(t *testing.T) {
	h := testHandler(t)
	us := newMemoryUserStore()
	h.SetUserStore(us)

	_ = us.UpsertUser(context.TODO(), "sub-admin", "https://issuer", "admin@co.com", "Admin", "")
	_, _ = us.SetRole(context.TODO(), "admin@co.com", consts.RoleAdmin)
	_ = us.UpsertUser(context.TODO(), "sub-target", "https://issuer", "target@co.com", "Target", "")

	sess := ServerSession{Email: "admin@co.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cookie := createSession(t, h, sess)

	mux := http.NewServeMux()
	h.RegisterUserAPI(mux)

	body := strings.NewReader(`{"role":"` + consts.RoleAdmin + `"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/users/target@co.com/role", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	ctx := context.WithValue(req.Context(), sessionKey, &Session{Email: "admin@co.com"})
	mux.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	u, _ := us.GetUser(context.TODO(), "target@co.com")
	if u.Role != consts.RoleAdmin {
		t.Fatalf("role = %q, want admin", u.Role)
	}

	changes, _ := us.ListRoleChanges(context.TODO())
	if len(changes) != 1 {
		t.Fatalf("changes = %d, want 1", len(changes))
	}
	if changes[0].OldRole != consts.RoleReviewer || changes[0].NewRole != consts.RoleAdmin {
		t.Fatalf("change = %s -> %s, want reviewer -> admin", changes[0].OldRole, changes[0].NewRole)
	}
}

func TestHandleSetRole_InvalidRole(t *testing.T) {
	h := testHandler(t)
	us := newMemoryUserStore()
	h.SetUserStore(us)

	_ = us.UpsertUser(context.TODO(), "sub-admin", "https://issuer", "admin@co.com", "Admin", "")
	_, _ = us.SetRole(context.TODO(), "admin@co.com", consts.RoleAdmin)

	mux := http.NewServeMux()
	h.RegisterUserAPI(mux)

	body := strings.NewReader(`{"role":"superadmin"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/users/someone@co.com/role", body)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), sessionKey, &Session{Email: "admin@co.com"})
	mux.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleListUsers_AdminRequired(t *testing.T) {
	h := testHandler(t)
	us := newMemoryUserStore()
	h.SetUserStore(us)

	_ = us.UpsertUser(context.TODO(), "sub-admin", "https://issuer", "admin@co.com", "Admin", "")
	_, _ = us.SetRole(context.TODO(), "admin@co.com", consts.RoleAdmin)
	_ = us.UpsertUser(context.TODO(), "sub-viewer", "https://issuer", "viewer@co.com", "Viewer", "")

	mux := http.NewServeMux()
	h.RegisterUserAPI(mux)

	t.Run("admin can list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		ctx := context.WithValue(req.Context(), sessionKey, &Session{Email: "admin@co.com"})
		mux.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var users []User
		_ = json.NewDecoder(rec.Body).Decode(&users)
		if len(users) != 2 {
			t.Fatalf("users = %d, want 2", len(users))
		}
	})

	t.Run("reviewer blocked", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		ctx := context.WithValue(req.Context(), sessionKey, &Session{Email: "viewer@co.com"})
		mux.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("unauthenticated blocked", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}
