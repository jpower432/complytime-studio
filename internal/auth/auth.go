// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookieName = "studio_session"
	stateCookieName   = "studio_oauth_state"
	sessionMaxAge     = 8 * time.Hour
)

// Config holds GitHub OAuth application credentials.
type Config struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

// Session represents the authenticated user session stored in the cookie.
type Session struct {
	GitHubToken string `json:"t"`
	Login       string `json:"l"`
	Name        string `json:"n"`
	AvatarURL   string `json:"a"`
	Email       string `json:"e"`
	ExpiresAt   int64  `json:"x"`
}

// UserInfo is the public-facing user info returned by /auth/me.
type UserInfo struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

type contextKey string

const sessionKey contextKey = "session"

// SessionFrom extracts the Session from request context.
func SessionFrom(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey).(*Session)
	return s, ok
}

// Handler manages OAuth login, callback, and session endpoints.
type Handler struct {
	cfg       Config
	signKey   []byte
	mux       *http.ServeMux
}

// NewHandler creates auth handlers and registers them on the provided mux.
func NewHandler(cfg Config, signKey []byte) *Handler {
	h := &Handler{cfg: cfg, signKey: signKey}
	return h
}

// Register mounts auth endpoints on the mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/auth/login", h.handleLogin)
	mux.HandleFunc("/auth/callback", h.handleCallback)
	mux.HandleFunc("/auth/me", h.handleMe)
	mux.HandleFunc("/auth/logout", h.handleLogout)
}

// Middleware returns an http.Handler that requires a valid session cookie
// on all /api/* paths. It injects the Session into request context.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		sess, err := h.sessionFromCookie(r)
		if err != nil || time.Now().Unix() > sess.ExpiresAt {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		ctx := context.WithValue(r.Context(), sessionKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/auth",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		h.cfg.ClientID, h.cfg.CallbackURL, "read:user,user:email,repo", state,
	)
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state parameter", http.StatusForbidden)
		return
	}
	clearCookie(w, stateCookieName, "/auth", r.TLS != nil)

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := exchangeCode(r.Context(), h.cfg, code)
	if err != nil {
		http.Error(w, "token exchange failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	user, err := fetchGitHubUser(r.Context(), token)
	if err != nil {
		http.Error(w, "user fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	sess := &Session{
		GitHubToken: token,
		Login:       user.Login,
		Name:        user.Name,
		AvatarURL:   user.AvatarURL,
		Email:       user.Email,
		ExpiresAt:   time.Now().Add(sessionMaxAge).Unix(),
	}

	if err := h.setSessionCookie(w, r, sess); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	sess, err := h.sessionFromCookie(r)
	if err != nil || time.Now().Unix() > sess.ExpiresAt {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	writeJSON(w, http.StatusOK, UserInfo{
		Login:     sess.Login,
		Name:      sess.Name,
		AvatarURL: sess.AvatarURL,
		Email:     sess.Email,
	})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, sessionCookieName, "/", r.TLS != nil)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, r *http.Request, sess *Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	payload := base64.RawURLEncoding.EncodeToString(data)
	sig := h.sign(payload)
	value := payload + "." + sig

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(sessionMaxAge.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
	return nil
}

func (h *Handler) sessionFromCookie(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed session cookie")
	}
	payload, sig := parts[0], parts[1]

	if !hmac.Equal([]byte(h.sign(payload)), []byte(sig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (h *Handler) sign(payload string) string {
	mac := hmac.New(sha256.New, h.signKey)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func exchangeCode(ctx context.Context, cfg Config, code string) (string, error) {
	body := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s", cfg.ClientID, cfg.ClientSecret, code)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("github: %s", result.Error)
	}
	return result.AccessToken, nil
}

type ghUser struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

func fetchGitHubUser(ctx context.Context, token string) (*ghUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github API %d: %s", resp.StatusCode, string(b))
	}

	var user ghUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func clearCookie(w http.ResponseWriter, name, path string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
