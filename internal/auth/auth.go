// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/httputil"
	"github.com/labstack/echo/v4"
)

// Session represents the authenticated user identity injected into request
// context. Populated from X-Forwarded-* headers set by OAuth2 Proxy.
type Session struct {
	Login          string   `json:"l"`
	Name           string   `json:"n"`
	AvatarURL      string   `json:"a"`
	Email          string   `json:"e"`
	Groups         []string `json:"g,omitempty"`
	ServiceAccount bool     `json:"-"`
}

// UserInfo is the public-facing user info returned by /auth/me.
type UserInfo struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

type contextKey string

const sessionKey contextKey = "session"

// SessionFrom extracts the Session from request context.
func SessionFrom(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey).(*Session)
	return s, ok
}

// Handler provides auth middleware and user management endpoints. Identity
// is established by OAuth2 Proxy via X-Forwarded-* headers. The handler
// trusts these headers, upserts users on first-seen, and enforces RBAC.
type Handler struct {
	apiToken string
	users    UserStore
}

// NewHandler creates an auth handler. OAuth2 Proxy handles OIDC externally;
// the handler only reads proxy-injected headers and manages the user store.
func NewHandler(apiToken string) *Handler {
	return &Handler{apiToken: apiToken}
}

// StripUntrustedProxyHeaders returns middleware that removes X-Forwarded-*
// identity headers from requests not originating from the trusted OAuth2
// Proxy sidecar. This prevents header spoofing when the gateway is
// accidentally exposed without the proxy in front.
//
// When proxySecret is non-empty, requests must carry a matching
// X-Proxy-Secret header or have their identity headers stripped.
// When proxySecret is empty, this middleware is a no-op (dev mode).
func StripUntrustedProxyHeaders(proxySecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if proxySecret == "" {
				return next(c)
			}
			r := c.Request()
			if r.Header.Get("X-Proxy-Secret") != proxySecret {
				r.Header.Del("X-Forwarded-Email")
				r.Header.Del("X-Forwarded-User")
				r.Header.Del("X-Forwarded-Preferred-Username")
				r.Header.Del("X-Forwarded-Groups")
				r.Header.Del("X-Forwarded-Access-Token")
				r.Header.Del("X-Proxy-Secret")
			} else {
				r.Header.Del("X-Proxy-Secret")
			}
			return next(c)
		}
	}
}

// SetUserStore configures the persistent user/role store.
func (h *Handler) SetUserStore(us UserStore) {
	h.users = us
}

// Register mounts auth endpoints. Login, callback, and logout are handled by
// OAuth2 Proxy at /oauth2/*. The /auth/logged-out page is excluded from proxy
// auth (via --skip-auth-route) so it renders after session cookie is cleared.
func (h *Handler) Register(e *echo.Echo) {
	e.GET("/auth/me", h.handleMeEcho)
	e.GET("/auth/logged-out", handleLoggedOut)
}

func handleLoggedOut(c echo.Context) error {
	return c.HTML(http.StatusOK, `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Signed Out</title>
<style>body{font-family:system-ui;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#1a1a2e;color:#e0e0e0}
.card{text-align:center}a{color:#6c9fff;text-decoration:none;font-weight:600}</style></head>
<body><div class="card"><h2>You have been signed out.</h2><p><a href="/">Sign in again</a></p></div></body></html>`)
}

// RegisterChatHistory mounts GET/PUT /api/chat/history for server-side chat persistence.
func (h *Handler) RegisterChatHistory(g *echo.Group, chatStore ChatStore) {
	g.GET("/chat/history", h.handleGetChatHistory(chatStore))
	g.PUT("/chat/history", h.handlePutChatHistory(chatStore))
}

// Middleware reads X-Forwarded-* headers from OAuth2 Proxy and injects a
// Session into the request context. Falls through to anonymous for non-API
// paths. Supports STUDIO_API_TOKEN bypass for CI/scripts.
func (h *Handler) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/config" {
				return next(c)
			}

			if h.apiToken != "" {
				if bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); bearer == h.apiToken {
					sess := &Session{
						Email:          "api-token@internal",
						Name:           "API Token",
						ServiceAccount: true,
					}
					ctx := context.WithValue(r.Context(), sessionKey, sess)
					ctx = httputil.WithIdentity(ctx, "api-token@internal")
					c.SetRequest(r.WithContext(ctx))
					authRequestTotal.Add("api_token", 1)
					return next(c)
				}
			}

			email := r.Header.Get("X-Forwarded-Email")
			if email == "" {
				authRequestTotal.Add("anonymous", 1)
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			name := r.Header.Get("X-Forwarded-Preferred-Username")
			if name == "" {
				name = emailLocalPart(email)
			}
			sess := &Session{
				Email:  email,
				Name:   name,
				Login:  r.Header.Get("X-Forwarded-User"),
				Groups: splitGroups(r.Header.Get("X-Forwarded-Groups")),
			}

			if h.users != nil {
				h.ensureUser(r.Context(), sess)
			}

			ctx := context.WithValue(r.Context(), sessionKey, sess)
			ctx = httputil.WithIdentity(ctx, email)
			c.SetRequest(r.WithContext(ctx))
			authRequestTotal.Add("authenticated", 1)
			return next(c)
		}
	}
}

