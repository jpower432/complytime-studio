// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/complytime/complytime-studio/internal/blob"
	"github.com/complytime/complytime-studio/internal/consts"
	gemarapkg "github.com/complytime/complytime-studio/internal/gemara"
	"github.com/complytime/complytime-studio/internal/httputil"
)

// EventPublisher emits NATS events for evidence and draft audit logs.
// Implemented by *events.Bus; nil-safe (callers check before use).
type EventPublisher interface {
	PublishEvidence(policyID string, count int)
	PublishDraftAuditLog(draftID, policyID, summary string)
}

// HealthChecker verifies backend connectivity for health probes.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// Stores groups all domain store interfaces for handler registration.
type Stores struct {
	Policies            PolicyStore
	Mappings            MappingStore
	Evidence            EvidenceStore
	Blob                blob.BlobStore
	AuditLogs           AuditLogStore
	DraftAuditLogs      DraftAuditLogStore
	Requirements        RequirementStore
	Controls            ControlStore
	Threats             ThreatStore
	Risks               RiskStore
	Catalogs            CatalogStore
	EvidenceAssessments EvidenceAssessmentStore
	Posture             PostureStore
	Notifications       NotificationStore
	Certifications      CertificationStore
	EventPublisher      EventPublisher
	HealthChecker       HealthChecker
}

// Register mounts all public store API endpoints on g (typically e.Group("/api")).
// Internal (agent-only) endpoints are registered via RegisterInternal.
func Register(g *echo.Group, s Stores) {
	g.GET("/policies", listPoliciesHandler(s.Policies))
	g.GET("/policies/:id", getPolicyHandler(s.Policies, s.Mappings))
	g.POST("/policies/import", importPolicyHandler(s.Policies, s.Controls))
	g.POST("/mappings/import", importMappingHandler(s.Mappings))
	g.GET("/evidence", queryEvidenceHandler(s.Evidence))
	g.POST("/evidence/ingest", echo.WrapHandler(IngestGemaraHandler(s.Evidence, s.EventPublisher)))
	if s.Certifications != nil {
		g.GET("/certifications", queryCertificationsHandler(s.Certifications))
	}
	g.GET("/audit-logs/:id", getAuditLogHandler(s.AuditLogs))
	g.GET("/audit-logs", listAuditLogsHandler(s.AuditLogs))
	g.POST("/audit-logs", createAuditLogHandler(s.AuditLogs))
	g.POST("/catalogs/import", importCatalogHandler(s.Catalogs, s.Controls, s.Threats, s.Risks))
	if s.Posture != nil {
		g.GET("/posture", listPostureHandler(s.Posture))
	}
	if s.Requirements != nil {
		g.GET("/requirements", listRequirementMatrixHandler(s.Requirements))
		g.GET("/requirements/:id/evidence", listRequirementEvidenceHandler(s.Requirements))
	}
	if s.DraftAuditLogs != nil {
		g.GET("/draft-audit-logs", listDraftAuditLogsHandler(s.DraftAuditLogs))
		g.GET("/draft-audit-logs/:id", getDraftAuditLogHandler(s.DraftAuditLogs))
		g.PATCH("/draft-audit-logs/:id", updateDraftEditsHandler(s.DraftAuditLogs))
		g.POST("/audit-logs/promote", promoteAuditLogHandler(s.DraftAuditLogs))
	}
	if s.Threats != nil {
		g.GET("/threats", listThreatsHandler(s.Threats))
		g.GET("/control-threats", listControlThreatsHandler(s.Threats))
	}
	if s.Risks != nil {
		g.GET("/risks", listRisksHandler(s.Risks))
		g.GET("/risks/severity", riskSeverityHandler(s.Risks))
		g.GET("/risk-threats", listRiskThreatsHandler(s.Risks))
	}
	if s.Notifications != nil {
		g.GET("/notifications", listNotificationsHandler(s.Notifications))
		g.GET("/notifications/unread-count", unreadCountHandler(s.Notifications))
		g.PATCH("/notifications/:id/read", markReadHandler(s.Notifications))
		g.POST("/notifications", createNotificationHandler(s.Notifications))
	}
}

