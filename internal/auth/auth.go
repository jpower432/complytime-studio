// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
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

	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/httputil"
)

const (
	sessionCookieName = "studio_session"
	stateCookieName   = "studio_oauth_state"
	sessionMaxAge     = 8 * time.Hour
)

var (
	httpClient = &http.Client{Timeout: 15 * time.Second}

	// ErrSessionNotFound indicates no session exists for the given ID.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired indicates the session has passed its expiration time.
	ErrSessionExpired = errors.New("session expired")
)

// Config holds OIDC application credentials and behaviour settings.
type Config struct {
	ClientID        string
	ClientSecret    string
	CallbackURL     string
	Provider        *OIDCProvider
	Scopes          string
	RolesClaim      string
	BootstrapEmails []string
}

// Session represents the authenticated user session injected into request
// context. It does not contain the access token — that stays server-side.
type Session struct {
	Login          string   `json:"l"`
	Name           string   `json:"n"`
	AvatarURL      string   `json:"a"`
	Email          string   `json:"e"`
	Groups         []string `json:"g,omitempty"`
	ExpiresAt      int64    `json:"x"`
	ServiceAccount bool     `json:"-"`
}

// cookiePayload is the encrypted cookie content — only a session ID.
type cookiePayload struct {
	SessionID string `json:"sid"`
}

// stateCookiePayload carries the OAuth state nonce and PKCE code_verifier.
type stateCookiePayload struct {
	State        string `json:"s"`
	CodeVerifier string `json:"cv"`
}

// UserInfo is the public-facing user info returned by /auth/me.
type UserInfo struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

// RequireAdmin returns middleware that rejects non-admin requests with 403.
// Fails closed: if the store lookup errors, the request is rejected.
func RequireAdmin(users UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := SessionFrom(r.Context())
			if !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
				return
			}
			if sess.ServiceAccount {
				next.ServeHTTP(w, r)
				return
			}
			if users == nil {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
				return
			}
			u, err := users.GetUser(r.Context(), sess.Email)
			if err != nil || u.Role != consts.RoleAdmin {
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
	users     UserStore
	jwks      *JWKSCache
}

// NewHandler creates auth handlers. The secretKey must be exactly 32 bytes
// (AES-256). Session cookies are encrypted with AES-GCM.
func NewHandler(cfg Config, secretKey []byte, store SessionStore) (*Handler, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, fmt.Errorf("auth: invalid secret key: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: GCM init: %w", err)
	}
	h := &Handler{cfg: cfg, secretKey: secretKey, gcm: gcm, store: store}
	if cfg.Provider != nil {
		h.jwks = newJWKSCache(cfg.Provider.JWKSURL, time.Hour)
	}
	return h, nil
}

// SetUserStore configures the persistent user/role store.
func (h *Handler) SetUserStore(us UserStore) {
	h.users = us
}

// SetAPIToken configures a static bearer token that bypasses session auth.
func (h *Handler) SetAPIToken(token string) {
	h.apiToken = token
}

// UpdateProvider hot-swaps the OIDC provider after a discovery refresh.
// Called by the periodic refresh goroutine in main.go.
func (h *Handler) UpdateProvider(p *OIDCProvider) {
	h.cfg.Provider = p
	if h.jwks != nil {
		h.jwks.jwksURL = p.JWKSURL
	}
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

// TokenFromRequest looks up the server-side session and returns the access token.
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
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/config" {
			next.ServeHTTP(w, r)
			return
		}

		if h.apiToken != "" {
			if bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); bearer == h.apiToken {
				sess := &Session{
					Email: "api-token@internal",
					Name:  "API Token",
				}
				if h.apiToken == consts.DefaultDevAPIToken {
					sess.ServiceAccount = true
				}
				ctx := context.WithValue(r.Context(), sessionKey, sess)
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
	if h.cfg.Provider == nil {
		http.Error(w, "OIDC provider not configured", http.StatusServiceUnavailable)
		authLoginTotal.Add("error", 1)
		return
	}

	state := generateState()
	codeVerifier := generateCodeVerifier()

	payload, err := json.Marshal(stateCookiePayload{State: state, CodeVerifier: codeVerifier})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		authLoginTotal.Add("error", 1)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(payload),
		Path:     "/auth",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecureRequest(r),
	})

	scopes := h.cfg.Scopes
	if scopes == "" {
		scopes = "openid email profile"
	}

	params := url.Values{
		"client_id":             {h.cfg.ClientID},
		"redirect_uri":          {h.cfg.CallbackURL},
		"response_type":         {"code"},
		"scope":                 {scopes},
		"state":                 {state},
		"code_challenge":        {pkceChallenge(codeVerifier)},
		"code_challenge_method": {"S256"},
	}
	authLoginTotal.Add("success", 1)
	http.Redirect(w, r, h.cfg.Provider.AuthURL+"?"+params.Encode(), http.StatusFound)
}

