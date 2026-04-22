// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/complytime/complytime-studio/internal/httputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options configures the registry proxy module.
type Options struct {
	MCPURL             string
	InsecureRegistries []string
}

// Register mounts OCI registry proxy routes on the mux.
func Register(mux *http.ServeMux, opts Options) {
	insecure := parseInsecureRegistries(opts.InsecureRegistries)

	if opts.MCPURL == "" && len(insecure) == 0 {
		mux.HandleFunc("/api/registry/", httputil.UnavailableHandler("oras-mcp unavailable"))
		slog.Info("registry proxy disabled", "reason", "ORAS_MCP_URL not set")
		return
	}

	rp := &proxy{
		mcpURL:   opts.MCPURL,
		insecure: insecure,
		client:   &http.Client{Timeout: 15 * time.Second},
	}

	mux.HandleFunc("/api/registry/repositories", rp.handleListRepositories)
	mux.HandleFunc("/api/registry/tags", rp.handleListTags)
	mux.HandleFunc("/api/registry/manifest", rp.handleFetchManifest)
	mux.HandleFunc("/api/registry/layer", rp.handleFetchLayer)
	if len(insecure) > 0 {
		slog.Info("registry proxy registered", "insecure", insecure)
	} else {
		slog.Info("registry proxy registered")
	}
}

type proxy struct {
	mcpURL   string
	insecure []string
	client   *http.Client
}

func (rp *proxy) connectMCP(ctx context.Context) (*mcp.ClientSession, error) {
	if rp.mcpURL == "" {
		return nil, fmt.Errorf("oras-mcp URL not configured")
	}
	transport := &mcp.StreamableClientTransport{Endpoint: rp.mcpURL}
	orasClient := mcp.NewClient(
		&mcp.Implementation{Name: "studio-oras-proxy", Version: "0.1.0"},
		&mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}},
	)
	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return orasClient.Connect(connectCtx, transport, nil)
}

func (rp *proxy) isInsecure(host string) bool {
	if !isValidRegistryHost(host) {
		return false
	}
	for _, h := range rp.insecure {
		if h == host {
			return true
		}
	}
	return false
}

// isValidRegistryHost rejects host values that could cause SSRF via URL
// injection (path separators, userinfo markers, backslashes).
func isValidRegistryHost(host string) bool {
	if host == "" {
		return false
	}
	return !strings.ContainsAny(host, "/@\\")
}

func (rp *proxy) directGet(ctx context.Context, host, path string) ([]byte, error) {
	if !isValidRegistryHost(host) {
		return nil, fmt.Errorf("invalid registry host: %q", host)
	}
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

func (rp *proxy) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	registry := r.URL.Query().Get("registry")
	if rp.isInsecure(registry) {
		body, err := rp.directGet(r.Context(), registry, "/v2/_catalog")
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		var catalog struct {
			Repositories []string `json:"repositories"`
		}
		if err := json.Unmarshal(body, &catalog); err != nil {
			httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": "parse catalog: " + err.Error()})
			return
		}
		type repoEntry struct {
			Name string `json:"name"`
		}
		out := make([]repoEntry, len(catalog.Repositories))
		for i, name := range catalog.Repositories {
			out[i] = repoEntry{Name: name}
		}
		httputil.WriteJSON(w, http.StatusOK, out)
		return
	}
	rp.toolCall(w, r, "list_repositories", map[string]any{"registry": registry})
}

func (rp *proxy) handleListTags(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	host, repo := splitReference(ref)
	if rp.isInsecure(host) {
		body, err := rp.directGet(r.Context(), host, fmt.Sprintf("/v2/%s/tags/list", repo))
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		var tagList struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal(body, &tagList); err != nil {
			httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": "parse tags: " + err.Error()})
			return
		}
		type tagEntry struct {
			Name string `json:"name"`
		}
		out := make([]tagEntry, len(tagList.Tags))
		for i, t := range tagList.Tags {
			out[i] = tagEntry{Name: t}
		}
		httputil.WriteJSON(w, http.StatusOK, out)
		return
	}
	rp.toolCall(w, r, "list_tags", map[string]any{"reference": ref})
}

func (rp *proxy) handleFetchManifest(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	host, repoAndTag := splitReference(ref)
	if rp.isInsecure(host) {
		repo, tag := splitRepoTag(repoAndTag)
		body, err := rp.directGet(r.Context(), host, fmt.Sprintf("/v2/%s/manifests/%s", repo, tag))
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
		return
	}
	rp.toolCall(w, r, "fetch_manifest", map[string]any{"reference": ref})
}

func (rp *proxy) handleFetchLayer(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	host, repoAndDigest := splitReference(ref)
	if rp.isInsecure(host) {
		repo, digest := splitRepoDigest(repoAndDigest)
		body, err := rp.directGet(r.Context(), host, fmt.Sprintf("/v2/%s/blobs/%s", repo, digest))
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
		return
	}
	rp.toolCall(w, r, "fetch_layer", map[string]any{"reference": ref})
}

func (rp *proxy) toolCall(w http.ResponseWriter, r *http.Request, toolName string, args map[string]any) {
	sess, err := rp.connectMCP(r.Context())
	if err != nil {
		httputil.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "oras-mcp unavailable: " + err.Error()})
		return
	}
	defer sess.Close()
	res, err := sess.CallTool(r.Context(), &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
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

func parseInsecureRegistries(raw []string) []string {
	var out []string
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// splitReference splits "host:port/repo/name:tag" into host and remainder.
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