// RegisterInternal mounts agent-only endpoints. Pass root = e.Group("") so
// GET /healthz and POST /internal/draft-audit-logs match the prior ServeMux layout.
// See docs/decisions/internal-endpoint-isolation.md.
func RegisterInternal(root *echo.Group, s Stores) {
	root.GET("/healthz", func(c echo.Context) error {
		if s.HealthChecker != nil {
			if err := s.HealthChecker.Ping(c.Request().Context()); err != nil {
				return c.String(http.StatusServiceUnavailable, "postgres unreachable")
			}
		}
		return c.String(http.StatusOK, "ok")
	})
	if s.DraftAuditLogs != nil {
		ig := root.Group("/internal")
		ig.POST("/draft-audit-logs", createDraftAuditLogHandler(s.DraftAuditLogs, s.EventPublisher))
	}
}

func listPoliciesHandler(s PolicyStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		policies, err := s.ListPolicies(c.Request().Context())
		if err != nil {
			slog.Error("list policies failed", "error", err)
			return c.String(http.StatusInternalServerError, "internal error")
		}
		if policies == nil {
			policies = []Policy{}
		}
		return c.JSON(http.StatusOK, policies)
	}
}

func getPolicyHandler(ps PolicyStore, ms MappingStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return c.String(http.StatusBadRequest, "missing policy id")
		}
		p, err := ps.GetPolicy(c.Request().Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.String(http.StatusNotFound, "not found")
			}
			slog.Error("get policy failed", "error", err, "id", id)
			return c.String(http.StatusInternalServerError, "internal server error")
		}
		mappings, _ := ms.ListMappings(c.Request().Context(), id)
		if mappings == nil {
			mappings = []MappingDocument{}
		}
		resp := struct {
			Policy   *Policy           `json:"policy"`
			Mappings []MappingDocument `json:"mappings"`
		}{Policy: p, Mappings: mappings}
		return c.JSON(http.StatusOK, resp)
	}
}

func importPolicyHandler(ps PolicyStore, ctrlS ControlStore) echo.HandlerFunc {
	type importReq struct {
		OCIReference string `json:"oci_reference"`
		Content      string `json:"content"`
		Title        string `json:"title"`
		Version      string `json:"version"`
		PolicyID     string `json:"policy_id"`
	}
	type importResp struct {
		Status   string `json:"status"`
		PolicyID string `json:"policy_id"`
	}
	return func(c echo.Context) error {
		var req importReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
		}
		if req.Content == "" || req.Title == "" {
			return c.String(http.StatusBadRequest, "content and title required")
		}
		p := Policy{
			PolicyID:     req.PolicyID,
			Title:        req.Title,
			Version:      req.Version,
			OCIReference: req.OCIReference,
			Content:      req.Content,
		}
		if err := ps.InsertPolicy(c.Request().Context(), p); err != nil {
			slog.Error("insert policy failed", "error", err)
			return c.String(http.StatusInternalServerError, "insert failed")
		}

		resp := importResp{Status: "imported", PolicyID: p.PolicyID}

		if ctrlS != nil {
			if _, err := ExtractPolicyCriteria(c.Request().Context(), p.PolicyID, req.Content, ctrlS); err != nil {
				slog.Warn("inline criteria extraction failed", "policy_id", p.PolicyID, "error", err)
			}
		}

		return c.JSON(http.StatusCreated, resp)
	}
}

