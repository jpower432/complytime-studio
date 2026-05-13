// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	chclient "github.com/complytime/complytime-studio/internal/clickhouse"
	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/gemara"
	"github.com/complytime/complytime-studio/internal/store"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "studio-mcp"
	serverVersion = "0.1.0"
	jsonMIME      = "application/json"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	var (
		transport      = flag.String("transport", "stdio", "MCP transport: stdio or http")
		port           = flag.String("port", "3000", "listen port when transport=http")
		clickhouseAddr = flag.String("clickhouse-addr", "", "ClickHouse address (host:9000)")
		clickhouseUser = flag.String("clickhouse-user", "default", "ClickHouse user")
		clickhousePass = flag.String("clickhouse-password", "", "ClickHouse password")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	chAddr := strings.TrimSpace(*clickhouseAddr)
	if chAddr == "" {
		chAddr = strings.TrimSpace(os.Getenv("CLICKHOUSE_ADDR"))
	}
	if chAddr == "" {
		slog.Error("missing ClickHouse address (set --clickhouse-addr or CLICKHOUSE_ADDR)")
		os.Exit(2)
	}

	chPass := strings.TrimSpace(*clickhousePass)
	if chPass == "" {
		chPass = os.Getenv("CLICKHOUSE_PASSWORD")
	}

	chc, err := chclient.New(ctx, chclient.Config{
		Addr:     chAddr,
		User:     strings.TrimSpace(*clickhouseUser),
		Password: chPass,
	})
	if err != nil {
		slog.Error("clickhouse connection failed", "error", err)
		os.Exit(1)
	}
	if err := chc.EnsureSchema(ctx, 24); err != nil {
		slog.Error("clickhouse schema init failed", "error", err)
		os.Exit(1)
	}

	st := store.New(chc.Conn())
	srv := newStudioServer(st)

	switch strings.ToLower(strings.TrimSpace(*transport)) {
	case "stdio":
		slog.Info("studio-mcp listening on stdio")
		if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("server stopped", "error", err)
			os.Exit(1)
		}
	case "http":
		addr := ":" + strings.TrimSpace(*port)
		h := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server { return srv }, &mcp.StreamableHTTPOptions{
			Logger: slog.Default(),
		})
		httpSrv := &http.Server{Addr: addr, Handler: h}
		go func() {
			slog.Info("studio-mcp listening", "transport", "http", "addr", addr)
			if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("http server error", "error", err)
				os.Exit(1)
			}
		}()
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	default:
		slog.Error("invalid --transport", "value", *transport)
		os.Exit(2)
	}
}

func newStudioServer(st *store.Store) *mcp.Server {
	opts := &mcp.ServerOptions{
		Instructions: "ComplyTime Studio data access: policies, evidence, posture, audit logs, mappings, catalogs, threats, risks.",
		Logger:       slog.Default(),
	}
	s := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, opts)

	s.AddResource(&mcp.Resource{
		Name:        "policies",
		Title:       "All policies",
		URI:         "studio://policies",
		Description: "JSON array of policies (metadata columns only).",
		MIMEType:    jsonMIME,
	}, readPolicies(st))

	s.AddResource(&mcp.Resource{
		Name:        "catalogs",
		Title:       "Catalog index",
		URI:         "studio://catalogs",
		Description: "JSON array of catalog rows (no full content).",
		MIMEType:    jsonMIME,
	}, readCatalogs(st))

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "policy-by-id",
		Title:       "Single policy",
		URITemplate: "studio://policies/{id}",
		Description: "Full policy YAML/content by policy_id.",
		MIMEType:    jsonMIME,
	}, readPolicyByID(st))

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "evidence-query",
		Title:       "Evidence rows",
		URITemplate: "studio://evidence{?policy_id,limit,offset}",
		Description: "Evidence rows; limit defaults to 100, offset to 0.",
		MIMEType:    jsonMIME,
	}, readEvidence(st))

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "posture",
		Title:       "Compliance posture",
		URITemplate: "studio://posture{?policy_id}",
		Description: "Per-policy posture aggregates; optional policy_id filters client-side.",
		MIMEType:    jsonMIME,
	}, readPosture(st))

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "audit-logs",
		Title:       "Audit logs",
		URITemplate: "studio://audit-logs{?policy_id,limit}",
		Description: "Audit logs for a policy; policy_id required.",
		MIMEType:    jsonMIME,
	}, readAuditLogs(st))

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "mappings",
		Title:       "Mapping documents",
		URITemplate: "studio://mappings{?source_catalog}",
		Description: "Mapping documents; optional source_catalog filters Framework.",
		MIMEType:    jsonMIME,
	}, readMappings(st))

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "threats",
		Title:       "Threat catalog rows",
		URITemplate: "studio://threats{?catalog_id}",
		Description: "Threat rows; optional catalog_id filter.",
		MIMEType:    jsonMIME,
	}, readThreats(st))

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "risks",
		Title:       "Risk catalog rows",
		URITemplate: "studio://risks{?catalog_id}",
		Description: "Risk rows; optional catalog_id filter.",
		MIMEType:    jsonMIME,
	}, readRisks(st))

	registerTools(s, st)
	return s
}

