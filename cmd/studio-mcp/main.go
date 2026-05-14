// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	studiov1 "github.com/complytime/complytime-studio/gen/studio/v1"
	"github.com/complytime/complytime-studio/gen/studio/v1/studiov1connect"
)

func main() {
	gatewayURL := os.Getenv("GATEWAY_INTERNAL_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8081"
	}

	client := studiov1connect.NewStudioServiceClient(http.DefaultClient, gatewayURL)

	server := mcp.NewServer(
		&mcp.Implementation{Name: "studio-mcp", Version: "v0.1.0"},
		nil,
	)

	registerResources(server, client)
	registerTools(server, client)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func registerResources(s *mcp.Server, client studiov1connect.StudioServiceClient) {
	s.AddResource(&mcp.Resource{
		URI:         "studio://policies",
		Name:        "policies",
		Description: "List all imported policies",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		resp, err := client.ListPolicies(ctx, connect.NewRequest(&studiov1.ListPoliciesRequest{}))
		if err != nil {
			return nil, fmt.Errorf("ListPolicies: %w", err)
		}
		data, _ := json.Marshal(resp.Msg.Policies)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "studio://policies/{policy_id}",
		Name:        "policy",
		Description: "Get a single policy with mappings",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		policyID := extractParam(req.Params.URI, "studio://policies/")
		resp, err := client.GetPolicy(ctx, connect.NewRequest(&studiov1.GetPolicyRequest{PolicyId: policyID}))
		if err != nil {
			return nil, fmt.Errorf("GetPolicy %q: %w", policyID, err)
		}
		data, _ := json.Marshal(resp.Msg)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})

	s.AddResource(&mcp.Resource{
		URI:         "studio://catalogs",
		Name:        "catalogs",
		Description: "List all imported catalogs",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		resp, err := client.ListCatalogs(ctx, connect.NewRequest(&studiov1.ListCatalogsRequest{}))
		if err != nil {
			return nil, fmt.Errorf("ListCatalogs: %w", err)
		}
		data, _ := json.Marshal(resp.Msg.Catalogs)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})

	s.AddResource(&mcp.Resource{
		URI:         "studio://posture",
		Name:        "posture",
		Description: "List compliance posture aggregates for all policies",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		resp, err := client.ListPosture(ctx, connect.NewRequest(&studiov1.ListPostureRequest{}))
		if err != nil {
			return nil, fmt.Errorf("ListPosture: %w", err)
		}
		data, _ := json.Marshal(resp.Msg.Rows)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})

	s.AddResource(&mcp.Resource{
		URI:         "studio://audit-logs",
		Name:        "audit-logs",
		Description: "List audit logs",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		resp, err := client.ListAuditLogs(ctx, connect.NewRequest(&studiov1.ListAuditLogsRequest{}))
		if err != nil {
			return nil, fmt.Errorf("ListAuditLogs: %w", err)
		}
		data, _ := json.Marshal(resp.Msg.AuditLogs)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})

	s.AddResource(&mcp.Resource{
		URI:         "studio://threats",
		Name:        "threats",
		Description: "List threat catalog entries",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		resp, err := client.ListThreats(ctx, connect.NewRequest(&studiov1.ListThreatsRequest{}))
		if err != nil {
			return nil, fmt.Errorf("ListThreats: %w", err)
		}
		data, _ := json.Marshal(resp.Msg.Threats)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})

	s.AddResource(&mcp.Resource{
		URI:         "studio://risks",
		Name:        "risks",
		Description: "List risk catalog entries",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		resp, err := client.ListRisks(ctx, connect.NewRequest(&studiov1.ListRisksRequest{}))
		if err != nil {
			return nil, fmt.Errorf("ListRisks: %w", err)
		}
		data, _ := json.Marshal(resp.Msg.Risks)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})

	s.AddResource(&mcp.Resource{
		URI:         "studio://mappings",
		Name:        "mappings",
		Description: "List cross-framework mapping documents",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		resp, err := client.ListMappings(ctx, connect.NewRequest(&studiov1.ListMappingsRequest{}))
		if err != nil {
			return nil, fmt.Errorf("ListMappings: %w", err)
		}
		data, _ := json.Marshal(resp.Msg.Mappings)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{textResource(req.Params.URI, data)},
		}, nil
	})
}

type IngestEvidenceInput struct {
	YAMLContent string `json:"yaml_content" jsonschema:"Gemara EvaluationLog or EnforcementLog YAML content"`
}

type IngestEvidenceOutput struct {
	Inserted int32  `json:"inserted"`
	PolicyID string `json:"policy_id"`
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
	PolicyIDs  []string `json:"policy_ids" jsonschema:"Filter by one or more policy IDs"`
	ControlID  string   `json:"control_id" jsonschema:"Filter by control ID"`
	TargetName string   `json:"target_name" jsonschema:"Filter by target name"`
	Limit      int      `json:"limit" jsonschema:"Max results to return"`
}

type QueryEvidenceOutput struct {
	JSON string `json:"json"`
}

func registerTools(s *mcp.Server, client studiov1connect.StudioServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ingest_evidence",
		Description: "Ingest Gemara EvaluationLog or EnforcementLog YAML into the data platform",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input IngestEvidenceInput) (*mcp.CallToolResult, IngestEvidenceOutput, error) {
		resp, err := client.IngestEvidence(ctx, connect.NewRequest(&studiov1.IngestEvidenceRequest{
			YamlContent: input.YAMLContent,
		}))
		if err != nil {
			return nil, IngestEvidenceOutput{}, fmt.Errorf("IngestEvidence: %w", err)
		}
		return nil, IngestEvidenceOutput{
			Inserted: resp.Msg.Inserted,
			PolicyID: resp.Msg.PolicyId,
		}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "save_draft_audit_log",
		Description: "Save an agent-produced draft audit log for human review",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input SaveDraftAuditLogInput) (*mcp.CallToolResult, SaveDraftAuditLogOutput, error) {
		resp, err := client.CreateDraftAuditLog(ctx, connect.NewRequest(&studiov1.CreateDraftAuditLogRequest{
			PolicyId:       input.PolicyID,
			Content:        input.Content,
			Summary:        input.Summary,
			AgentReasoning: input.AgentReasoning,
			Model:          input.Model,
			PromptVersion:  input.PromptVersion,
		}))
		if err != nil {
			return nil, SaveDraftAuditLogOutput{}, fmt.Errorf("CreateDraftAuditLog: %w", err)
		}
		return nil, SaveDraftAuditLogOutput{DraftID: resp.Msg.DraftId}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "query_evidence",
		Description: "Query evidence records filtered by policy, control, target, or time range",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input QueryEvidenceInput) (*mcp.CallToolResult, QueryEvidenceOutput, error) {
		resp, err := client.QueryEvidence(ctx, connect.NewRequest(&studiov1.QueryEvidenceRequest{
			PolicyIds:  input.PolicyIDs,
			ControlId:  input.ControlID,
			TargetName: input.TargetName,
			Limit:      int32(input.Limit),
		}))
		if err != nil {
			return nil, QueryEvidenceOutput{}, fmt.Errorf("QueryEvidence: %w", err)
		}
		data, _ := json.Marshal(resp.Msg.Records)
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