func importMappingHandler(s MappingStore) echo.HandlerFunc {
	type importReq struct {
		MappingID       string `json:"mapping_id"`
		SourceCatalogID string `json:"source_catalog_id"`
		TargetCatalogID string `json:"target_catalog_id"`
		Framework       string `json:"framework"`
		Content         string `json:"content"`
	}
	return func(c echo.Context) error {
		var req importReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
		}
		if req.SourceCatalogID == "" || req.TargetCatalogID == "" || req.Content == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "source_catalog_id, target_catalog_id, and content required",
			})
		}
		m := MappingDocument{
			MappingID:       req.MappingID,
			SourceCatalogID: req.SourceCatalogID,
			TargetCatalogID: req.TargetCatalogID,
			Framework:       req.Framework,
			Content:         req.Content,
		}
		if err := s.InsertMapping(c.Request().Context(), m); err != nil {
			slog.Error("insert mapping failed", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		}

		entries, parseErr := gemarapkg.ParseMappingEntries(
			req.Content, m.MappingID, req.SourceCatalogID, req.TargetCatalogID, req.Framework,
		)
		if parseErr != nil {
			slog.Warn("mapping YAML parse failed, structured entries skipped",
				"mapping_id", m.MappingID, "error", parseErr)
		} else if len(entries) > 0 {
			if err := s.InsertMappingEntries(c.Request().Context(), entries); err != nil {
				slog.Warn("insert mapping entries failed",
					"mapping_id", m.MappingID, "error", err)
			} else {
				slog.Info("mapping entries stored",
					"mapping_id", m.MappingID, "count", len(entries))
			}
		}

		return c.JSON(http.StatusCreated, map[string]string{"status": "imported", "mapping_id": m.MappingID})
	}
}

func queryCertificationsHandler(s CertificationStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		evidenceID := c.QueryParam("evidence_id")
		if evidenceID == "" {
			return c.String(http.StatusBadRequest, "evidence_id required")
		}
		rows, err := s.QueryCertifications(c.Request().Context(), evidenceID)
		if err != nil {
			slog.Error("query certifications failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []CertificationRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func queryEvidenceHandler(s EvidenceStore) echo.HandlerFunc {
	const maxPolicyIDs = 50

	return func(c echo.Context) error {
		f := EvidenceFilter{
			ControlID:     c.QueryParam("control_id"),
			TargetName:    c.QueryParam("target_name"),
			TargetType:    c.QueryParam("target_type"),
			TargetEnv:     c.QueryParam("target_env"),
			EngineVersion: c.QueryParam("engine_version"),
			Owner:         c.QueryParam("owner"),
		}
		if policyID := c.QueryParam("policy_id"); policyID != "" {
			f.PolicyIDs = []string{policyID}
		}
		if pids := c.QueryParam("policy_ids"); pids != "" {
			for _, p := range strings.Split(pids, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					f.PolicyIDs = append(f.PolicyIDs, p)
				}
			}
		}
		if len(f.PolicyIDs) > maxPolicyIDs {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("too many policy_ids (max %d)", maxPolicyIDs),
			})
		}
		if v := c.QueryParam("start"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				f.Start = t
			} else if t, err := time.Parse("2006-01-02", v); err == nil {
				f.Start = t
			}
		}
		if v := c.QueryParam("end"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				f.End = t
			} else if t, err := time.Parse("2006-01-02", v); err == nil {
				f.End = t
			}
		}
		if v := c.QueryParam("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				f.Limit = n
			}
		}
		if v := c.QueryParam("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				f.Offset = n
			}
		}
		f.Limit = consts.ClampLimit(f.Limit)

		records, err := s.QueryEvidence(c.Request().Context(), f)
		if err != nil {
			slog.Error("query evidence failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if records == nil {
			records = []EvidenceRecord{}
		}
		return c.JSON(http.StatusOK, records)
	}
}

func getAuditLogHandler(s AuditLogStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return c.String(http.StatusBadRequest, "missing audit id")
		}
		a, err := s.GetAuditLog(c.Request().Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.String(http.StatusNotFound, "not found")
			}
			slog.Error("get audit log failed", "error", err, "id", id)
			return c.String(http.StatusInternalServerError, "internal server error")
		}
		return c.JSON(http.StatusOK, a)
	}
}

func listAuditLogsHandler(s AuditLogStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		policyID := c.QueryParam("policy_id")
		if policyID == "" {
			return c.String(http.StatusBadRequest, "policy_id required")
		}
		var start, end time.Time
		if v := c.QueryParam("start"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				start = t
			}
		}
		if v := c.QueryParam("end"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				end = t
			}
		}
		limit := consts.ClampLimit(0)
		if v := c.QueryParam("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = consts.ClampLimit(n)
			}
		}

		logs, err := s.ListAuditLogs(c.Request().Context(), policyID, start, end, limit)
		if err != nil {
			slog.Error("list audit logs failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if logs == nil {
			logs = []AuditLog{}
		}
		return c.JSON(http.StatusOK, logs)
	}
}

