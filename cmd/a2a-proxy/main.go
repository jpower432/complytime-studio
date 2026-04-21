// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/complytime/complytime-studio/internal/agents"
	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/httputil"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port := httputil.EnvOr("PORT", "8081")
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	agentCards := agents.ParseDirectory(os.Getenv("AGENT_DIRECTORY"))
	agents.RegisterA2AProxy(mux, agents.Options{
		Cards:          agentCards,
		TokenProvider:  headerTokenProvider{},
		KagentA2AURL:   os.Getenv("KAGENT_A2A_URL"),
		AgentNamespace: os.Getenv("KAGENT_AGENT_NAMESPACE"),
	})

	var handler http.Handler = mux
	if origins := splitComma(os.Getenv("CORS_ORIGINS")); len(origins) > 0 {
		handler = httputil.CORS(httputil.CORSOptions{AllowedOrigins: origins})(handler)
	}
	handler = httputil.SecurityHeaders(handler)

	addr := net.JoinHostPort("0.0.0.0", port)
	slog.Info("a2a-proxy starting", "addr", addr)
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
}

// headerTokenProvider extracts Bearer tokens from the Authorization header.
// The proxy receives pre-injected tokens from the gateway or ingress — no
// cookie parsing or OAuth state needed.
type headerTokenProvider struct{}

func (headerTokenProvider) TokenFromRequest(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:], true
	}
	return "", false
}

func splitComma(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range splitTrim(raw) {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func splitTrim(raw string) []string {
	var out []string
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] == ',' {
			s := trim(raw[start:i])
			out = append(out, s)
			start = i + 1
		}
	}
	out = append(out, trim(raw[start:]))
	return out
}

func trim(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
