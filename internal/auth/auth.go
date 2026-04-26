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
	"errors"
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

var (
	httpClient = &http.Client{Timeout: 15 * time.Second}

	// ErrSessionNotFound indicates no session exists for the given ID.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired indicates the session has passed its expiration time.
	ErrSessionExpired = errors.New("session expired")
)

// Config holds Google OAuth application credentials.
type Config struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

// Session represents the authenticated user session injected into request
// context. It does not contain the access token — that stays server-side.
type Session struct {
	Login     string   `json:"l"`
	Name      string   `json:"n"`
	AvatarURL string   `json:"a"`
	Email     string   `json:"e"`
	Groups    []string `json:"g,omitempty"`
	ExpiresAt int64    `json:"x"`
}

// cookiePayload is the encrypted cookie content — only a session ID.
type cookiePayload struct {
	SessionID string `json:"sid"`
}

// UserInfo is the public-facing user info returned by /auth/me.
type UserInfo struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

// RoleForEmail returns "admin" if the email is in the admin set, "viewer" otherwise.
// An empty admin set means everyone is admin (fail-open for dev clusters).
func RoleForEmail(email string, admins map[string]bool) string {
	if len(admins) == 0 {
		return "admin"
	}
	if admins[strings.ToLower(email)] {
		return "admin"
	}
	return "viewer"
}

// RequireAdmin returns middleware that rejects non-admin requests with 403.
func RequireAdmin(admins map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := SessionFrom(r.Context())
			if !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
				return
			}
			if RoleForEmail(sess.Email, admins) != "admin" {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
	store     SessionStore
	admins    map[string]bool
}

// NewHandler creates auth handlers. The secretKey must be exactly 32 bytes
// (AES-256). Session cookies are encrypted with AES-GCM. The cookie now
// carries only a session ID; tokens are stored in the SessionStore.
func NewHandler(cfg Config, secretKey []byte, store SessionStore) (*Handler, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, fmt.Errorf("auth: invalid secret key: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: GCM init: %w", err)
	}
	return &Handler{cfg: cfg, secretKey: secretKey, gcm: gcm, store: store, admins: make(map[string]bool)}, nil
}

// SetAdmins configures the admin email allowlist. An empty map means
// everyone is admin (dev-mode fail-open).
func (h *Handler) SetAdmins(admins map[string]bool) {
	h.admins = admins
}