func createAuditLogHandler(s AuditLogStore) echo.HandlerFunc {
	type createReq struct {
		PolicyID      string `json:"policy_id"`
		Content       string `json:"content"`
		Model         string `json:"model,omitempty"`
		PromptVersion string `json:"prompt_version,omitempty"`
	}
	return func(c echo.Context) error {
		var req createReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
		}
		if req.PolicyID == "" || req.Content == "" {
			return c.String(http.StatusBadRequest, "policy_id and content required")
		}

		summary, parseErr := gemarapkg.ParseAuditLog(req.Content)
		if parseErr != nil {
			slog.Warn("audit log YAML parse failed", "policy_id", req.PolicyID, "error", parseErr)
			return c.String(http.StatusBadRequest, fmt.Sprintf("invalid audit log content: %v", parseErr))
		}

		a := AuditLog{
			PolicyID:   req.PolicyID,
			Content:    req.Content,
			AuditStart: summary.AuditStart,
			AuditEnd:   summary.AuditEnd,
			Framework:  summary.Framework,
			Summary: fmt.Sprintf(
				`{"strengths":%d,"findings":%d,"gaps":%d,"observations":%d}`,
				summary.Strengths, summary.Findings, summary.Gaps, summary.Observations,
			),
			Model:         req.Model,
			PromptVersion: req.PromptVersion,
		}

		if err := s.InsertAuditLog(c.Request().Context(), a); err != nil {
			slog.Error("insert audit log failed", "error", err)
			return c.String(http.StatusInternalServerError, "insert failed")
		}
		return c.JSON(http.StatusCreated, map[string]string{"status": "stored", "audit_id": a.AuditID})
	}
}

func importCatalogHandler(cs CatalogStore, ctrlS ControlStore, threatS ThreatStore, riskS RiskStore) echo.HandlerFunc {
	type importReq struct {
		CatalogID string `json:"catalog_id"`
		PolicyID  string `json:"policy_id"`
		Content   string `json:"content"`
	}
	return func(c echo.Context) error {
		var req importReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
		}
		if req.Content == "" {
			return c.String(http.StatusBadRequest, "content required")
		}

		catalogType, title := detectCatalogType(req.Content)
		if catalogType == "" {
			return c.String(http.StatusBadRequest,
				"could not detect catalog type from content (expected ControlCatalog, ThreatCatalog, or RiskCatalog)")
		}

		catalogID := req.CatalogID
		if catalogID == "" {
			catalogID = detectCatalogID(req.Content)
		}

		if cs != nil {
			if err := cs.InsertCatalog(c.Request().Context(), Catalog{
				CatalogID:   catalogID,
				CatalogType: catalogType,
				Title:       title,
				Content:     req.Content,
				PolicyID:    req.PolicyID,
			}); err != nil {
				slog.Error("insert catalog failed", "error", err)
				return c.String(http.StatusInternalServerError, "insert failed")
			}
		}

		parseCatalogStructuredRows(c.Request().Context(), catalogType, req.Content, catalogID, req.PolicyID, ctrlS, threatS, riskS)

		return c.JSON(http.StatusCreated, map[string]string{
			"status":       "imported",
			"catalog_id":   catalogID,
			"catalog_type": catalogType,
		})
	}
}

