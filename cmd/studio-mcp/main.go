// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}
	gatewayURL = strings.TrimRight(gatewayURL, "/")

	identity := os.Getenv("MCP_IDENTITY")
	if identity == "" {
		identity = "studio-mcp@complytime.dev"
	}

	gw := &gatewayClient{baseURL: gatewayURL, identity: identity}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "studio-mcp", Version: "v0.2.0"},
		nil,
	)

	registerResources(server, gw)
	registerTools(server, gw)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

type gatewayClient struct {
	baseURL  string
	identity string
}

func (g *gatewayClient) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Forwarded-Email", g.identity)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s: %d %s", path, resp.StatusCode, string(body))
	}
	return body, nil
}

func (g *gatewayClient) post(ctx context.Context, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Email", g.identity)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("POST %s: %d %s", path, resp.StatusCode, string(body))
	}
	return body, nil
}

func registerResources(s *mcp.Server, gw *gatewayClient) {
	addJSONResource(s, gw, "studio://policies", "policies", "List all imported policies", "/api/policies")
	addJSONResource(s, gw, "studio://catalogs", "catalogs", "List all imported catalogs", "/api/catalogs")
	addJSONResource(s, gw, "studio://posture", "posture", "List compliance posture aggregates", "/api/posture")
	addJSONResource(s, gw, "studio://audit-logs", "audit-logs", "List audit logs", "/api/audit-logs")
	addJSONResource(s, gw, "studio://threats", "threats", "List threat catalog entries", "/api/threats")
	addJSONResource(s, gw, "studio://risks", "risks", "List risk catalog entries", "/api/risks")
	addJSONResource(s, gw, "studio://mappings", "mappings", "List cross-framework mapping documents", "/api/mappings")

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "studio://policies/{policy_id}",
		Name:        "policy",
		Description: "Get a single policy with mappings",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		policyID := extractParam(req.Params.URI, "studio://policies/")
		data, err := gw.get(ctx, "/api/policies/"+policyID)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})
}

func addJSONResource(s *mcp.Server, gw *gatewayClient, uri, name, desc, path string) {
	s.AddResource(&mcp.Resource{
		URI:         uri,
		Name:        name,
		Description: desc,
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		data, err := gw.get(ctx, path)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})
}

type SaveDraftAuditLogInput struct {
	PolicyID       string `json:"policy_id" jsonschema:"Policy ID the audit covers"`
	Content        string `json:"content" jsonschema:"Full audit log content (YAML or markdown)"`
	Summary        string `json:"summary" jsonschema:"One-line summary of the audit"`
	AgentReasoning string `json:"agent_reasoning" jsonschema:"Agent reasoning trace"`
	Model          string `json:"model" jsonschema:"Model that produced the draft"`
	PromptVersion  string `json:"prompt_version" jsonschema:"Prompt version used"`
}

type SaveDraftAuditLogOutput struct {
	DraftID string `json:"draft_id"`
}

type QueryEvidenceInput struct {
	PolicyID string `json:"policy_id" jsonschema:"Filter by policy ID"`
	Limit    int    `json:"limit" jsonschema:"Max results to return"`
}

type QueryEvidenceOutput struct {
	JSON string `json:"json"`
}

func registerTools(s *mcp.Server, gw *gatewayClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "query_evidence",
		Description: "Query evidence records filtered by policy, control, target, or time range",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input QueryEvidenceInput) (*mcp.CallToolResult, QueryEvidenceOutput, error) {
		path := "/api/evidence"
		if input.PolicyID != "" {
			path += "?policy_id=" + input.PolicyID
		}
		data, err := gw.get(ctx, path)
		if err != nil {
			return nil, QueryEvidenceOutput{}, fmt.Errorf("query_evidence: %w", err)
		}
		return nil, QueryEvidenceOutput{JSON: string(data)}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "save_draft_audit_log",
		Description: "Save an agent-produced draft audit log for human review",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input SaveDraftAuditLogInput) (*mcp.CallToolResult, SaveDraftAuditLogOutput, error) {
		body, err := gw.post(ctx, "/api/draft-audit-logs", map[string]string{
			"policy_id":       input.PolicyID,
			"content":         input.Content,
			"summary":         input.Summary,
			"agent_reasoning": input.AgentReasoning,
			"model":           input.Model,
			"prompt_version":  input.PromptVersion,
		})
		if err != nil {
			return nil, SaveDraftAuditLogOutput{}, fmt.Errorf("save_draft_audit_log: %w", err)
		}
		var out SaveDraftAuditLogOutput
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, SaveDraftAuditLogOutput{}, fmt.Errorf("parse response: %w", err)
		}
		return nil, out, nil
	})
}

func textResource(uri string, data []byte) *mcp.ResourceContents {
	return &mcp.ResourceContents{
		URI:      uri,
		MIMEType: "application/json",
		Text:     string(data),
	}
}

func extractParam(uri, prefix string) string {
	if len(uri) > len(prefix) {
		return uri[len(prefix):]
	}
	return ""
}
