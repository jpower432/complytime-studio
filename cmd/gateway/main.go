// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/complytime/complytime-studio/internal/agents"
	"github.com/complytime/complytime-studio/internal/auth"
	chclient "github.com/complytime/complytime-studio/internal/clickhouse"
	"github.com/complytime/complytime-studio/internal/config"
	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/httputil"
	"github.com/complytime/complytime-studio/internal/proxy"
	"github.com/complytime/complytime-studio/internal/publish"
	"github.com/complytime/complytime-studio/internal/registry"
	"github.com/complytime/complytime-studio/internal/store"
	"github.com/complytime/complytime-studio/internal/web"
	"github.com/complytime/complytime-studio/workbench"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port := httputil.EnvOr("PORT", "8080")
	mux := http.NewServeMux()

	var chClient *chclient.Client
	if chAddr := os.Getenv("CLICKHOUSE_ADDR"); chAddr != "" {
		chCfg := chclient.Config{
			Addr:     chAddr,
			User:     httputil.EnvOr("CLICKHOUSE_USER", "default"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		}
		maxRetries := 90
		for attempt := 1; attempt <= maxRetries; attempt++ {
			var err error
			chClient, err = chclient.New(ctx, chCfg)
			if err == nil {
				break
			}
			if attempt == maxRetries {
				slog.Error("clickhouse connection failed after retries", "error", err, "attempts", maxRetries)
				os.Exit(1)
			}
			slog.Warn("clickhouse not ready, retrying", "attempt", attempt, "error", err)
			select {
			case <-ctx.Done():
				os.Exit(1)
			case <-time.After(2 * time.Second):
			}
		}
		if err := chClient.EnsureSchema(ctx, 24); err != nil {
			slog.Error("clickhouse schema init failed", "error", err)
			os.Exit(1)
		}
		slog.Info("clickhouse ready")
	}

	var st *store.Store
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if chClient != nil {
			if err := chClient.Ping(r.Context()); err != nil {
				http.Error(w, "clickhouse unreachable", http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if chClient != nil {
		st = store.New(chClient.Conn())
		if err := store.PopulateMappingEntries(ctx, st); err != nil {
			slog.Warn("mapping entries backfill failed", "error", err)
		}
		if err := store.PopulateControls(ctx, st, st); err != nil {
			slog.Warn("controls backfill failed", "error", err)
		}
		if err := store.PopulateThreats(ctx, st, st); err != nil {
			slog.Warn("threats backfill failed", "error", err)
		}
		store.Register(mux, store.Stores{
			Policies:  st,
			Mappings:  st,
			Evidence:  st,
			AuditLogs: st,
			Controls:  st,
			Threats:   st,
			Catalogs:  st,
		})
		slog.Info("store API registered", "routes", []string{
			"/api/policies", "/api/evidence", "/api/audit-logs", "/api/mappings",
		})
	}

	authCfg := auth.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		CallbackURL:  httputil.EnvOr("GOOGLE_CALLBACK_URL", "http://localhost:8080/auth/callback"),
	}
	secretKey := cookieSecretKey(authCfg.ClientID != "")
	sessionStore := auth.NewMemorySessionStore()
	authHandler, err := auth.NewHandler(authCfg, secretKey, sessionStore)
	if err != nil {
		slog.Error("auth handler init failed", "error", err)
		os.Exit(1)
	}
	adminEmails := splitComma(os.Getenv("ADMIN_EMAILS"))
	admins := make(map[string]bool, len(adminEmails))
	for _, e := range adminEmails {
		admins[strings.ToLower(e)] = true
	}
	authHandler.SetAdmins(admins)
	if len(admins) > 0 {
		slog.Info("admin allowlist configured", "count", len(admins))
	} else {
		slog.Info("no admin allowlist — all users are admin")
	}
	authHandler.Register(mux)
	authHandler.RegisterChatHistory(mux, sessionStore)

	if apiToken := os.Getenv("STUDIO_API_TOKEN"); apiToken != "" {
		authHandler.SetAPIToken(apiToken)
		slog.Info("api token auth enabled for seed/CI scripts")
	}

	registerGemaraProxy(mux, os.Getenv("GEMARA_MCP_URL"))

	insecureRaw := os.Getenv("REGISTRY_INSECURE")
	insecureList := splitComma(insecureRaw)

	registry.Register(mux, registry.Options{
		MCPURL:             os.Getenv("ORAS_MCP_URL"),
		InsecureRegistries: insecureList,
	})

	publish.Register(mux, publish.Options{
		TokenProvider:      authHandler,
		InsecureRegistries: insecureList,
	})

	agentCards := agents.ParseDirectory(os.Getenv("AGENT_DIRECTORY"))
	agents.RegisterDirectory(mux, agentCards)

	if a2aProxyURL := os.Getenv("A2A_PROXY_URL"); a2aProxyURL != "" {
		agents.RegisterA2AForward(mux, a2aProxyURL)
	} else {
		autoPersist := httputil.EnvOr("AUTO_PERSIST_ARTIFACTS", "true") == "true"
		if autoPersist && st != nil {
			slog.Info("artifact auto-persist enabled")
		}
		agents.RegisterA2AProxy(mux, agents.Options{
			Cards:                agentCards,
			TokenProvider:        authHandler,
			KagentA2AURL:         os.Getenv("KAGENT_A2A_URL"),
			AgentNamespace:       os.Getenv("KAGENT_AGENT_NAMESPACE"),
			AutoPersistArtifacts: autoPersist && st != nil,
			AuditLogStore:        st,
		})
		slog.Info("a2a proxy embedded in gateway (A2A_PROXY_URL not set)")
	}

	config.Register(mux, config.Options{
		Values: map[string]string{
			"github_org":             httputil.EnvOr("GITHUB_ORG", ""),
			"github_repo":            httputil.EnvOr("GITHUB_REPO", "complytime-studio"),
			"registry_insecure":      httputil.EnvOr("REGISTRY_INSECURE", ""),
			"model_provider":         httputil.EnvOr("MODEL_PROVIDER", ""),
			"model_name":             httputil.EnvOr("MODEL_NAME", ""),
			"auto_persist_artifacts": httputil.EnvOr("AUTO_PERSIST_ARTIFACTS", "true"),
		},
	})

	web.Register(mux, workbench.Assets)

	var handler http.Handler = mux
	if authCfg.ClientID != "" {
		adminGuard := auth.RequireAdmin(admins)
		handler = authHandler.Middleware(writeProtect(mux, adminGuard))
		slog.Info("auth enabled", "provider", "google-oauth")
	} else {
		slog.Info("auth disabled")
	}

	if origins := splitComma(os.Getenv("CORS_ORIGINS")); len(origins) > 0 {
		handler = httputil.CORS(httputil.CORSOptions{AllowedOrigins: origins})(handler)
		slog.Info("CORS enabled", "origins", origins)
	}

	handler = httputil.SecurityHeaders(handler)

	addr := net.JoinHostPort("0.0.0.0", port)
	slog.Info("gateway starting", "addr", addr)
	srv := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    consts.ServerReadTimeout,
		WriteTimeout:   consts.ServerWriteTimeout,
		IdleTimeout:    consts.ServerIdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("http server failed", "error", err)
		os.Exit(1)
	}
	_ = ctx
}

func registerGemaraProxy(mux *http.ServeMux, mcpURL string) {
	if mcpURL == "" {
		unavailable := httputil.UnavailableHandler("gemara-mcp unavailable")
		mux.HandleFunc("/api/validate", unavailable)
		mux.HandleFunc("/api/migrate", unavailable)
		slog.Info("gemara-mcp proxy disabled", "reason", "GEMARA_MCP_URL not set")
		return
	}

	transport := &mcp.StreamableClientTransport{Endpoint: mcpURL}
	p, err := proxy.New(transport)
	if err != nil {
		slog.Warn("gemara-mcp proxy disabled", "error", err)
		unavailable := httputil.UnavailableHandler("gemara-mcp unavailable")
		mux.HandleFunc("/api/validate", unavailable)
		mux.HandleFunc("/api/migrate", unavailable)
		return
	}

	mux.HandleFunc("/api/validate", p.ValidateHandler())
	mux.HandleFunc("/api/migrate", p.MigrateHandler())
	slog.Info("gemara-mcp proxy registered", "routes", []string{"/api/validate", "/api/migrate"})
}

func splitComma(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// writeProtect wraps a handler so that POST/PUT/PATCH/DELETE requests to /api/*
// pass through the adminGuard middleware first. GET and non-API requests are unaffected.
func writeProtect(next http.Handler, adminGuard func(http.Handler) http.Handler) http.Handler {
	guarded := adminGuard(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet {
			if r.URL.Path == "/api/chat/history" {
				next.ServeHTTP(w, r)
				return
			}
			guarded.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cookieSecretKey returns the 32-byte AES-256 key for session cookie encryption.
// Reads hex-encoded key from COOKIE_SECRET. Falls back to COOKIE_SIGN_KEY for
// backward compatibility. Generates an ephemeral key for development only.
func cookieSecretKey(authEnabled bool) []byte {
	raw := os.Getenv("COOKIE_SECRET")
	if raw == "" {
		raw = os.Getenv("COOKIE_SIGN_KEY")
	}
	if raw != "" {
		key, err := hex.DecodeString(raw)
		if err != nil || len(key) != 32 {
			slog.Error("COOKIE_SECRET must be 64 hex chars (32 bytes)", "hint", "openssl rand -hex 32")
			os.Exit(1)
		}
		return key
	}
	if authEnabled {
		slog.Error("COOKIE_SECRET is required when auth is enabled", "hint", "openssl rand -hex 32")
		os.Exit(1)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		slog.Error("failed to generate cookie secret", "error", err)
		os.Exit(1)
	}
	return key
}