func parseCatalogStructuredRows(ctx context.Context, catalogType, content, catalogID, policyID string, ctrlS ControlStore, threatS ThreatStore, riskS RiskStore) {
	switch catalogType {
	case "ControlCatalog":
		if ctrlS == nil {
			return
		}
		controls, reqs, threats, err := gemarapkg.ParseControlCatalog(ctx, content, catalogID, policyID)
		if err != nil {
			slog.Warn("control catalog parse failed, structured rows skipped", "catalog_id", catalogID, "error", err)
			return
		}
		if len(controls) > 0 {
			if err := ctrlS.InsertControls(ctx, controls); err != nil {
				slog.Warn("insert controls failed", "catalog_id", catalogID, "error", err)
			}
		}
		if len(reqs) > 0 {
			if err := ctrlS.InsertAssessmentRequirements(ctx, reqs); err != nil {
				slog.Warn("insert assessment requirements failed", "catalog_id", catalogID, "error", err)
			}
		}
		if len(threats) > 0 {
			if err := ctrlS.InsertControlThreats(ctx, threats); err != nil {
				slog.Warn("insert control threats failed", "catalog_id", catalogID, "error", err)
			}
		}
		slog.Info("control catalog indexed", "catalog_id", catalogID, "controls", len(controls), "requirements", len(reqs), "control_threats", len(threats))

	case "ThreatCatalog":
		if threatS == nil {
			return
		}
		rows, err := gemarapkg.ParseThreatCatalog(ctx, content, catalogID, policyID)
		if err != nil {
			slog.Warn("threat catalog parse failed, structured rows skipped", "catalog_id", catalogID, "error", err)
			return
		}
		if len(rows) > 0 {
			if err := threatS.InsertThreats(ctx, rows); err != nil {
				slog.Warn("insert threats failed", "catalog_id", catalogID, "error", err)
			}
		}
		slog.Info("threat catalog indexed", "catalog_id", catalogID, "threats", len(rows))

	case "RiskCatalog":
		if riskS == nil {
			return
		}
		riskRows, linkRows, err := gemarapkg.ParseRiskCatalog(ctx, content, catalogID, policyID)
		if err != nil {
			slog.Warn("risk catalog parse failed, structured rows skipped", "catalog_id", catalogID, "error", err)
			return
		}
		if len(riskRows) > 0 {
			if err := riskS.InsertRisks(ctx, riskRows); err != nil {
				slog.Warn("insert risks failed", "catalog_id", catalogID, "error", err)
			}
		}
		if len(linkRows) > 0 {
			if err := riskS.InsertRiskThreats(ctx, linkRows); err != nil {
				slog.Warn("insert risk threats failed", "catalog_id", catalogID, "error", err)
			}
		}
		slog.Info("risk catalog indexed", "catalog_id", catalogID, "risks", len(riskRows), "risk_threats", len(linkRows))
	}
}

func detectCatalogType(content string) (catalogType, title string) {
	var meta struct {
		Title    string `yaml:"title"`
		Metadata struct {
			Type string `yaml:"type"`
		} `yaml:"metadata"`
	}
	if err := gemarapkg.UnmarshalYAML([]byte(content), &meta); err != nil {
		return "", ""
	}
	switch meta.Metadata.Type {
	case "ControlCatalog", "ThreatCatalog", "RiskCatalog", "GuidanceCatalog":
		return meta.Metadata.Type, meta.Title
	default:
		return "", ""
	}
}

func detectCatalogID(content string) string {
	var meta struct {
		Metadata struct {
			ID string `yaml:"id"`
		} `yaml:"metadata"`
	}
	if err := gemarapkg.UnmarshalYAML([]byte(content), &meta); err != nil {
		return ""
	}
	return meta.Metadata.ID
}

