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
	"net/url"
	"strings"
	"time"

	"github.com/complytime/complytime-studio/internal/httputil"
)

const (
	sessionCookieName = "studio_session"
	stateCookieName   = "studio_oauth_state"
	sessionMaxAge     = 8 * time.Hour

	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"
	googleUserURL  = "https://openidconnect.googleapis.com/v1/userinfo"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Config holds Google OAuth application credentials.
type Config struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

// Session represents the authenticated user session stored in the cookie.
type Session struct {
	AccessToken string `json:"t"`
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
	apiToken  string
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

// SetAPIToken configures a static bearer token that bypasses session auth.
// Intended for dev/CI seeding scripts. No-op if token is empty.
func (h *Handler) SetAPIToken(token string) {
	h.apiToken = token
}

// Register mounts auth endpoints on the mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/auth/login", h.handleLogin)
	mux.HandleFunc("/auth/callback", h.handleCallback)
	mux.HandleFunc("/auth/me", h.handleMe)
	mux.HandleFunc("/auth/logout", h.handleLogout)
}

// TokenFromRequest extracts the user's access token from the session cookie.
// Satisfies the httputil.TokenProvider interface.
func (h *Handler) TokenFromRequest(r *http.Request) (string, bool) {
	sess, err := h.sessionFromCookie(r)
	if err != nil {
		return "", false
	}
	if sess.AccessToken == "" {
		return "", false
	}
	return sess.AccessToken, true
}

// Middleware returns an http.Handler that requires a valid session cookie
// on all /api/* paths. It injects the Session into request context.
// When an API token is configured, requests with a matching
// Authorization: Bearer header bypass the session cookie check.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/config" {
			next.ServeHTTP(w, r)
			return
		}

		if h.apiToken != "" {
			if bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); bearer == h.apiToken {
				next.ServeHTTP(w, r)
				return
			}
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

	params := url.Values{
		"client_id":     {h.cfg.ClientID},
		"redirect_uri":  {h.cfg.CallbackURL},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"access_type":   {"online"},
	}
	http.Redirect(w, r, googleAuthURL+"?"+params.Encode(), http.StatusFound)
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

	user, err := fetchGoogleUser(r.Context(), token)
	if err != nil {
		slog.Error("oauth user fetch failed", "error", err)
		http.Error(w, "authentication failed — please try again", http.StatusBadGateway)
		return
	}

	sess := &Session{
		AccessToken: token,
		Login:       user.Email,
		Name:        user.Name,
		AvatarURL:   user.Picture,
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
	data := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {cfg.CallbackURL},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("google token endpoint %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("google: %s", result.Error)
	}
	return result.AccessToken, nil
}

type googleUser struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func fetchGoogleUser(ctx context.Context, token string) (*googleUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("google userinfo %d: %s", resp.StatusCode, string(b))
	}

	var user googleUser
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
