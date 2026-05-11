// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	reOCIRepo   = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*$`)
	reOCITag    = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`)
	reOCIDigest = regexp.MustCompile(`^sha(256:[a-f0-9]{64}|512:[a-f0-9]{128})$`)
)

// Options configures the registry proxy module.
type Options struct {
	MCPURL             string
	InsecureRegistries []string
}

// Register mounts OCI registry proxy routes on the Echo group.
func Register(g *echo.Group, opts Options) {
	insecure := parseInsecureRegistries(opts.InsecureRegistries)

	unavail := func(c echo.Context) error {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "oras-mcp unavailable"})
	}

	if opts.MCPURL == "" && len(insecure) == 0 {
		g.GET("/registry/repositories", unavail)
		g.GET("/registry/tags", unavail)
		g.GET("/registry/manifest", unavail)
		g.GET("/registry/layer", unavail)
		slog.Info("registry proxy disabled", "reason", "ORAS_MCP_URL not set")
		return
	}

	rp := &proxy{
		mcpURL:   opts.MCPURL,
		insecure: insecure,
		client:   &http.Client{Timeout: 15 * time.Second},
	}

	g.GET("/registry/repositories", rp.handleListRepositories)
	g.GET("/registry/tags", rp.handleListTags)
	g.GET("/registry/manifest", rp.handleFetchManifest)
	g.GET("/registry/layer", rp.handleFetchLayer)
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

// directGet performs an HTTP GET against an allowlisted insecure registry.
// The host MUST already have passed rp.isInsecure before calling this.
func (rp *proxy) directGet(ctx context.Context, host, path string) ([]byte, error) {
	if !rp.isInsecure(host) {
		return nil, fmt.Errorf("host %q not in insecure allowlist", host)
	}
	target := &url.URL{Scheme: "http", Host: host, Path: path}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json")
	resp, err := rp.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry %s: %s", target.Redacted(), resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func (rp *proxy) handleListRepositories(c echo.Context) error {
	registry := c.QueryParam("registry")
	if rp.isInsecure(registry) {
		body, err := rp.directGet(c.Request().Context(), registry, "/v2/_catalog")
		if err != nil {
			return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		var catalog struct {
			Repositories []string `json:"repositories"`
		}
		if err := json.Unmarshal(body, &catalog); err != nil {
			return c.JSON(http.StatusBadGateway, map[string]string{"error": "parse catalog: " + err.Error()})
		}
		type repoEntry struct {
			Name string `json:"name"`
		}
		out := make([]repoEntry, len(catalog.Repositories))
		for i, name := range catalog.Repositories {
			out[i] = repoEntry{Name: name}
		}
		return c.JSON(http.StatusOK, out)
	}
	return rp.toolCall(c, "list_repositories", map[string]any{"registry": registry})
}

func (rp *proxy) handleListTags(c echo.Context) error {
	ref := c.QueryParam("ref")
	host, repo := splitReference(ref)
	if rp.isInsecure(host) {
		if !reOCIRepo.MatchString(repo) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid repository name"})
		}
		body, err := rp.directGet(c.Request().Context(), host, fmt.Sprintf("/v2/%s/tags/list", repo))
		if err != nil {
			return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		var tagList struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal(body, &tagList); err != nil {
			return c.JSON(http.StatusBadGateway, map[string]string{"error": "parse tags: " + err.Error()})
		}
		type tagEntry struct {
			Name string `json:"name"`
		}
		out := make([]tagEntry, len(tagList.Tags))
		for i, t := range tagList.Tags {
			out[i] = tagEntry{Name: t}
		}
		return c.JSON(http.StatusOK, out)
	}
	return rp.toolCall(c, "list_tags", map[string]any{"reference": ref})
}

func (rp *proxy) handleFetchManifest(c echo.Context) error {
	ref := c.QueryParam("ref")
	host, repoAndTag := splitReference(ref)
	if rp.isInsecure(host) {
		repo, tag := splitRepoTag(repoAndTag)
		if !reOCIRepo.MatchString(repo) || !reOCITag.MatchString(tag) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid repository or tag"})
		}
		body, err := rp.directGet(c.Request().Context(), host, fmt.Sprintf("/v2/%s/manifests/%s", repo, tag))
		if err != nil {
			return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		return c.Blob(http.StatusOK, "application/json", body)
	}
	return rp.toolCall(c, "fetch_manifest", map[string]any{"reference": ref})
}

func (rp *proxy) handleFetchLayer(c echo.Context) error {
	ref := c.QueryParam("ref")
	host, repoAndDigest := splitReference(ref)
	if rp.isInsecure(host) {
		repo, digest := splitRepoDigest(repoAndDigest)
		if !reOCIRepo.MatchString(repo) || !reOCIDigest.MatchString(digest) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid repository or digest"})
		}
		body, err := rp.directGet(c.Request().Context(), host, fmt.Sprintf("/v2/%s/blobs/%s", repo, digest))
		if err != nil {
			return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		return c.Blob(http.StatusOK, "application/json", body)
	}
	return rp.toolCall(c, "fetch_layer", map[string]any{"reference": ref})
}

func (rp *proxy) toolCall(c echo.Context, toolName string, args map[string]any) error {
	ctx := c.Request().Context()
	sess, err := rp.connectMCP(ctx)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "oras-mcp unavailable: " + err.Error()})
	}
	defer func() { _ = sess.Close() }()
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	var sb strings.Builder
	for _, ct := range res.Content {
		if t, ok := ct.(*mcp.TextContent); ok {
			sb.WriteString(t.Text)
		}
	}
	body := sb.String()
	if !json.Valid([]byte(body)) {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": body})
	}
	return c.Blob(http.StatusOK, "application/json", []byte(body))
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