// serviceAccountAllowedPaths are the only write paths that STUDIO_API_TOKEN
// can access. All other mutating /api/* routes require a real admin user.
// This limits the blast radius of a leaked token to seed/ingest operations.
var serviceAccountAllowedPaths = []string{
	"/api/evidence/ingest",
	"/api/policies/import",
	"/api/catalogs",
}

// RequireAdmin returns middleware that rejects non-admin requests with 403.
// Fails closed: if the store lookup errors, the request is rejected.
// Service accounts (API token) are restricted to serviceAccountAllowedPaths.
func RequireAdmin(users UserStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, ok := SessionFrom(c.Request().Context())
			if !ok {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "admin role required"})
			}
			if sess.ServiceAccount {
				path := c.Request().URL.Path
				for _, allowed := range serviceAccountAllowedPaths {
					if strings.HasPrefix(path, allowed) {
						return next(c)
					}
				}
				return c.JSON(http.StatusForbidden, map[string]string{"error": "api token not authorized for this endpoint"})
			}
			if users == nil {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "admin role required"})
			}
			u, err := users.GetUser(c.Request().Context(), sess.Email)
			if err != nil || u.Role != consts.RoleAdmin {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "admin role required"})
			}
			return next(c)
		}
	}
}

// TokenFromRequest reads the access token from the X-Forwarded-Access-Token
// header injected by OAuth2 Proxy. Used by A2A proxy and publish modules.
func (h *Handler) TokenFromRequest(r *http.Request) (string, bool) {
	token := r.Header.Get("X-Forwarded-Access-Token")
	if token == "" {
		return "", false
	}
	return token, true
}

func (h *Handler) handleMeEcho(c echo.Context) error {
	r := c.Request()
	email := r.Header.Get("X-Forwarded-Email")
	if email == "" {
		sess, ok := SessionFrom(r.Context())
		if ok {
			email = sess.Email
		}
	}
	if email == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}
	name := r.Header.Get("X-Forwarded-Preferred-Username")
	if name == "" {
		name = emailLocalPart(email)
	}
	info := UserInfo{
		Login: r.Header.Get("X-Forwarded-User"),
		Name:  name,
		Email: email,
		Role:  consts.RoleReviewer,
	}
	if h.users != nil {
		if u, err := h.users.GetUser(r.Context(), email); err == nil {
			info.Role = u.Role
			info.Name = u.Name
			info.AvatarURL = u.AvatarURL
		}
	}
	return c.JSON(http.StatusOK, info)
}

// ensureUser upserts the user on first-seen and seeds the admin role if the
// user's groups contain "admin" and no admin exists yet.
func (h *Handler) ensureUser(ctx context.Context, sess *Session) {
	sub := sess.Login
	if sub == "" {
		sub = sess.Email
	}
	err := h.users.UpsertUser(ctx, sub, "oauth2-proxy", sess.Email, sess.Name, sess.AvatarURL)
	if err != nil {
		slog.Warn("user upsert failed", "email", sess.Email, "error", err)
		return
	}

	if !containsAdmin(sess.Groups) {
		return
	}
	oldRole, err := h.users.BootstrapAdmin(ctx, sess.Email)
	if err != nil {
		return
	}
	_ = h.users.InsertRoleChange(ctx, RoleChange{
		ChangedBy:   "oauth2-proxy-group-seed",
		TargetEmail: sess.Email,
		OldRole:     oldRole,
		NewRole:     consts.RoleAdmin,
	})
	slog.Info("admin role seeded from proxy groups", "email", sess.Email)
	authUserUpsertTotal.Add(1)
}

func containsAdmin(groups []string) bool {
	for _, g := range groups {
		if g == "admin" || g == "admins" || g == consts.RoleAdmin {
			return true
		}
	}
	return false
}

func emailLocalPart(email string) string {
	if i := strings.IndexByte(email, '@'); i > 0 {
		return email[:i]
	}
	return email
}

func splitGroups(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, g := range strings.Split(raw, ",") {
		g = strings.TrimSpace(g)
		if g != "" {
			out = append(out, g)
		}
	}
	return out
}

func (h *Handler) handleGetChatHistory(cs ChatStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, ok := SessionFrom(c.Request().Context())
		if !ok || sess.Email == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		}
		chat, err := cs.GetChat(c.Request().Context(), sess.Email)
		if err != nil {
			if errors.Is(err, ErrChatNotFound) || errors.Is(err, ErrChatExpired) {
				return c.JSON(http.StatusOK, map[string]any{"messages": nil, "taskId": ""})
			}
			slog.Error("chat history load failed", "email", sess.Email, "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "chat history unavailable"})
		}
		return c.JSON(http.StatusOK, chat)
	}
}

func (h *Handler) handlePutChatHistory(cs ChatStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, ok := SessionFrom(c.Request().Context())
		if !ok || sess.Email == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		}
		var chat ChatSession
		if err := c.Bind(&chat); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}
		if err := cs.PutChat(c.Request().Context(), sess.Email, chat); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "save failed"})
		}
		return c.NoContent(http.StatusNoContent)
	}
}