func (h *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		http.Error(w, "invalid state parameter", http.StatusForbidden)
		authCallbackTotal.Add("invalid_state", 1)
		return
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(stateCookie.Value)
	if err != nil {
		http.Error(w, "invalid state parameter", http.StatusForbidden)
		authCallbackTotal.Add("invalid_state", 1)
		return
	}
	var statePayload stateCookiePayload
	if err := json.Unmarshal(rawPayload, &statePayload); err != nil || statePayload.State != r.URL.Query().Get("state") {
		http.Error(w, "invalid state parameter", http.StatusForbidden)
		authCallbackTotal.Add("invalid_state", 1)
		return
	}
	clearCookie(w, stateCookieName, "/auth", isSecureRequest(r))

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		authCallbackTotal.Add("token_error", 1)
		return
	}

	if h.cfg.Provider == nil || h.jwks == nil {
		http.Error(w, "OIDC provider not configured", http.StatusServiceUnavailable)
		authCallbackTotal.Add("token_error", 1)
		return
	}

	tokenResp, err := exchangeCode(r.Context(), h.cfg, code, statePayload.CodeVerifier)
	if err != nil {
		slog.Error("oidc token exchange failed", "error", err)
		http.Error(w, "authentication failed — please try again", http.StatusBadGateway)
		authCallbackTotal.Add("token_error", 1)
		return
	}

	// Cryptographic verification of the ID token.
	claims, err := h.jwks.VerifyIDToken(r.Context(), tokenResp.IDToken, h.cfg.Provider.IssuerURL, h.cfg.ClientID)
	if err != nil {
		slog.Error("oidc id token verification failed", "error", err)
		http.Error(w, "authentication failed — please try again", http.StatusBadGateway)
		authCallbackTotal.Add("verify_error", 1)
		return
	}

	// When the discovery document omits userinfo_endpoint, fall back to the
	// verified ID token claims rather than failing the login.
	var user *oidcUser
	if h.cfg.Provider.UserInfoURL == "" {
		slog.Warn("oidc: userinfo_endpoint absent — building profile from ID token claims")
		user = &oidcUser{
			Sub:     claims.Subject,
			Email:   claims.Email,
			Name:    claims.Name,
			Picture: claims.Picture,
		}
	} else {
		user, err = fetchUserInfo(r.Context(), h.cfg.Provider.UserInfoURL, tokenResp.AccessToken)
		if err != nil {
			slog.Error("oidc userinfo fetch failed", "error", err)
			http.Error(w, "authentication failed — please try again", http.StatusBadGateway)
			authCallbackTotal.Add("userinfo_error", 1)
			return
		}
	}

	if h.users != nil {
		h.seedUserRole(r.Context(), claims, user)
	}

	sid := generateSessionID()
	serverSess := ServerSession{
		AccessToken: tokenResp.AccessToken,
		Login:       user.Email,
		Name:        user.Name,
		AvatarURL:   user.Picture,
		Email:       user.Email,
		Groups:      claims.Groups,
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

	authCallbackTotal.Add("success", 1)
	http.Redirect(w, r, "/", http.StatusFound)
}