// Admins returns the admin email set for use with RequireAdmin middleware.
func (h *Handler) Admins() map[string]bool {
	return h.admins
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

// RegisterChatHistory mounts GET/PUT /api/chat/history for server-side chat persistence.
func (h *Handler) RegisterChatHistory(mux *http.ServeMux, chatStore ChatStore) {
	mux.HandleFunc("GET /api/chat/history", h.handleGetChatHistory(chatStore))
	mux.HandleFunc("PUT /api/chat/history", h.handlePutChatHistory(chatStore))
}

func (h *Handler) handleGetChatHistory(cs ChatStore) http.HandlerFunc {
	type chatResp struct {
		Messages json.RawMessage `json:"messages"`
		TaskID   *string         `json:"taskId"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		sess, ok := SessionFrom(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		chat, err := cs.GetChat(r.Context(), sess.Email)
		if err != nil {
			writeJSON(w, http.StatusOK, chatResp{Messages: json.RawMessage("[]"), TaskID: nil})
			return
		}
		tid := &chat.TaskID
		if chat.TaskID == "" {
			tid = nil
		}
		writeJSON(w, http.StatusOK, chatResp{Messages: chat.Messages, TaskID: tid})
	}
}

func (h *Handler) handlePutChatHistory(cs ChatStore) http.HandlerFunc {
	type chatReq struct {
		Messages json.RawMessage `json:"messages"`
		TaskID   string          `json:"taskId"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		sess, ok := SessionFrom(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		var req chatReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := cs.PutChat(r.Context(), sess.Email, ChatSession{Messages: req.Messages, TaskID: req.TaskID}); err != nil {
			http.Error(w, "store failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// TokenFromRequest looks up the server-side session and returns the access
// token. Satisfies the httputil.TokenProvider interface.
func (h *Handler) TokenFromRequest(r *http.Request) (string, bool) {
	sid, err := h.sessionIDFromCookie(r)
	if err != nil {
		return "", false
	}
	sess, err := h.store.Get(r.Context(), sid)
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
				ctx := context.WithValue(r.Context(), sessionKey, &Session{
					Email: "api-token@internal",
					Name:  "API Token",
				})
				ctx = httputil.WithIdentity(ctx, "api-token@internal")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		sid, err := h.sessionIDFromCookie(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		serverSess, err := h.store.Get(r.Context(), sid)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		sess := &Session{
			Login:     serverSess.Login,
			Name:      serverSess.Name,
			AvatarURL: serverSess.AvatarURL,
			Email:     serverSess.Email,
			Groups:    serverSess.Groups,
			ExpiresAt: serverSess.ExpiresAt,
		}
		ctx := context.WithValue(r.Context(), sessionKey, sess)
		if id := sess.Email; id != "" {
			ctx = httputil.WithIdentity(ctx, id)
		} else if id := sess.Login; id != "" {
			ctx = httputil.WithIdentity(ctx, id)
		}
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

	tokenResp, err := exchangeCode(r.Context(), h.cfg, code)
	if err != nil {
		slog.Error("oauth token exchange failed", "error", err)
		http.Error(w, "authentication failed — please try again", http.StatusBadGateway)
		return
	}

	user, err := fetchGoogleUser(r.Context(), tokenResp.AccessToken)
	if err != nil {
		slog.Error("oauth user fetch failed", "error", err)
		http.Error(w, "authentication failed — please try again", http.StatusBadGateway)
		return
	}

	groups := groupsFromIDToken(tokenResp.IDToken)

	sid := generateSessionID()
	serverSess := ServerSession{
		AccessToken: tokenResp.AccessToken,
		Login:       user.Email,
		Name:        user.Name,
		AvatarURL:   user.Picture,
		Email:       user.Email,
		Groups:      groups,
		ExpiresAt:   time.Now().Add(sessionMaxAge).Unix(),
	}
	if err := h.store.Put(r.Context(), sid, serverSess); err != nil {
		slog.Error("session store put failed", "error", err)
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	if err := h.setSessionCookie(w, r, sid); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	sid, err := h.sessionIDFromCookie(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	serverSess, err := h.store.Get(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	writeJSON(w, http.StatusOK, UserInfo{
		Login:     serverSess.Login,
		Name:      serverSess.Name,
		AvatarURL: serverSess.AvatarURL,
		Email:     serverSess.Email,
		Role:      RoleForEmail(serverSess.Email, h.admins),
	})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if sid, err := h.sessionIDFromCookie(r); err == nil {
		_ = h.store.Delete(r.Context(), sid)
	}
	clearCookie(w, sessionCookieName, "/", isSecureRequest(r))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, r *http.Request, sid string) error {
	plaintext, err := json.Marshal(cookiePayload{SessionID: sid})
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

// sessionIDFromCookie decrypts the cookie and returns the session ID.
func (h *Handler) sessionIDFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", fmt.Errorf("decode cookie: %w", err)
	}

	nonceSize := h.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("malformed session cookie")
	}
	nonce, sealed := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := h.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt session: %w", err)
	}

	var payload cookiePayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return "", err
	}
	if payload.SessionID == "" {
		return "", fmt.Errorf("empty session ID in cookie")
	}
	return payload.SessionID, nil
}

func generateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// tokenResponse holds the result of a Google OAuth token exchange.
type tokenResponse struct {
	AccessToken string
	IDToken     string
}

func exchangeCode(ctx context.Context, cfg Config, code string) (*tokenResponse, error) {
	data := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {cfg.CallbackURL},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("google token endpoint %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("google: %s", result.Error)
	}
	return &tokenResponse{AccessToken: result.AccessToken, IDToken: result.IDToken}, nil
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
	defer func() { _ = resp.Body.Close() }()

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

// groupsFromIDToken extracts the "groups" claim from a Google OIDC ID token.
// The ID token is a JWT; we decode the payload without signature verification
// because the token was just received from Google's token endpoint over TLS.
// Returns nil if the claim is absent or unparseable.
func groupsFromIDToken(idToken string) []string {
	if idToken == "" {
		return nil
	}
	parts := strings.SplitN(idToken, ".", 3)
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims struct {
		Groups []string `json:"groups"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims.Groups
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
