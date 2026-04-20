// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/complytime/complytime-studio/internal/httputil"
)

const (
	sessionCookieName = "studio_session"
	stateCookieName   = "studio_oauth_state"
	sessionMaxAge     = 8 * time.Hour
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

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
	secretKey []byte
	gcm       cipher.AEAD
}

// NewHandler creates auth handlers. The secretKey must be exactly 32 bytes
// (AES-256). Session cookies are encrypted with AES-GCM.
func NewHandler(cfg Config, secretKey []byte) (*Handler, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, fmt.Errorf("auth: invalid secret key: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: GCM init: %w", err)
	}
	return &Handler{cfg: cfg, secretKey: secretKey, gcm: gcm}, nil
}

// Register mounts auth endpoints on the mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/auth/login", h.handleLogin)
	mux.HandleFunc("/auth/callback", h.handleCallback)
	mux.HandleFunc("/auth/me", h.handleMe)
	mux.HandleFunc("/auth/logout", h.handleLogout)
}

// TokenFromRequest extracts the user's GitHub token from the session cookie.
// It satisfies the httputil.TokenProvider interface.
func (h *Handler) TokenFromRequest(r *http.Request) (string, bool) {
	sess, err := h.sessionFromCookie(r)
	if err != nil {
		return "", false
	}
	if sess.GitHubToken == "" {
		return "", false
	}
	return sess.GitHubToken, true
}

// Middleware returns an http.Handler that requires a valid session cookie
// on all /api/* paths. It injects the Session into request context.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/config" {
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
		Secure:   isSecureRequest(r),
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
	clearCookie(w, stateCookieName, "/auth", isSecureRequest(r))

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := exchangeCode(r.Context(), h.cfg, code)
	if err != nil {
		slog.Error("oauth token exchange failed", "error", err)
		http.Error(w, "authentication failed — please try again", http.StatusBadGateway)
		return
	}

	user, err := fetchGitHubUser(r.Context(), token)
	if err != nil {
		slog.Error("oauth user fetch failed", "error", err)
		http.Error(w, "authentication failed — please try again", http.StatusBadGateway)
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
	clearCookie(w, sessionCookieName, "/", isSecureRequest(r))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, r *http.Request, sess *Session) error {
	plaintext, err := json.Marshal(sess)
	if err != nil {
		return err
	}

	nonce := make([]byte, h.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := h.gcm.Seal(nonce, nonce, plaintext, nil)
	value := base64.RawURLEncoding.EncodeToString(ciphertext)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(sessionMaxAge.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecureRequest(r),
	})
	return nil
}

func (h *Handler) sessionFromCookie(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("decode cookie: %w", err)
	}

	nonceSize := h.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("malformed session cookie")
	}
	nonce, sealed := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := h.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(plaintext, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func exchangeCode(ctx context.Context, cfg Config, code string) (string, error) {
	body := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s", cfg.ClientID, cfg.ClientSecret, code)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("github token endpoint %d: %s", resp.StatusCode, string(b))
	}

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

	resp, err := httpClient.Do(req)
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
	httputil.WriteJSON(w, status, v)
}

// isSecureRequest returns true when the original client connection used TLS,
// honoring X-Forwarded-Proto set by reverse proxies that terminate TLS.
func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
