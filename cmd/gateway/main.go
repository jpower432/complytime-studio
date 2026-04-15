// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/goccy/go-yaml"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/complytime/complytime-studio/internal/auth"
	"github.com/complytime/complytime-studio/internal/proxy"
	"github.com/complytime/complytime-studio/internal/publish"
	"github.com/complytime/complytime-studio/workbench"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port := envOr("PORT", "8080")
	gemaraMCPURL := os.Getenv("GEMARA_MCP_URL")
	orasMCPURL := os.Getenv("ORAS_MCP_URL")
	signingEnabled := os.Getenv("SIGNING_ENABLED") == "true"
	signingKeyRef := os.Getenv("SIGNING_KEY_REF")

	mux := http.NewServeMux()

	authCfg := auth.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		CallbackURL:  envOr("GITHUB_CALLBACK_URL", "http://localhost:8080/auth/callback"),
	}
	signKey := cookieSignKey()

	authHandler := auth.NewHandler(authCfg, signKey)
	authHandler.Register(mux)

	registerGemaraProxy(mux, gemaraMCPURL)
	registerRegistryProxy(mux, orasMCPURL)
	registerPublishEndpoint(mux, signingEnabled, signingKeyRef)
	registerAgentDirectory(mux)
	registerA2AProxy(mux)
	registerWorkbench(mux)

	var handler http.Handler = mux
	if authCfg.ClientID != "" {
		handler = authHandler.Middleware(mux)
		log.Print("auth: GitHub OAuth enabled")
	} else {
		log.Print("auth: disabled (GITHUB_CLIENT_ID not set)")
	}

	addr := net.JoinHostPort("0.0.0.0", port)
	log.Printf("ComplyTime Studio Gateway: http://localhost:%s", port)
	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http: %v", err)
	}
	_ = ctx
}

func registerGemaraProxy(mux *http.ServeMux, mcpURL string) {
	if mcpURL == "" {
		unavailable := unavailableHandler("gemara-mcp unavailable")
		mux.HandleFunc("/api/validate", unavailable)
		mux.HandleFunc("/api/migrate", unavailable)
		log.Print("gemara-mcp proxy: disabled (GEMARA_MCP_URL not set)")
		return
	}

	transport := &mcp.StreamableClientTransport{Endpoint: mcpURL}
	p, err := proxy.New(transport)
	if err != nil {
		log.Printf("WARNING: gemara-mcp proxy disabled: %v", err)
		unavailable := unavailableHandler("gemara-mcp unavailable")
		mux.HandleFunc("/api/validate", unavailable)
		mux.HandleFunc("/api/migrate", unavailable)
		return
	}

	mux.HandleFunc("/api/validate", p.ValidateHandler())
	mux.HandleFunc("/api/migrate", p.MigrateHandler())
	log.Print("gemara-mcp proxy: /api/validate, /api/migrate")
}

func registerRegistryProxy(mux *http.ServeMux, mcpURL string) {
	if mcpURL == "" {
		mux.HandleFunc("/api/registry/", unavailableHandler("oras-mcp unavailable"))
		log.Print("registry proxy: disabled (ORAS_MCP_URL not set)")
		return
	}

	transport := &mcp.StreamableClientTransport{Endpoint: mcpURL}
	orasClient := mcp.NewClient(
		&mcp.Implementation{Name: "studio-oras-proxy", Version: "0.1.0"},
		&mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}},
	)
	connectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sess, err := orasClient.Connect(connectCtx, transport, nil)
	if err != nil {
		log.Printf("WARNING: registry proxy disabled: %v", err)
		mux.HandleFunc("/api/registry/", unavailableHandler("oras-mcp unavailable"))
		return
	}

	mux.HandleFunc("/api/registry/repositories", registryToolHandler(sess, "list_repositories", func(r *http.Request) map[string]any {
		return map[string]any{"registry": r.URL.Query().Get("registry")}
	}))
	mux.HandleFunc("/api/registry/tags", registryToolHandler(sess, "list_tags", func(r *http.Request) map[string]any {
		return map[string]any{"reference": r.URL.Query().Get("ref")}
	}))
	mux.HandleFunc("/api/registry/manifest", registryToolHandler(sess, "fetch_manifest", func(r *http.Request) map[string]any {
		return map[string]any{"reference": r.URL.Query().Get("ref")}
	}))
	mux.HandleFunc("/api/registry/layer", registryToolHandler(sess, "fetch_manifest", func(r *http.Request) map[string]any {
		return map[string]any{"reference": r.URL.Query().Get("ref")}
	}))
	log.Print("registry proxy: /api/registry/*")
}