func jsonResource(uri string, v any) (*mcp.ReadResourceResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: jsonMIME, Text: string(b)}},
	}, nil
}

func readPolicies(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		list, err := st.ListPolicies(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, list)
	}
}

func readPolicyByID(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		id, err := pathSegmentAfterHost(req.Params.URI, "policies")
		if err != nil {
			return nil, err
		}
		if id == "" {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		p, err := st.GetPolicy(ctx, id)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return jsonResource(req.Params.URI, p)
	}
}

func readEvidence(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		q, err := url.Parse(req.Params.URI)
		if err != nil {
			return nil, err
		}
		vs := q.Query()
		limit := consts.DefaultQueryLimit
		if v := vs.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				limit = consts.ClampLimit(n)
			}
		}
		offset := 0
		if v := vs.Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}
		recs, err := st.QueryEvidence(ctx, store.EvidenceFilter{
			PolicyID: vs.Get("policy_id"),
			Limit:    limit,
			Offset:   offset,
		})
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, recs)
	}
}

func readPosture(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		q, err := url.Parse(req.Params.URI)
		if err != nil {
			return nil, err
		}
		policyID := q.Query().Get("policy_id")
		rows, err := st.ListPosture(ctx, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if policyID != "" {
			filtered := rows[:0]
			for _, r := range rows {
				if r.PolicyID == policyID {
					filtered = append(filtered, r)
				}
			}
			rows = filtered
		}
		return jsonResource(req.Params.URI, rows)
	}
}

func readAuditLogs(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		q, err := url.Parse(req.Params.URI)
		if err != nil {
			return nil, err
		}
		policyID := q.Query().Get("policy_id")
		if policyID == "" {
			return nil, fmt.Errorf("policy_id query parameter is required")
		}
		limit := consts.ClampLimit(0)
		if v := q.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = consts.ClampLimit(n)
			}
		}
		logs, err := st.ListAuditLogs(ctx, policyID, time.Time{}, time.Time{}, limit)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, logs)
	}
}

func readMappings(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		q, err := url.Parse(req.Params.URI)
		if err != nil {
			return nil, err
		}
		src := q.Query().Get("source_catalog")
		all, err := st.ListAllMappings(ctx)
		if err != nil {
			return nil, err
		}
		if src != "" {
			filtered := all[:0]
			for _, m := range all {
				if m.Framework == src || strings.EqualFold(m.Framework, src) {
					filtered = append(filtered, m)
				}
			}
			all = filtered
		}
		return jsonResource(req.Params.URI, all)
	}
}

func readCatalogs(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		list, err := st.ListCatalogs(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, list)
	}
}

func readThreats(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		q, err := url.Parse(req.Params.URI)
		if err != nil {
			return nil, err
		}
		catalogID := q.Query().Get("catalog_id")
		limit := consts.ClampLimit(0)
		rows, err := st.QueryThreats(ctx, catalogID, "", limit)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, rows)
	}
}

func readRisks(st *store.Store) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		q, err := url.Parse(req.Params.URI)
		if err != nil {
			return nil, err
		}
		catalogID := q.Query().Get("catalog_id")
		limit := consts.ClampLimit(0)
		rows, err := st.QueryRisks(ctx, catalogID, "", limit)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, rows)
	}
}