// seedUserRole implements the role seeding algorithm from the design spec.
// Existing users keep their DB role; new users are seeded from JWT claims,
// bootstrap allowlist, or first-admin promotion.
func (h *Handler) seedUserRole(ctx context.Context, claims *IDTokenClaims, user *oidcUser) {
	// Check existence BEFORE upsert so we can distinguish new vs. returning users.
	_, lookupErr := h.users.GetUserBySub(ctx, claims.Subject, h.cfg.Provider.IssuerURL)
	switch {
	case lookupErr == nil:
		// Returning user — DB role is authoritative, skip all mutation.
		if err := h.users.UpsertUser(ctx, claims.Subject, h.cfg.Provider.IssuerURL, user.Email, user.Name, user.Picture); err != nil {
			slog.Error("user upsert failed (login continues)", "error", err)
		}
		return
	case errors.Is(lookupErr, ErrUserNotFound):
		// New user — fall through to seeding logic.
	default:
		// Genuine store error: we cannot safely determine new vs. returning, so we
		// skip role seeding entirely to avoid silently bypassing returning-user checks.
		// The session is still created; the user will appear as reviewer until the
		// store recovers and they log in again.
		slog.Error("GetUserBySub store error — skipping role seeding (login continues as reviewer)", "error", lookupErr)
		if err := h.users.UpsertUser(ctx, claims.Subject, h.cfg.Provider.IssuerURL, user.Email, user.Name, user.Picture); err != nil {
			slog.Error("user upsert failed (login continues)", "error", err)
		}
		return
	}

	if err := h.users.UpsertUser(ctx, claims.Subject, h.cfg.Provider.IssuerURL, user.Email, user.Name, user.Picture); err != nil {
		slog.Error("user upsert failed (login continues)", "error", err)
	}

	// New user: determine role.

	// Bootstrap allowlist gate: when configured, only listed emails can become admin.
	if len(h.cfg.BootstrapEmails) > 0 && !contains(h.cfg.BootstrapEmails, user.Email) {
		slog.Info("new user not in bootstrap allowlist — assigning reviewer", "email", user.Email)
		return
	}

	// JWT seed: verified ID token carries an admin role claim.
	jwtRoles := claims.ExtractRolesClaim(h.cfg.RolesClaim)
	if claims.EmailVerified && containsStr(jwtRoles, consts.RoleAdmin) {
		if _, err := h.users.SetRole(ctx, user.Email, consts.RoleAdmin); err != nil {
			slog.Error("jwt role seed failed", "email", user.Email, "error", err)
		} else {
			slog.Info("new user seeded as admin from JWT claim", "email", user.Email)
			_ = h.users.InsertRoleChange(ctx, RoleChange{
				ChangedBy: "jwt-seed", TargetEmail: user.Email,
				OldRole: consts.RoleReviewer, NewRole: consts.RoleAdmin,
			})
		}
		return
	}

	// email_verified gate: unverified email cannot become admin.
	if !claims.EmailVerified {
		slog.Warn("new user email not verified — skipping admin promotion", "email", user.Email)
		return
	}

	// First-admin promotion: atomic INSERT-if-zero-admins.
	adminCount, adminErr := h.users.CountAdmins(ctx)
	if adminErr != nil {
		slog.Error("admin count check failed (login continues)", "error", adminErr)
		return
	}
	if adminCount == 0 {
		if _, err := h.users.SetRole(ctx, user.Email, consts.RoleAdmin); err != nil {
			slog.Error("first-admin promotion failed", "email", user.Email, "error", err)
		} else {
			slog.Info("first admin promoted", "email", user.Email)
			_ = h.users.InsertRoleChange(ctx, RoleChange{
				ChangedBy: "first-admin", TargetEmail: user.Email,
				OldRole: consts.RoleReviewer, NewRole: consts.RoleAdmin,
			})
		}
	}
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
	role := consts.RoleReviewer
	if h.users != nil {
		if u, err := h.users.GetUser(r.Context(), serverSess.Email); err == nil {
			role = u.Role
		}
	}
	writeJSON(w, http.StatusOK, UserInfo{
		Login:     serverSess.Login,
		Name:      serverSess.Name,
		AvatarURL: serverSess.AvatarURL,
		Email:     serverSess.Email,
		Role:      role,
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

func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// generateCodeVerifier returns a high-entropy PKCE code_verifier (RFC 7636).
func generateCodeVerifier() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// pkceChallenge computes the S256 code_challenge for a given verifier.
func pkceChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// tokenResponse holds the result of an OIDC token exchange.
type tokenResponse struct {
	AccessToken string
	IDToken     string
}

func exchangeCode(ctx context.Context, cfg Config, code, codeVerifier string) (*tokenResponse, error) {
	data := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {cfg.CallbackURL},
		"code_verifier": {codeVerifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Provider.TokenURL, strings.NewReader(data.Encode()))
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
		return nil, fmt.Errorf("token endpoint %d: %s", resp.StatusCode, string(b))
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
		return nil, fmt.Errorf("token error: %s", result.Error)
	}
	return &tokenResponse{AccessToken: result.AccessToken, IDToken: result.IDToken}, nil
}

// oidcUser holds the userinfo response fields.
type oidcUser struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func fetchUserInfo(ctx context.Context, userInfoURL, token string) (*oidcUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
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
		return nil, fmt.Errorf("userinfo %d: %s", resp.StatusCode, string(b))
	}
	var user oidcUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
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

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// containsStr is an alias for contains for readability at call sites.
func containsStr(slice []string, s string) bool {
	return contains(slice, s)
}
