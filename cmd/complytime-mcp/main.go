// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
		&mcp.Implementation{Name: "complytime-mcp", Version: "v0.4.0"},
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

func registerResources(s *mcp.Server, gw *gatewayClient) {
	addJSONResource(s, gw, "complytime://policies", "policies", "List all imported policies", "/api/policies")
	addJSONResource(s, gw, "complytime://catalogs", "catalogs", "List all imported catalogs", "/api/catalogs")
	addJSONResource(s, gw, "complytime://posture", "posture", "List compliance posture aggregates", "/api/posture")
	addJSONResource(s, gw, "complytime://audit-logs", "audit-logs", "List audit logs", "/api/audit-logs")
	addJSONResource(s, gw, "complytime://draft-audit-logs", "draft-audit-logs", "List draft audit logs pending review", "/api/draft-audit-logs")
	addJSONResource(s, gw, "complytime://threats", "threats", "List threat catalog entries", "/api/threats")
	addJSONResource(s, gw, "complytime://risks", "risks", "List risk catalog entries", "/api/risks")
	addJSONResource(s, gw, "complytime://certifications", "certifications", "List evidence certification results", "/api/certifications")
	addJSONResource(s, gw, "complytime://requirements", "requirements", "List assessment requirements", "/api/requirements")
	addJSONResource(s, gw, "complytime://control-threats", "control-threats", "List control-to-threat mappings", "/api/control-threats")
	addJSONResource(s, gw, "complytime://risk-threats", "risk-threats", "List risk-to-threat mappings", "/api/risk-threats")
	addJSONResource(s, gw, "complytime://inventory", "inventory", "List imported artifact inventory", "/api/inventory")

	addResourceTemplate(s, gw, "complytime://policies/{policy_id}", "policy", "Get a single policy with mappings", "complytime://policies/", "/api/policies/")
	addResourceTemplate(s, gw, "complytime://audit-logs/{audit_log_id}", "audit-log", "Get a single audit log", "complytime://audit-logs/", "/api/audit-logs/")
	addResourceTemplate(s, gw, "complytime://draft-audit-logs/{draft_id}", "draft-audit-log", "Get a single draft audit log", "complytime://draft-audit-logs/", "/api/draft-audit-logs/")
	addResourceTemplate(s, gw, "complytime://requirements/{requirement_id}/evidence", "requirement-evidence", "Get evidence for a specific requirement", "complytime://requirements/", "/api/requirements/")
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

func addResourceTemplate(s *mcp.Server, gw *gatewayClient, uriTemplate, name, desc, uriPrefix, apiPrefix string) {
	s.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: uriTemplate,
		Name:        name,
		Description: desc,
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		param := extractParam(req.Params.URI, uriPrefix)
		data, err := gw.get(ctx, apiPrefix+param)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})
}

type QueryEvidenceInput struct {
	PolicyID   string `json:"policy_id" jsonschema:"Filter by policy ID"`
	ControlID  string `json:"control_id" jsonschema:"Filter by control ID"`
	TargetType string `json:"target_type" jsonschema:"Filter by target type"`
	TargetID   string `json:"target_id" jsonschema:"Filter by target ID"`
	Start      string `json:"start" jsonschema:"Start of time range (RFC3339)"`
	End        string `json:"end" jsonschema:"End of time range (RFC3339)"`
	Limit      int    `json:"limit" jsonschema:"Max results to return"`
	Offset     int    `json:"offset" jsonschema:"Offset for pagination"`
}

type QueryEvidenceOutput struct {
	JSON string `json:"json"`
}

func registerTools(s *mcp.Server, gw *gatewayClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "query_evidence",
		Description: "Query evidence records filtered by policy, control, target, or time range",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input QueryEvidenceInput) (*mcp.CallToolResult, QueryEvidenceOutput, error) {
		params := url.Values{}
		if input.PolicyID != "" {
			params.Set("policy_id", input.PolicyID)
		}
		if input.ControlID != "" {
			params.Set("control_id", input.ControlID)
		}
		if input.TargetType != "" {
			params.Set("target_type", input.TargetType)
		}
		if input.TargetID != "" {
			params.Set("target_id", input.TargetID)
		}
		if input.Start != "" {
			params.Set("start", input.Start)
		}
		if input.End != "" {
			params.Set("end", input.End)
		}
		if input.Limit > 0 {
			params.Set("limit", strconv.Itoa(input.Limit))
		}
		if input.Offset > 0 {
			params.Set("offset", strconv.Itoa(input.Offset))
		}
		path := "/api/evidence"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
		data, err := gw.get(ctx, path)
		if err != nil {
			return nil, QueryEvidenceOutput{}, fmt.Errorf("query_evidence: %w", err)
		}
		return nil, QueryEvidenceOutput{JSON: string(data)}, nil
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