func listRequirementMatrixHandler(s RequirementStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		policyID := c.QueryParam("policy_id")
		if policyID == "" {
			return c.String(http.StatusBadRequest, "policy_id required")
		}

		f := RequirementFilter{PolicyID: policyID}

		if v := c.QueryParam("audit_start"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err != nil {
				return c.String(http.StatusBadRequest, "invalid audit_start format")
			}
			f.Start = t
		}
		if v := c.QueryParam("audit_end"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err != nil {
				return c.String(http.StatusBadRequest, "invalid audit_end format")
			}
			f.End = t
		}
		if !f.Start.IsZero() && !f.End.IsZero() && f.End.Before(f.Start) {
			return c.String(http.StatusBadRequest, "audit_end must be >= audit_start")
		}

		f.Classification = c.QueryParam("classification")
		f.ControlFamily = c.QueryParam("control_family")

		if v := c.QueryParam("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Limit = n
			}
		}
		if v := c.QueryParam("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Offset = n
			}
		}
		f.Limit = consts.ClampLimit(f.Limit)

		rows, err := s.ListRequirementMatrix(c.Request().Context(), f)
		if err != nil {
			slog.Error("list requirement matrix failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []RequirementRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listRequirementEvidenceHandler(s RequirementStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqID := c.Param("id")
		if reqID == "" {
			return c.String(http.StatusBadRequest, "missing requirement id")
		}
		policyID := c.QueryParam("policy_id")
		if policyID == "" {
			return c.String(http.StatusBadRequest, "policy_id required")
		}

		f := RequirementFilter{PolicyID: policyID}
		if v := c.QueryParam("audit_start"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err == nil {
				f.Start = t
			}
		}
		if v := c.QueryParam("audit_end"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err == nil {
				f.End = t
			}
		}
		if v := c.QueryParam("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Limit = n
			}
		}
		if v := c.QueryParam("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Offset = n
			}
		}
		f.Limit = consts.ClampLimit(f.Limit)

		rows, err := s.ListRequirementEvidence(c.Request().Context(), reqID, f)
		if err != nil {
			if errors.Is(err, ErrRequirementNotFound) {
				return c.String(http.StatusNotFound, "not found")
			}
			slog.Error("list requirement evidence failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []RequirementEvidenceRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

// createDraftAuditLogHandler handles POST /internal/draft-audit-logs.
// No auth required — cluster-internal only, restricted by NetworkPolicy.
func createDraftAuditLogHandler(s DraftAuditLogStore, pub EventPublisher) echo.HandlerFunc {
	type createReq struct {
		PolicyID       string `json:"policy_id"`
		Content        string `json:"content"`
		AgentReasoning string `json:"agent_reasoning,omitempty"`
		Model          string `json:"model,omitempty"`
		PromptVersion  string `json:"prompt_version,omitempty"`
	}
	return func(c echo.Context) error {
		var req createReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
		}
		if req.PolicyID == "" || req.Content == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "policy_id and content required"})
		}

		summary, parseErr := gemarapkg.ParseAuditLog(req.Content)
		if parseErr != nil {
			slog.Warn("draft audit log YAML parse failed", "policy_id", req.PolicyID, "error", parseErr)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid audit log content: %v", parseErr)})
		}

		d := DraftAuditLog{
			DraftID:        uuid.New().String(),
			PolicyID:       req.PolicyID,
			Content:        req.Content,
			AuditStart:     summary.AuditStart,
			AuditEnd:       summary.AuditEnd,
			Framework:      summary.Framework,
			AgentReasoning: req.AgentReasoning,
			Summary: fmt.Sprintf(
				`{"strengths":%d,"findings":%d,"gaps":%d,"observations":%d}`,
				summary.Strengths, summary.Findings, summary.Gaps, summary.Observations,
			),
			Model:         req.Model,
			PromptVersion: req.PromptVersion,
		}

		if err := s.InsertDraftAuditLog(c.Request().Context(), d); err != nil {
			slog.Error("insert draft audit log failed", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		}
		if pub != nil {
			pub.PublishDraftAuditLog(d.DraftID, d.PolicyID, d.Summary)
		}
		return c.JSON(http.StatusCreated, map[string]string{"status": "drafted", "draft_id": d.DraftID})
	}
}

func listDraftAuditLogsHandler(s DraftAuditLogStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		status := c.QueryParam("status")
		limit := consts.ClampLimit(0)
		if v := c.QueryParam("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = consts.ClampLimit(n)
			}
		}
		drafts, err := s.ListDraftAuditLogs(c.Request().Context(), status, limit)
		if err != nil {
			slog.Error("list draft audit logs failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if drafts == nil {
			drafts = []DraftAuditLog{}
		}
		return c.JSON(http.StatusOK, drafts)
	}
}

func getDraftAuditLogHandler(s DraftAuditLogStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		draftID := c.Param("id")
		if draftID == "" {
			return c.String(http.StatusBadRequest, "missing draft id")
		}
		draft, err := s.GetDraftAuditLog(c.Request().Context(), draftID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.String(http.StatusNotFound, "draft not found")
			}
			slog.Error("get draft audit log failed", "error", err, "id", draftID)
			return c.String(http.StatusInternalServerError, "internal server error")
		}
		return c.JSON(http.StatusOK, draft)
	}
}

