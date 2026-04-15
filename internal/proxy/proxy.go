// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	mcpClientName    = "studio-gemara-proxy"
	mcpClientVersion = "0.1.0"

	mcpConnectTimeout = 30 * time.Second

	toolValidateGemaraArtifact = "validate_gemara_artifact"
	toolMigrateGemaraArtifact  = "migrate_gemara_artifact"

	defaultValidateVersion = "latest"
)

// Proxy holds a dedicated MCP client session to gemara-mcp (separate from the
// agent's mcptoolset session).
type Proxy struct {
	session *mcp.ClientSession
}

// New connects an MCP client over transport and returns a Proxy ready for HTTP
// handlers. The caller must invoke [Proxy.Close] when the proxy is no longer
// needed.
func New(transport mcp.Transport) (*Proxy, error) {
	if transport == nil {
		return nil, fmt.Errorf("nil MCP transport")
	}
	client := mcp.NewClient(
		&mcp.Implementation{Name: mcpClientName, Version: mcpClientVersion},
		&mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}},
	)
	ctx, cancel := context.WithTimeout(context.Background(), mcpConnectTimeout)
	defer cancel()
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp connect: %w", err)
	}
	return &Proxy{session: session}, nil
}

// Close shuts down the MCP session.
func (p *Proxy) Close() error {
	if p == nil || p.session == nil {
		return nil
	}
	return p.session.Close()
}

// ValidateHandler proxies POST JSON { yaml, definition, version? } to the
// validate_gemara_artifact MCP tool.
func (p *Proxy) ValidateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		var req validateHTTPRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.YAML) == "" {
			http.Error(w, "missing required field: yaml", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Definition) == "" {
			http.Error(w, "missing required field: definition", http.StatusBadRequest)
			return
		}
		version := strings.TrimSpace(req.Version)
		if version == "" {
			version = defaultValidateVersion
		}
		args := map[string]any{
			"artifact_content": req.YAML,
			"definition":       req.Definition,
			"version":          version,
		}
		res, err := p.session.CallTool(r.Context(), &mcp.CallToolParams{
			Name:      toolValidateGemaraArtifact,
			Arguments: args,
		})
		if err != nil {
			writeGemaraUnavailable(w, err)
			return
		}
		out, err := buildValidateResponse(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// MigrateHandler proxies POST JSON { yaml, artifact_type?, gemara_version? }
// to the migrate_gemara_artifact MCP tool.
func (p *Proxy) MigrateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		var req migrateHTTPRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.YAML) == "" {
			http.Error(w, "missing required field: yaml", http.StatusBadRequest)
			return
		}
		args := map[string]any{"artifact_content": req.YAML}
		if t := strings.TrimSpace(req.ArtifactType); t != "" {
			args["artifact_type"] = t
		}
		if v := strings.TrimSpace(req.GemaraVersion); v != "" {
			args["gemara_version"] = v
		}
		res, err := p.session.CallTool(r.Context(), &mcp.CallToolParams{
			Name:      toolMigrateGemaraArtifact,
			Arguments: args,
		})
		if err != nil {
			writeGemaraUnavailable(w, err)
			return
		}
		yamlOut, err := extractMigratedYAML(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, migrateHTTPResponse{YAML: yamlOut})
	}
}

type validateHTTPRequest struct {
	YAML       string `json:"yaml"`
	Definition string `json:"definition"`
	Version    string `json:"version"`
}

type migrateHTTPRequest struct {
	YAML          string `json:"yaml"`
	ArtifactType  string `json:"artifact_type"`
	GemaraVersion string `json:"gemara_version"`
}

type validateHTTPResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
	Result string   `json:"result,omitempty"`
}

type migrateHTTPResponse struct {
	YAML string `json:"yaml"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type gemaraUnavailableResponse struct {
	Error string `json:"error"`
}

func writeGemaraUnavailable(w http.ResponseWriter, err error) {
	if isMCPUnavailable(err) {
		writeJSON(w, http.StatusServiceUnavailable, gemaraUnavailableResponse{
			Error: "gemara-mcp unavailable",
		})
		return
	}
	http.Error(w, "mcp call: "+err.Error(), http.StatusInternalServerError)
}

func isMCPUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mcp.ErrConnectionClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "process exited") ||
		strings.Contains(msg, "no such process")
}

func toolResultPayload(res *mcp.CallToolResult) ([]byte, error) {
	if res == nil {
		return nil, fmt.Errorf("nil tool result")
	}
	if res.StructuredContent != nil {
		return json.Marshal(res.StructuredContent)
	}
	var b strings.Builder
	for _, c := range res.Content {
		switch t := c.(type) {
		case *mcp.TextContent:
			b.WriteString(t.Text)
		default:
			raw, err := json.Marshal(c)
			if err != nil {
				return nil, err
			}
			b.Write(raw)
		}
	}
	if b.Len() == 0 {
		return nil, fmt.Errorf("empty tool result content")
	}
	return []byte(b.String()), nil
}

func buildValidateResponse(res *mcp.CallToolResult) (validateHTTPResponse, error) {
	raw, err := toolResultPayload(res)
	if err != nil {
		return validateHTTPResponse{}, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		if res.IsError {
			s := strings.TrimSpace(string(raw))
			return validateHTTPResponse{
				Valid:  false,
				Errors: []string{s},
				Result: string(raw),
			}, nil
		}
		return validateHTTPResponse{}, fmt.Errorf("decode validate tool output: %w", err)
	}
	out := validateHTTPResponse{Errors: []string{}}
	if e, ok := m["errors"]; ok {
		out.Errors = decodeErrorList(e)
	}
	if v, ok := m["valid"]; ok {
		_ = json.Unmarshal(v, &out.Valid)
	} else {
		out.Valid = !res.IsError && len(out.Errors) == 0
	}
	if r, ok := m["result"]; ok {
		var s string
		if json.Unmarshal(r, &s) == nil {
			out.Result = s
		} else {
			out.Result = strings.Trim(string(r), `"`)
		}
	}
	if out.Errors == nil {
		out.Errors = []string{}
	}
	if !out.Valid && len(out.Errors) == 0 && res.IsError {
		out.Errors = []string{strings.TrimSpace(string(raw))}
	}
	if out.Result == "" {
		out.Result = string(raw)
	}
	return out, nil
}

func decodeErrorList(raw json.RawMessage) []string {
	var strs []string
	if err := json.Unmarshal(raw, &strs); err == nil {
		return strs
	}
	var anyList []any
	if err := json.Unmarshal(raw, &anyList); err != nil {
		return []string{string(raw)}
	}
	out := make([]string, 0, len(anyList))
	for _, item := range anyList {
		switch x := item.(type) {
		case string:
			out = append(out, x)
		default:
			line, err := json.Marshal(x)
			if err != nil {
				out = append(out, fmt.Sprint(x))
			} else {
				out = append(out, string(line))
			}
		}
	}
	return out
}

func extractMigratedYAML(res *mcp.CallToolResult) (string, error) {
	raw, err := toolResultPayload(res)
	if err != nil {
		return "", err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("decode migrate tool output: %w", err)
	}
	y, ok := m["yaml"]
	if !ok {
		return "", fmt.Errorf("migrate tool output missing yaml field")
	}
	var s string
	if err := json.Unmarshal(y, &s); err != nil {
		return "", fmt.Errorf("decode yaml field: %w", err)
	}
	return s, nil
}
