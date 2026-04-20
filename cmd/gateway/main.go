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
	"github.com/complytime/complytime-studio/internal/config"
	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/httputil"
	"github.com/complytime/complytime-studio/internal/proxy"
	"github.com/complytime/complytime-studio/internal/publish"
	"github.com/complytime/complytime-studio/internal/registry"
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

	authCfg := auth.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		CallbackURL:  httputil.EnvOr("GITHUB_CALLBACK_URL", "http://localhost:8080/auth/callback"),
	}
	secretKey := cookieSecretKey(authCfg.ClientID != "")
	authHandler, err := auth.NewHandler(authCfg, secretKey)
	if err != nil {
		slog.Error("auth handler init failed", "error", err)
		os.Exit(1)
	}
	authHandler.Register(mux)

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
	agents.Register(mux, agents.Options{
		Cards:          agentCards,
		TokenProvider:  authHandler,
		KagentA2AURL:   os.Getenv("KAGENT_A2A_URL"),
		AgentNamespace: os.Getenv("KAGENT_AGENT_NAMESPACE"),
	})

	config.Register(mux, config.Options{
		Values: map[string]string{
			"github_org":        httputil.EnvOr("GITHUB_ORG", ""),
			"github_repo":       httputil.EnvOr("GITHUB_REPO", "complytime-studio"),
			"registry_insecure": httputil.EnvOr("REGISTRY_INSECURE", ""),
		},
	})

	web.Register(mux, workbench.Assets)

	var handler http.Handler = mux
	if authCfg.ClientID != "" {
		handler = authHandler.Middleware(mux)
		slog.Info("auth enabled", "provider", "github-oauth")
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
		slog.Warn("COOKIE_SECRET not set — sessions will not survive restarts", "hint", "openssl rand -hex 32")
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		slog.Error("failed to generate cookie secret", "error", err)
		os.Exit(1)
	}
	return key
}