// updateDraftEditsHandler handles PATCH /api/draft-audit-logs/{id}.
// Persists reviewer type overrides and notes. Truncates notes to 2000 chars.
func updateDraftEditsHandler(s DraftAuditLogStore) echo.HandlerFunc {
	type editEntry struct {
		TypeOverride string `json:"type_override,omitempty"`
		Note         string `json:"note,omitempty"`
	}
	type patchReq struct {
		ReviewerEdits map[string]editEntry `json:"reviewer_edits"`
	}
	return func(c echo.Context) error {
		draftID := c.Param("id")
		if draftID == "" {
			return c.String(http.StatusBadRequest, "missing draft id")
		}

		var req patchReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
		}

		for k, v := range req.ReviewerEdits {
			if len(v.Note) > 2000 {
				v.Note = v.Note[:2000]
				req.ReviewerEdits[k] = v
			}
		}

		editsJSON, err := json.Marshal(req.ReviewerEdits)
		if err != nil {
			return c.String(http.StatusInternalServerError, "failed to serialize edits")
		}

		if err := s.UpdateDraftEdits(c.Request().Context(), draftID, string(editsJSON)); err != nil {
			if errors.Is(err, ErrDraftAlreadyPromoted) {
				return c.JSON(http.StatusConflict, map[string]string{"error": "draft already promoted"})
			}
			if errors.Is(err, ErrDraftNotFound) {
				return c.String(http.StatusNotFound, "draft not found")
			}
			slog.Error("update draft edits failed", "error", err)
			return c.String(http.StatusInternalServerError, "update failed")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "saved"})
	}
}

// promoteAuditLogHandler handles POST /api/audit-logs/promote.
// Requires an authenticated admin session. The promoting user's identity
// becomes created_by on the official AuditLog.
func promoteAuditLogHandler(s DraftAuditLogStore) echo.HandlerFunc {
	type promoteReq struct {
		DraftID string `json:"draft_id"`
	}
	return func(c echo.Context) error {
		var req promoteReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
		}
		if req.DraftID == "" {
			return c.String(http.StatusBadRequest, "draft_id required")
		}

		reviewedBy := authSessionEmail(c.Request().Context())

		if err := s.PromoteDraftAuditLog(c.Request().Context(), req.DraftID, reviewedBy); err != nil {
			if errors.Is(err, ErrDraftAlreadyPromoted) {
				return c.JSON(http.StatusConflict, map[string]string{"error": "draft already promoted"})
			}
			slog.Error("promote draft failed", "error", err)
			return c.String(http.StatusInternalServerError, "promote failed")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "promoted", "draft_id": req.DraftID})
	}
}

func authSessionEmail(ctx context.Context) string {
	if id, ok := httputil.IdentityFrom(ctx); ok {
		return id
	}
	return "unknown"
}

func listPostureHandler(s PostureStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		start, end, err := parseOptionalTimeRange(c)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		rows, err := s.ListPosture(c.Request().Context(), start, end)
		if err != nil {
			slog.Error("list posture failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []PostureRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

// parseQueryLimit extracts an optional "limit" query parameter and clamps it
// to the project-wide range [DefaultQueryLimit, MaxQueryLimit].
func parseQueryLimit(c echo.Context) int {
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return consts.ClampLimit(n)
		}
	}
	return consts.ClampLimit(0)
}

// parseOptionalTimeRange extracts optional start/end query parameters.
// Accepts date-only (2006-01-02) or RFC 3339 formats.
func parseOptionalTimeRange(c echo.Context) (start, end time.Time, err error) {
	if v := c.QueryParam("start"); v != "" {
		start, err = parseFlexibleTime(v, false)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidStart
		}
	}
	if v := c.QueryParam("end"); v != "" {
		end, err = parseFlexibleTime(v, true)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidEnd
		}
	}
	return start, end, nil
}