func pathSegmentAfterHost(rawURI, hostName string) (string, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(u.Scheme, "studio") || u.Host != hostName {
		return "", fmt.Errorf("unexpected URI for policies resource")
	}
	return strings.Trim(strings.TrimPrefix(u.Path, "/"), " "), nil
}

func registerTools(s *mcp.Server, st *store.Store) {
	ingestSchema := json.RawMessage(`{
  "type": "object",
  "description": "Either a bare JSON array of evidence records or an object with a records array.",
  "properties": {
    "records": {
      "type": "array",
      "items": {"type": "object"}
    }
  }
}`)

	s.AddTool(&mcp.Tool{
		Name:        "ingest_evidence",
		Description: "Insert evidence rows into ClickHouse. Pass {\"records\":[...]} or a JSON array as arguments.",
		InputSchema: ingestSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		records, err := decodeEvidenceArgs(req.Params.Arguments)
		if err != nil {
			return toolErr(err), nil
		}
		if len(records) == 0 {
			return toolErr(errors.New("at least one evidence record is required")), nil
		}
		var valErrs []string
		for i, rec := range records {
			if rec.PolicyID == "" || rec.TargetID == "" || rec.ControlID == "" || rec.CollectedAt.IsZero() {
				valErrs = append(valErrs, fmt.Sprintf(
					"row %d: missing required fields (policy_id, target_id, control_id, collected_at)", i))
			}
		}
		if len(valErrs) > 0 {
			b, _ := json.Marshal(map[string]any{"errors": valErrs})
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}, IsError: true}, nil
		}
		n, err := st.InsertEvidence(ctx, records)
		if err != nil {
			return toolErr(err), nil
		}
		return structuredOK(map[string]int{"inserted": n})
	})

	type saveDraftIn struct {
		PolicyID       string `json:"policy_id"`
		YAML           string `json:"yaml"`
		AgentReasoning string `json:"agent_reasoning,omitempty"`
		Model          string `json:"model,omitempty"`
		PromptVersion  string `json:"prompt_version,omitempty"`
	}
	type saveDraftOut struct {
		Status  string `json:"status"`
		DraftID string `json:"draft_id"`
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "save_draft_audit_log",
		Description: "Parse Gemara audit log YAML and store a draft audit log pending human review.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in saveDraftIn) (*mcp.CallToolResult, saveDraftOut, error) {
		if strings.TrimSpace(in.PolicyID) == "" || strings.TrimSpace(in.YAML) == "" {
			return nil, saveDraftOut{}, fmt.Errorf("policy_id and yaml are required")
		}
		summary, parseErr := gemara.ParseAuditLog(in.YAML)
		if parseErr != nil {
			return nil, saveDraftOut{}, fmt.Errorf("invalid audit log yaml: %w", parseErr)
		}
		d := store.DraftAuditLog{
			PolicyID:       in.PolicyID,
			Content:        in.YAML,
			AuditStart:     summary.AuditStart,
			AuditEnd:       summary.AuditEnd,
			Framework:      summary.Framework,
			AgentReasoning: in.AgentReasoning,
			Summary: fmt.Sprintf(
				`{"strengths":%d,"findings":%d,"gaps":%d,"observations":%d}`,
				summary.Strengths, summary.Findings, summary.Gaps, summary.Observations,
			),
			Model:         in.Model,
			PromptVersion: in.PromptVersion,
		}
		d.DraftID = uuid.New().String()
		if err := st.InsertDraftAuditLog(ctx, d); err != nil {
			return nil, saveDraftOut{}, err
		}
		return nil, saveDraftOut{Status: "drafted", DraftID: d.DraftID}, nil
	})
}

func decodeEvidenceArgs(raw json.RawMessage) ([]store.EvidenceRecord, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing arguments")
	}
	var records []store.EvidenceRecord
	if err := json.Unmarshal(raw, &records); err == nil && records != nil {
		return records, nil
	}
	var wrap struct {
		Records []store.EvidenceRecord `json:"records"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, fmt.Errorf("decode arguments: %w", err)
	}
	return wrap.Records, nil
}

func toolErr(err error) *mcp.CallToolResult {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
		IsError: true,
	}
}

func structuredOK(v any) (*mcp.CallToolResult, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var structured any
	if err := json.Unmarshal(raw, &structured); err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		StructuredContent: structured,
		Content:           []mcp.Content{&mcp.TextContent{Text: string(raw)}},
	}, nil
}