func registryToolHandler(sess *mcp.ClientSession, toolName string, argsBuilder func(*http.Request) map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		args := argsBuilder(r)
		res, err := sess.CallTool(r.Context(), &mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		var sb strings.Builder
		for _, c := range res.Content {
			if t, ok := c.(*mcp.TextContent); ok {
				sb.WriteString(t.Text)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sb.String()))
	}
}

func registerPublishEndpoint(mux *http.ServeMux, signingEnabled bool, signingKeyRef string) {
	mux.HandleFunc("/api/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Artifacts []string `json:"artifacts"`
			Target    string   `json:"target"`
			Tag       string   `json:"tag"`
			Sign      bool     `json:"sign"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var artifacts []publish.ArtifactInput
		for i, raw := range req.Artifacts {
			at, err := extractArtifactType(raw)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("artifact[%d]: %v", i, err)})
				return
			}
			artifacts = append(artifacts, publish.ArtifactInput{Type: at, Content: []byte(raw)})
		}

		result, err := publish.AssembleAndPush(r.Context(), artifacts, req.Target, req.Tag)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		if req.Sign {
			if err := publish.SignBundle(r.Context(), publish.SigningConfig{
				Enabled: signingEnabled,
				KeyRef:  signingKeyRef,
			}, result.Reference, result.Digest); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "signing: " + err.Error()})
				return
			}
		}

		writeJSON(w, http.StatusOK, result)
	})
	log.Print("publish endpoint: /api/publish")
}

// AgentCard represents a specialist agent entry in the directory.
type AgentCard struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	URL         string       `json:"url"`
	Skills      []AgentSkill `json:"skills"`
}

// AgentSkill describes one A2A skill exposed by a specialist agent.
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

func registerAgentDirectory(mux *http.ServeMux) {
	raw := os.Getenv("AGENT_DIRECTORY")
	var cards []AgentCard
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &cards); err != nil {
			log.Printf("WARNING: AGENT_DIRECTORY parse error: %v", err)
		}
	}

	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cards == nil {
			cards = []AgentCard{}
		}
		writeJSON(w, http.StatusOK, cards)
	})
	log.Printf("agent directory: /api/agents (%d agents)", len(cards))
}

// registerA2AProxy sets up the reverse proxy for A2A requests to agent pods.
// It injects the user's GitHub token as the Authorization header.
func registerA2AProxy(mux *http.ServeMux) {
	mux.HandleFunc("/api/a2a/", func(w http.ResponseWriter, r *http.Request) {
		agentName := strings.TrimPrefix(r.URL.Path, "/api/a2a/")
		agentName = strings.SplitN(agentName, "/", 2)[0]
		if agentName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing agent name"})
			return
		}

		target, err := url.Parse(fmt.Sprintf("http://%s:8080", agentName))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent name"})
			return
		}

		rp := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.URL.Path = "/"
				req.Host = target.Host

				if sess, ok := auth.SessionFrom(req.Context()); ok {
					req.Header.Set("Authorization", "Bearer "+sess.GitHubToken)
				}
			},
			FlushInterval: -1,
		}

		rp.ServeHTTP(w, r)
	})
	log.Print("a2a proxy: /api/a2a/{agent-name}")
}

func registerWorkbench(mux *http.ServeMux) {
	assets, err := fs.Sub(workbench.Assets, "dist")
	if err != nil {
		log.Fatalf("embed workbench: %v", err)
	}

	fileServer := http.FileServer(http.FS(assets))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}
		f, err := assets.(fs.ReadFileFS).ReadFile(strings.TrimPrefix(path, "/"))
		if err != nil {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		_ = f
		fileServer.ServeHTTP(w, r)
	})
}

func extractArtifactType(yamlContent string) (gemara.ArtifactType, error) {
	var hdr struct {
		Metadata gemara.Metadata `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(yamlContent), &hdr); err != nil {
		return gemara.InvalidArtifact, fmt.Errorf("invalid YAML: %w", err)
	}
	if hdr.Metadata.Type == gemara.InvalidArtifact {
		return gemara.InvalidArtifact, fmt.Errorf("missing or invalid metadata.type")
	}
	return hdr.Metadata.Type, nil
}

func unavailableHandler(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": msg})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// readBody reads up to maxBytes from an io.Reader.
func readBody(body io.Reader, maxBytes int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(body, maxBytes))
}

// cookieSignKey returns the signing key from env or generates an ephemeral one.
func cookieSignKey() []byte {
	if key := os.Getenv("COOKIE_SIGN_KEY"); key != "" {
		return []byte(key)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("failed to generate cookie signing key: %v", err)
	}
	log.Print("WARNING: using ephemeral cookie signing key (sessions lost on restart)")
	return key
}