var (
	errInvalidStart = errors.New("invalid start parameter")
	errInvalidEnd   = errors.New("invalid end parameter")
)

// parseFlexibleTime parses RFC 3339 or date-only (YYYY-MM-DD) strings.
// Date-only values are treated as end-of-day (next day at 00:00 minus 1ns)
// when used as an upper bound so that the full calendar day is included.
func parseFlexibleTime(s string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if endOfDay {
			t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}
		return t, nil
	}
	return time.Time{}, errInvalidDateFormat
}

var errInvalidDateFormat = errors.New("expected YYYY-MM-DD or RFC 3339 format")

func listThreatsHandler(s ThreatStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		catalogID := c.QueryParam("catalog_id")
		policyID := c.QueryParam("policy_id")
		limit := parseQueryLimit(c)
		rows, err := s.QueryThreats(c.Request().Context(), catalogID, policyID, limit)
		if err != nil {
			slog.Error("query threats failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []gemarapkg.ThreatRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listControlThreatsHandler(s ThreatStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		catalogID := c.QueryParam("catalog_id")
		controlID := c.QueryParam("control_id")
		limit := parseQueryLimit(c)
		rows, err := s.QueryControlThreats(c.Request().Context(), catalogID, controlID, limit)
		if err != nil {
			slog.Error("query control threats failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []gemarapkg.ControlThreatRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listRisksHandler(s RiskStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		catalogID := c.QueryParam("catalog_id")
		policyID := c.QueryParam("policy_id")
		limit := parseQueryLimit(c)
		rows, err := s.QueryRisks(c.Request().Context(), catalogID, policyID, limit)
		if err != nil {
			slog.Error("query risks failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []gemarapkg.RiskRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listRiskThreatsHandler(s RiskStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		catalogID := c.QueryParam("catalog_id")
		riskID := c.QueryParam("risk_id")
		limit := parseQueryLimit(c)
		rows, err := s.QueryRiskThreats(c.Request().Context(), catalogID, riskID, limit)
		if err != nil {
			slog.Error("query risk threats failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []gemarapkg.RiskThreatRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func riskSeverityHandler(s RiskStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		policyID := c.QueryParam("policy_id")
		if policyID == "" {
			return c.String(http.StatusBadRequest, "policy_id required")
		}
		rows, err := s.GetPolicyRiskSeverity(c.Request().Context(), policyID)
		if err != nil {
			slog.Error("risk severity query failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []RiskSeverityRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listNotificationsHandler(s NotificationStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		limit := consts.ClampLimit(0)
		if v := c.QueryParam("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = consts.ClampLimit(n)
			}
		}
		notifs, err := s.ListNotifications(c.Request().Context(), limit)
		if err != nil {
			slog.Error("list notifications failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		if notifs == nil {
			notifs = []Notification{}
		}
		return c.JSON(http.StatusOK, notifs)
	}
}

func unreadCountHandler(s NotificationStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		count, err := s.UnreadCount(c.Request().Context())
		if err != nil {
			slog.Error("unread count failed", "error", err)
			return c.String(http.StatusInternalServerError, "query failed")
		}
		return c.JSON(http.StatusOK, map[string]int{"count": count})
	}
}

func markReadHandler(s NotificationStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return c.String(http.StatusBadRequest, "missing notification id")
		}
		if err := s.MarkRead(c.Request().Context(), id); err != nil {
			if errors.Is(err, ErrNotFound) {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "notification not found"})
			}
			slog.Error("mark read failed", "error", err, "id", id)
			return c.String(http.StatusInternalServerError, "update failed")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "read"})
	}
}

func createNotificationHandler(s NotificationStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var n Notification
		if err := c.Bind(&n); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
		}
		if n.NotificationID == "" || n.Type == "" || n.PolicyID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "notification_id, type, and policy_id are required",
			})
		}
		if err := s.InsertNotification(c.Request().Context(), n); err != nil {
			slog.Error("create notification failed", "error", err)
			return c.String(http.StatusInternalServerError, "insert failed")
		}
		return c.JSON(http.StatusCreated, map[string]string{"status": "created"})
	}
}

