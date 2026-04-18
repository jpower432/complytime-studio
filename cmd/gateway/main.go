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
	"path/filepath"
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
	registerWorkspaceSave(mux)
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
	insecureRegistries := parseInsecureRegistries(os.Getenv("REGISTRY_INSECURE"))

	var sess *mcp.ClientSession
	if mcpURL != "" {
		transport := &mcp.StreamableClientTransport{Endpoint: mcpURL}
		orasClient := mcp.NewClient(
			&mcp.Implementation{Name: "studio-oras-proxy", Version: "0.1.0"},
			&mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}},
		)
		connectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		var err error
		sess, err = orasClient.Connect(connectCtx, transport, nil)
		if err != nil {
			log.Printf("WARNING: oras-mcp connection failed: %v", err)
		}
	}

	if sess == nil && len(insecureRegistries) == 0 {
		mux.HandleFunc("/api/registry/", unavailableHandler("oras-mcp unavailable"))
		log.Print("registry proxy: disabled (ORAS_MCP_URL not set)")
		return
	}

	rp := &registryProxy{sess: sess, insecure: insecureRegistries, client: &http.Client{Timeout: 15 * time.Second}}

	mux.HandleFunc("/api/registry/repositories", rp.handleListRepositories)
	mux.HandleFunc("/api/registry/tags", rp.handleListTags)
	mux.HandleFunc("/api/registry/manifest", rp.handleFetchManifest)
	mux.HandleFunc("/api/registry/layer", rp.handleFetchLayer)
	if len(insecureRegistries) > 0 {
		log.Printf("registry proxy: /api/registry/* (insecure: %v)", insecureRegistries)
	} else {
		log.Print("registry proxy: /api/registry/*")
	}
}

func parseInsecureRegistries(raw string) []string {
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

type registryProxy struct {
	sess     *mcp.ClientSession
	insecure []string
	client   *http.Client
}

func (rp *registryProxy) isInsecure(host string) bool {
	for _, h := range rp.insecure {
		if h == host {
			return true
		}
	}
	return false
}

func (rp *registryProxy) directGet(ctx context.Context, host, path string) ([]byte, error) {
	u := fmt.Sprintf("http://%s%s", host, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json")
	resp, err := rp.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry %s: %s", u, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func (rp *registryProxy) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	registry := r.URL.Query().Get("registry")
	if rp.isInsecure(registry) {
		body, err := rp.directGet(r.Context(), registry, "/v2/_catalog")
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		var catalog struct {
			Repositories []string `json:"repositories"`
		}
		if err := json.Unmarshal(body, &catalog); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "parse catalog: " + err.Error()})
			return
		}
		type repoEntry struct {
			Name string `json:"name"`
		}
		out := make([]repoEntry, len(catalog.Repositories))
		for i, name := range catalog.Repositories {
			out[i] = repoEntry{Name: name}
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	rp.toolCall(w, r, "list_repositories", map[string]any{"registry": registry})
}

func (rp *registryProxy) handleListTags(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	host, repo := splitReference(ref)
	if rp.isInsecure(host) {
		body, err := rp.directGet(r.Context(), host, fmt.Sprintf("/v2/%s/tags/list", repo))
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		var tagList struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal(body, &tagList); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "parse tags: " + err.Error()})
			return
		}
		type tagEntry struct {
			Name string `json:"name"`
		}
		out := make([]tagEntry, len(tagList.Tags))
		for i, t := range tagList.Tags {
			out[i] = tagEntry{Name: t}
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	rp.toolCall(w, r, "list_tags", map[string]any{"reference": ref})
}

func (rp *registryProxy) handleFetchManifest(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	host, repoAndTag := splitReference(ref)
	if rp.isInsecure(host) {
		repo, tag := splitRepoTag(repoAndTag)
		body, err := rp.directGet(r.Context(), host, fmt.Sprintf("/v2/%s/manifests/%s", repo, tag))
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
		return
	}
	rp.toolCall(w, r, "fetch_manifest", map[string]any{"reference": ref})
}

func (rp *registryProxy) handleFetchLayer(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	host, repoAndDigest := splitReference(ref)
	if rp.isInsecure(host) {
		repo, digest := splitRepoDigest(repoAndDigest)
		body, err := rp.directGet(r.Context(), host, fmt.Sprintf("/v2/%s/blobs/%s", repo, digest))
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
		return
	}
	rp.toolCall(w, r, "fetch_layer", map[string]any{"reference": ref})
}

func (rp *registryProxy) toolCall(w http.ResponseWriter, r *http.Request, toolName string, args map[string]any) {
	if rp.sess == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "oras-mcp unavailable"})
		return
	}
	res, err := rp.sess.CallTool(r.Context(), &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(t.Text)
		}
	}
	body := sb.String()
	w.Header().Set("Content-Type", "application/json")
	if !json.Valid([]byte(body)) {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": body})
		return
	}
	_, _ = w.Write([]byte(body))
}

// splitReference splits "host:port/repo/name:tag" into host ("host:port") and remainder ("repo/name:tag").
func splitReference(ref string) (host, remainder string) {
	idx := strings.Index(ref, "/")
	if idx < 0 {
		return ref, ""
	}
	return ref[:idx], ref[idx+1:]
}

// splitRepoTag splits "repo/name:tag" into ("repo/name", "tag").
func splitRepoTag(s string) (repo, tag string) {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return s, "latest"
	}
	return s[:idx], s[idx+1:]
}

// splitRepoDigest splits "repo/name@sha256:abc" into ("repo/name", "sha256:abc").
func splitRepoDigest(s string) (repo, digest string) {
	idx := strings.Index(s, "@")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
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

func registerWorkspaceSave(mux *http.ServeMux) {
	const artifactsDir = ".complytime/artifacts"

	mux.HandleFunc("/api/workspace/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Filename string `json:"filename"`
			Content  string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Filename == "" || req.Content == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "filename and content are required"})
			return
		}

		cleaned := filepath.Clean(req.Filename)
		if filepath.IsAbs(cleaned) || strings.Contains(cleaned, "..") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid filename"})
			return
		}

		if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create directory: " + err.Error()})
			return
		}

		dest := filepath.Join(artifactsDir, cleaned)
		if err := os.WriteFile(dest, []byte(req.Content), 0o644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "write file: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"path": dest})
	})
	log.Print("workspace save: /api/workspace/save")
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
				req.URL.Path = "/invoke"
				req.Host = target.Host

				if sess, ok := auth.SessionFrom(req.Context()); ok {
					req.Header.Set("Authorization", "Bearer "+sess.GitHubToken)
				}
			},
			Transport: &http.Transport{
				ResponseHeaderTimeout: 5 * time.Minute,
			},
			FlushInterval: -1,
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				log.Printf("a2a proxy error for %s: %v", agentName, err)
				writeJSON(rw, http.StatusBadGateway, map[string]string{
					"error": fmt.Sprintf("agent %s unreachable: %v", agentName, err),
				})
			},
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
