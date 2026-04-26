// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/complytime/complytime-studio/internal/blob"
	"github.com/complytime/complytime-studio/internal/consts"
	gemarapkg "github.com/complytime/complytime-studio/internal/gemara"
	"github.com/complytime/complytime-studio/internal/httputil"
)

// EvidencePublisher emits events after evidence is ingested.
// Implemented by *events.Bus; nil-safe (callers check before use).
type EvidencePublisher interface {
	PublishEvidence(policyID string, count int)
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
	EventPublisher      EvidencePublisher
}

// Register mounts all public store API endpoints on the mux.
// Internal (agent-only) endpoints are registered via RegisterInternal.
func Register(mux *http.ServeMux, s Stores) {
	mux.HandleFunc("GET /api/policies", listPoliciesHandler(s.Policies))
	mux.HandleFunc("GET /api/policies/{id}", getPolicyHandler(s.Policies, s.Mappings))
	mux.HandleFunc("POST /api/policies/import", importPolicyHandler(s.Policies))
	mux.HandleFunc("POST /api/mappings/import", importMappingHandler(s.Mappings))
	mux.HandleFunc("GET /api/evidence", queryEvidenceHandler(s.Evidence))
	mux.HandleFunc("POST /api/evidence", ingestEvidenceHandler(s.Evidence, s.Blob, s.EventPublisher))
	mux.HandleFunc("POST /api/evidence/upload", uploadEvidenceHandler(s.Evidence))
	mux.HandleFunc("GET /api/audit-logs/{id}", getAuditLogHandler(s.AuditLogs))
	mux.HandleFunc("GET /api/audit-logs", listAuditLogsHandler(s.AuditLogs))
	mux.HandleFunc("POST /api/audit-logs", createAuditLogHandler(s.AuditLogs))
	mux.HandleFunc("POST /api/catalogs/import", importCatalogHandler(s.Catalogs, s.Controls, s.Threats, s.Risks))
	if s.Posture != nil {
		mux.HandleFunc("GET /api/posture", listPostureHandler(s.Posture))
	}
	if s.Requirements != nil {
		mux.HandleFunc("GET /api/requirements", listRequirementMatrixHandler(s.Requirements))
		mux.HandleFunc("GET /api/requirements/{id}/evidence", listRequirementEvidenceHandler(s.Requirements))
		mux.HandleFunc("GET /api/export/csv", exportCSVHandler(s.Requirements, s.Policies))
		mux.HandleFunc("GET /api/export/excel", exportExcelHandler(s.Requirements, s.Evidence, s.Policies, s.AuditLogs))
		mux.HandleFunc("GET /api/export/pdf", exportPDFHandler(s.Requirements, s.Policies, s.AuditLogs))
	}
	if s.DraftAuditLogs != nil {
		mux.HandleFunc("GET /api/draft-audit-logs", listDraftAuditLogsHandler(s.DraftAuditLogs))
		mux.HandleFunc("GET /api/draft-audit-logs/{id}", getDraftAuditLogHandler(s.DraftAuditLogs))
		mux.HandleFunc("PATCH /api/draft-audit-logs/{id}", updateDraftEditsHandler(s.DraftAuditLogs))
		mux.HandleFunc("POST /api/audit-logs/promote", promoteAuditLogHandler(s.DraftAuditLogs))
	}
	if s.Risks != nil {
		mux.HandleFunc("GET /api/risks/severity", riskSeverityHandler(s.Risks))
	}
	if s.Notifications != nil {
		mux.HandleFunc("GET /api/notifications", listNotificationsHandler(s.Notifications))
		mux.HandleFunc("GET /api/notifications/unread-count", unreadCountHandler(s.Notifications))
		mux.HandleFunc("PATCH /api/notifications/{id}/read", markReadHandler(s.Notifications))
	}
}

// RegisterInternal mounts agent-only endpoints on a separate mux served on
// the internal port. These routes have no auth middleware — access is
// restricted at the network layer via NetworkPolicy.
// See docs/decisions/internal-endpoint-isolation.md.
func RegisterInternal(mux *http.ServeMux, s Stores) {
	if s.DraftAuditLogs != nil {
		mux.HandleFunc("POST /internal/draft-audit-logs", createDraftAuditLogHandler(s.DraftAuditLogs))
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func listPoliciesHandler(s PolicyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policies, err := s.ListPolicies(r.Context())
		if err != nil {
			slog.Error("list policies failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if policies == nil {
			policies = []Policy{}
		}
		httputil.WriteJSON(w, http.StatusOK, policies)
	}
}

func getPolicyHandler(ps PolicyStore, ms MappingStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing policy id", http.StatusBadRequest)
			return
		}
		p, err := ps.GetPolicy(r.Context(), id)
		if err != nil {
			slog.Error("get policy failed", "error", err, "id", id)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		mappings, _ := ms.ListMappings(r.Context(), id)
		if mappings == nil {
			mappings = []MappingDocument{}
		}
		resp := struct {
			Policy   *Policy           `json:"policy"`
			Mappings []MappingDocument `json:"mappings"`
		}{Policy: p, Mappings: mappings}
		httputil.WriteJSON(w, http.StatusOK, resp)
	}
}

func importPolicyHandler(s PolicyStore) http.HandlerFunc {
	type importReq struct {
		OCIReference string `json:"oci_reference"`
		Content      string `json:"content"`
		Title        string `json:"title"`
		Version      string `json:"version"`
		PolicyID     string `json:"policy_id"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req importReq
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Content == "" || req.Title == "" {
			http.Error(w, "content and title required", http.StatusBadRequest)
			return
		}
		p := Policy{
			PolicyID:     req.PolicyID,
			Title:        req.Title,
			Version:      req.Version,
			OCIReference: req.OCIReference,
			Content:      req.Content,
		}
		if err := s.InsertPolicy(r.Context(), p); err != nil {
			slog.Error("insert policy failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}

		httputil.WriteJSON(w, http.StatusCreated, map[string]string{"status": "imported", "policy_id": p.PolicyID})
	}
}

func importMappingHandler(s MappingStore) http.HandlerFunc {
	type importReq struct {
		MappingID string `json:"mapping_id"`
		PolicyID  string `json:"policy_id"`
		Framework string `json:"framework"`
		Content   string `json:"content"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req importReq
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.PolicyID == "" || req.Framework == "" || req.Content == "" {
			http.Error(w, "policy_id, framework, and content required", http.StatusBadRequest)
			return
		}
		m := MappingDocument{
			MappingID: req.MappingID,
			PolicyID:  req.PolicyID,
			Framework: req.Framework,
			Content:   req.Content,
		}
		if err := s.InsertMapping(r.Context(), m); err != nil {
			slog.Error("insert mapping failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}

		entries, parseErr := gemarapkg.ParseMappingEntries(req.Content, m.MappingID, req.PolicyID, req.Framework)
		if parseErr != nil {
			slog.Warn("mapping YAML parse failed, structured entries skipped",
				"mapping_id", m.MappingID, "error", parseErr)
		} else if len(entries) > 0 {
			if err := s.InsertMappingEntries(r.Context(), entries); err != nil {
				slog.Warn("insert mapping entries failed",
					"mapping_id", m.MappingID, "error", err)
			} else {
				slog.Info("mapping entries stored",
					"mapping_id", m.MappingID, "count", len(entries))
			}
		}

		httputil.WriteJSON(w, http.StatusCreated, map[string]string{"status": "imported", "mapping_id": m.MappingID})
	}
}

func ingestEvidenceHandler(s EvidenceStore, blobs blob.BlobStore, pub EvidencePublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var records []EvidenceRecord
		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/form-data") {
			if err := r.ParseMultipartForm(consts.MaxRequestBody); err != nil {
				http.Error(w, "invalid multipart form", http.StatusBadRequest)
				return
			}
			dataStr := r.FormValue("data")
			if dataStr == "" {
				httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
					"errors": []string{`multipart request requires form field "data" with a JSON array of evidence records`},
				})
				return
			}
			if err := json.NewDecoder(strings.NewReader(dataStr)).Decode(&records); err != nil {
				httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{"errors": []string{"invalid JSON in data field"}})
				return
			}
			f, hdr, hasFile, ferr := formFileOptional(r, "file", "attachment")
			if ferr != nil {
				http.Error(w, ferr.Error(), http.StatusBadRequest)
				return
			}
			if hasFile {
				defer func() { _ = f.Close() }()
				if blobs == nil {
					httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
						"errors": []string{consts.MsgBlobStorageNotConfigured},
					})
					return
				}
				key := blob.EvidenceObjectKey(hdr.Filename)
				size := hdr.Size
				if size <= 0 {
					size = -1
				}
				ref, err := blobs.Upload(r.Context(), key, f, size)
				if err != nil {
					slog.Error("blob upload failed", "error", err)
					http.Error(w, "blob upload failed", http.StatusBadGateway)
					return
				}
				for i := range records {
					records[i].BlobRef = ref
				}
			}
		} else {
			if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&records); err != nil {
				http.Error(w, "invalid json array", http.StatusBadRequest)
				return
			}
		}

		var valErrs []string
		for i, rec := range records {
			if rec.PolicyID == "" || rec.TargetID == "" || rec.ControlID == "" || rec.CollectedAt.IsZero() {
				valErrs = append(valErrs, fmt.Sprintf("row %d: missing required fields (policy_id, target_id, control_id, collected_at)", i))
			}
			valErrs = append(valErrs, validateEvidenceRecordEnums(rec, i)...)
		}
		if len(valErrs) > 0 {
			httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{"errors": valErrs})
			return
		}
		count, err := s.InsertEvidence(r.Context(), records)
		if err != nil {
			slog.Error("insert evidence failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}
		if pub != nil && count > 0 {
			policyIDs := make(map[string]int)
			for _, rec := range records {
				if rec.PolicyID != "" {
					policyIDs[rec.PolicyID]++
				}
			}
			for pid, n := range policyIDs {
				pub.PublishEvidence(pid, n)
			}
		}
		httputil.WriteJSON(w, http.StatusCreated, map[string]int{"inserted": count})
	}
}

func formFileOptional(r *http.Request, names ...string) (multipart.File, *multipart.FileHeader, bool, error) {
	for _, n := range names {
		f, h, err := r.FormFile(n)
		if err == nil {
			return f, h, true, nil
		}
		if !errors.Is(err, http.ErrMissingFile) {
			return nil, nil, false, err
		}
	}
	return nil, nil, false, nil
}

func uploadEvidenceHandler(s EvidenceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(consts.MaxRequestBody); err != nil {
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file field required", http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()

		var records []EvidenceRecord
		var parseErrors []string
		var parseWarnings []string

		if strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
			records, parseErrors, parseWarnings = parseCSVEvidence(file)
		} else {
			if err := json.NewDecoder(io.LimitReader(file, consts.MaxRequestBody)).Decode(&records); err != nil {
				http.Error(w, "invalid json file", http.StatusBadRequest)
				return
			}
		}

		if len(records) == 0 && len(parseErrors) == 0 {
			http.Error(w, "no records found", http.StatusBadRequest)
			return
		}

		var valid []EvidenceRecord
		for i, rec := range records {
			if rec.PolicyID == "" || rec.TargetID == "" || rec.ControlID == "" || rec.CollectedAt.IsZero() {
				parseErrors = append(parseErrors, fmt.Sprintf("row %d: missing required fields", i))
				continue
			}
			valid = append(valid, rec)
		}

		inserted := 0
		if len(valid) > 0 {
			inserted, err = s.InsertEvidence(r.Context(), valid)
			if err != nil {
				slog.Error("insert evidence from upload failed", "error", err)
				http.Error(w, "insert failed", http.StatusInternalServerError)
				return
			}
		}

		resp := map[string]any{
			"inserted": inserted,
			"failed":   len(parseErrors),
			"errors":   parseErrors,
		}
		if len(parseWarnings) > 0 {
			resp["warnings"] = parseWarnings
		}
		httputil.WriteJSON(w, http.StatusOK, resp)
	}
}

func parseCSVEvidence(r io.Reader) ([]EvidenceRecord, []string, []string) {
	reader := csv.NewReader(r)
	headers, err := reader.Read()
	if err != nil {
		return nil, []string{"failed to read CSV header"}, nil
	}

	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	required := []string{"policy_id", "eval_result", "collected_at"}
	for _, req := range required {
		if _, ok := colIdx[req]; !ok {
			return nil, []string{fmt.Sprintf("missing required column: %s", req)}, nil
		}
	}

	var warnings []string
	recommended := []string{"requirement_id"}
	for _, col := range recommended {
		if _, ok := colIdx[col]; !ok {
			warnings = append(warnings, fmt.Sprintf("recommended column '%s' not in header", col))
		}
	}

	csvStr := func(row []string, col string) string {
		idx, ok := colIdx[col]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	var records []EvidenceRecord
	var errors []string
	lineNum := 1
	for {
		row, err := reader.Read()
		if err != nil {
			break
		}
		lineNum++

		t, tErr := time.Parse(time.RFC3339, csvStr(row, "collected_at"))
		if tErr != nil {
			errors = append(errors, fmt.Sprintf("line %d: invalid collected_at timestamp", lineNum))
			continue
		}

		rec := EvidenceRecord{
			EvidenceID:       csvStr(row, "evidence_id"),
			PolicyID:         csvStr(row, "policy_id"),
			TargetID:         csvStr(row, "target_id"),
			TargetName:       csvStr(row, "target_name"),
			TargetType:       csvStr(row, "target_type"),
			TargetEnv:        csvStr(row, "target_env"),
			EngineName:       csvStr(row, "engine_name"),
			EngineVersion:    csvStr(row, "engine_version"),
			RuleID:           csvStr(row, "rule_id"),
			RuleName:         csvStr(row, "rule_name"),
			EvalResult:       csvStr(row, "eval_result"),
			EvalMessage:      csvStr(row, "eval_message"),
			ControlID:        csvStr(row, "control_id"),
			ControlCatalogID: csvStr(row, "control_catalog_id"),
			ControlCategory:  csvStr(row, "control_category"),
			RequirementID:    csvStr(row, "requirement_id"),
			PlanID:           csvStr(row, "plan_id"),
			Confidence:       csvStr(row, "confidence"),
			ComplianceStatus: csvStr(row, "compliance_status"),
			RiskLevel:        csvStr(row, "risk_level"),
			EnrichmentStatus: csvStr(row, "enrichment_status"),
			AttestationRef:   csvStr(row, "attestation_ref"),
			SourceRegistry:   csvStr(row, "source_registry"),
			BlobRef:          csvStr(row, "blob_ref"),
			CollectedAt:      t,
		}
		records = append(records, rec)
	}
	return records, errors, warnings
}

func queryEvidenceHandler(s EvidenceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		f := EvidenceFilter{
			PolicyID:      q.Get("policy_id"),
			ControlID:     q.Get("control_id"),
			TargetName:    q.Get("target_name"),
			TargetType:    q.Get("target_type"),
			TargetEnv:     q.Get("target_env"),
			EngineVersion: q.Get("engine_version"),
			Owner:         q.Get("owner"),
		}
		if v := q.Get("start"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				f.Start = t
			} else if t, err := time.Parse("2006-01-02", v); err == nil {
				f.Start = t
			}
		}
		if v := q.Get("end"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				f.End = t
			} else if t, err := time.Parse("2006-01-02", v); err == nil {
				f.End = t
			}
		}
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				f.Limit = n
			}
		}
		if v := q.Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				f.Offset = n
			}
		}
		f.Limit = consts.ClampLimit(f.Limit)

		records, err := s.QueryEvidence(r.Context(), f)
		if err != nil {
			slog.Error("query evidence failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		if records == nil {
			records = []EvidenceRecord{}
		}
		httputil.WriteJSON(w, http.StatusOK, records)
	}
}

func getAuditLogHandler(s AuditLogStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing audit id", http.StatusBadRequest)
			return
		}
		a, err := s.GetAuditLog(r.Context(), id)
		if err != nil {
			slog.Error("get audit log failed", "error", err, "id", id)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, a)
	}
}

func listAuditLogsHandler(s AuditLogStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		policyID := q.Get("policy_id")
		if policyID == "" {
			http.Error(w, "policy_id required", http.StatusBadRequest)
			return
		}
		var start, end time.Time
		if v := q.Get("start"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				start = t
			}
		}
		if v := q.Get("end"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				end = t
			}
		}
		limit := consts.ClampLimit(0)
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = consts.ClampLimit(n)
			}
		}

		logs, err := s.ListAuditLogs(r.Context(), policyID, start, end, limit)
		if err != nil {
			slog.Error("list audit logs failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		if logs == nil {
			logs = []AuditLog{}
		}
		httputil.WriteJSON(w, http.StatusOK, logs)
	}
}

func createAuditLogHandler(s AuditLogStore) http.HandlerFunc {
	type createReq struct {
		PolicyID      string `json:"policy_id"`
		Content       string `json:"content"`
		Model         string `json:"model,omitempty"`
		PromptVersion string `json:"prompt_version,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req createReq
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.PolicyID == "" || req.Content == "" {
			http.Error(w, "policy_id and content required", http.StatusBadRequest)
			return
		}

		summary, parseErr := gemarapkg.ParseAuditLog(req.Content)
		if parseErr != nil {
			slog.Warn("audit log YAML parse failed", "policy_id", req.PolicyID, "error", parseErr)
			http.Error(w, fmt.Sprintf("invalid audit log content: %v", parseErr), http.StatusBadRequest)
			return
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

		if err := s.InsertAuditLog(r.Context(), a); err != nil {
			slog.Error("insert audit log failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, map[string]string{"status": "stored", "audit_id": a.AuditID})
	}
}

func importCatalogHandler(cs CatalogStore, ctrlS ControlStore, threatS ThreatStore, riskS RiskStore) http.HandlerFunc {
	type importReq struct {
		CatalogID string `json:"catalog_id"`
		PolicyID  string `json:"policy_id"`
		Content   string `json:"content"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req importReq
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Content == "" {
			http.Error(w, "content required", http.StatusBadRequest)
			return
		}

		catalogType, title := detectCatalogType(req.Content)
		if catalogType == "" {
			http.Error(w, "could not detect catalog type from content (expected ControlCatalog, ThreatCatalog, or RiskCatalog)", http.StatusBadRequest)
			return
		}

		catalogID := req.CatalogID
		if catalogID == "" {
			catalogID = detectCatalogID(req.Content)
		}

		if cs != nil {
			if err := cs.InsertCatalog(r.Context(), Catalog{
				CatalogID:   catalogID,
				CatalogType: catalogType,
				Title:       title,
				Content:     req.Content,
				PolicyID:    req.PolicyID,
			}); err != nil {
				slog.Error("insert catalog failed", "error", err)
				http.Error(w, "insert failed", http.StatusInternalServerError)
				return
			}
		}

		parseCatalogStructuredRows(r.Context(), catalogType, req.Content, catalogID, req.PolicyID, ctrlS, threatS, riskS)

		httputil.WriteJSON(w, http.StatusCreated, map[string]string{
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
		controls, reqs, threats, err := gemarapkg.ParseControlCatalog(content, catalogID, policyID)
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
		rows, err := gemarapkg.ParseThreatCatalog(content, catalogID, policyID)
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
		riskRows, linkRows, err := gemarapkg.ParseRiskCatalog(content, catalogID, policyID)
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
	case "ControlCatalog", "ThreatCatalog", "RiskCatalog":
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

func listRequirementMatrixHandler(s RequirementStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policyID := r.URL.Query().Get("policy_id")
		if policyID == "" {
			http.Error(w, "policy_id required", http.StatusBadRequest)
			return
		}

		f := RequirementFilter{PolicyID: policyID}

		if v := r.URL.Query().Get("audit_start"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err != nil {
				http.Error(w, "invalid audit_start format", http.StatusBadRequest)
				return
			}
			f.Start = t
		}
		if v := r.URL.Query().Get("audit_end"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err != nil {
				http.Error(w, "invalid audit_end format", http.StatusBadRequest)
				return
			}
			f.End = t
		}
		if !f.Start.IsZero() && !f.End.IsZero() && f.End.Before(f.Start) {
			http.Error(w, "audit_end must be >= audit_start", http.StatusBadRequest)
			return
		}

		f.Classification = r.URL.Query().Get("classification")
		f.ControlFamily = r.URL.Query().Get("control_family")

		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Offset = n
			}
		}
		f.Limit = consts.ClampLimit(f.Limit)

		rows, err := s.ListRequirementMatrix(r.Context(), f)
		if err != nil {
			slog.Error("list requirement matrix failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		if rows == nil {
			rows = []RequirementRow{}
		}
		httputil.WriteJSON(w, http.StatusOK, rows)
	}
}

func listRequirementEvidenceHandler(s RequirementStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := r.PathValue("id")
		if reqID == "" {
			http.Error(w, "missing requirement id", http.StatusBadRequest)
			return
		}
		policyID := r.URL.Query().Get("policy_id")
		if policyID == "" {
			http.Error(w, "policy_id required", http.StatusBadRequest)
			return
		}

		f := RequirementFilter{PolicyID: policyID}
		if v := r.URL.Query().Get("audit_start"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err == nil {
				f.Start = t
			}
		}
		if v := r.URL.Query().Get("audit_end"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err == nil {
				f.End = t
			}
		}
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Offset = n
			}
		}
		f.Limit = consts.ClampLimit(f.Limit)

		rows, err := s.ListRequirementEvidence(r.Context(), reqID, f)
		if err != nil {
			if errors.Is(err, ErrRequirementNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			slog.Error("list requirement evidence failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		if rows == nil {
			rows = []RequirementEvidenceRow{}
		}
		httputil.WriteJSON(w, http.StatusOK, rows)
	}
}

// createDraftAuditLogHandler handles POST /internal/draft-audit-logs.
// No auth required — cluster-internal only, restricted by NetworkPolicy.
func createDraftAuditLogHandler(s DraftAuditLogStore) http.HandlerFunc {
	type createReq struct {
		PolicyID       string `json:"policy_id"`
		Content        string `json:"content"`
		AgentReasoning string `json:"agent_reasoning,omitempty"`
		Model          string `json:"model,omitempty"`
		PromptVersion  string `json:"prompt_version,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req createReq
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.PolicyID == "" || req.Content == "" {
			http.Error(w, "policy_id and content required", http.StatusBadRequest)
			return
		}

		summary, parseErr := gemarapkg.ParseAuditLog(req.Content)
		if parseErr != nil {
			slog.Warn("draft audit log YAML parse failed", "policy_id", req.PolicyID, "error", parseErr)
			http.Error(w, fmt.Sprintf("invalid audit log content: %v", parseErr), http.StatusBadRequest)
			return
		}

		d := DraftAuditLog{
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

		if err := s.InsertDraftAuditLog(r.Context(), d); err != nil {
			slog.Error("insert draft audit log failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, map[string]string{"status": "drafted", "draft_id": d.DraftID})
	}
}

func listDraftAuditLogsHandler(s DraftAuditLogStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		status := q.Get("status")
		limit := consts.ClampLimit(0)
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = consts.ClampLimit(n)
			}
		}
		drafts, err := s.ListDraftAuditLogs(r.Context(), status, limit)
		if err != nil {
			slog.Error("list draft audit logs failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		if drafts == nil {
			drafts = []DraftAuditLog{}
		}
		httputil.WriteJSON(w, http.StatusOK, drafts)
	}
}

func getDraftAuditLogHandler(s DraftAuditLogStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		draftID := r.PathValue("id")
		if draftID == "" {
			http.Error(w, "missing draft id", http.StatusBadRequest)
			return
		}
		draft, err := s.GetDraftAuditLog(r.Context(), draftID)
		if err != nil {
			http.Error(w, "draft not found", http.StatusNotFound)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, draft)
	}
}

// updateDraftEditsHandler handles PATCH /api/draft-audit-logs/{id}.
// Persists reviewer type overrides and notes. Truncates notes to 2000 chars.
func updateDraftEditsHandler(s DraftAuditLogStore) http.HandlerFunc {
	type editEntry struct {
		TypeOverride string `json:"type_override,omitempty"`
		Note         string `json:"note,omitempty"`
	}
	type patchReq struct {
		ReviewerEdits map[string]editEntry `json:"reviewer_edits"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		draftID := r.PathValue("id")
		if draftID == "" {
			http.Error(w, "missing draft id", http.StatusBadRequest)
			return
		}

		var req patchReq
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		for k, v := range req.ReviewerEdits {
			if len(v.Note) > 2000 {
				v.Note = v.Note[:2000]
				req.ReviewerEdits[k] = v
			}
		}

		editsJSON, err := json.Marshal(req.ReviewerEdits)
		if err != nil {
			http.Error(w, "failed to serialize edits", http.StatusInternalServerError)
			return
		}

		if err := s.UpdateDraftEdits(r.Context(), draftID, string(editsJSON)); err != nil {
			if errors.Is(err, ErrDraftAlreadyPromoted) {
				httputil.WriteJSON(w, http.StatusConflict, map[string]string{"error": "draft already promoted"})
				return
			}
			if errors.Is(err, ErrDraftNotFound) {
				http.Error(w, "draft not found", http.StatusNotFound)
				return
			}
			slog.Error("update draft edits failed", "error", err)
			http.Error(w, "update failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	}
}

// promoteAuditLogHandler handles POST /api/audit-logs/promote.
// Requires an authenticated admin session. The promoting user's identity
// becomes created_by on the official AuditLog.
func promoteAuditLogHandler(s DraftAuditLogStore) http.HandlerFunc {
	type promoteReq struct {
		DraftID string `json:"draft_id"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req promoteReq
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.DraftID == "" {
			http.Error(w, "draft_id required", http.StatusBadRequest)
			return
		}

		reviewedBy := authSessionEmail(r.Context())

		if err := s.PromoteDraftAuditLog(r.Context(), req.DraftID, reviewedBy); err != nil {
			if errors.Is(err, ErrDraftAlreadyPromoted) {
				httputil.WriteJSON(w, http.StatusConflict, map[string]string{"error": "draft already promoted"})
				return
			}
			slog.Error("promote draft failed", "error", err)
			http.Error(w, "promote failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "promoted", "draft_id": req.DraftID})
	}
}

func authSessionEmail(ctx context.Context) string {
	if id, ok := httputil.IdentityFrom(ctx); ok {
		return id
	}
	return "unknown"
}

func listPostureHandler(s PostureStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.ListPosture(r.Context())
		if err != nil {
			slog.Error("list posture failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		if rows == nil {
			rows = []PostureRow{}
		}
		httputil.WriteJSON(w, http.StatusOK, rows)
	}
}

func riskSeverityHandler(s RiskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policyID := r.URL.Query().Get("policy_id")
		if policyID == "" {
			http.Error(w, "policy_id required", http.StatusBadRequest)
			return
		}
		rows, err := s.GetPolicyRiskSeverity(r.Context(), policyID)
		if err != nil {
			slog.Error("risk severity query failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		if rows == nil {
			rows = []RiskSeverityRow{}
		}
		httputil.WriteJSON(w, http.StatusOK, rows)
	}
}

func listNotificationsHandler(s NotificationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := consts.ClampLimit(0)
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = consts.ClampLimit(n)
			}
		}
		notifs, err := s.ListNotifications(r.Context(), limit)
		if err != nil {
			slog.Error("list notifications failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		if notifs == nil {
			notifs = []Notification{}
		}
		httputil.WriteJSON(w, http.StatusOK, notifs)
	}
}

func unreadCountHandler(s NotificationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := s.UnreadCount(r.Context())
		if err != nil {
			slog.Error("unread count failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
	}
}

func markReadHandler(s NotificationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing notification id", http.StatusBadRequest)
			return
		}
		if err := s.MarkRead(r.Context(), id); err != nil {
			slog.Error("mark read failed", "error", err, "id", id)
			http.Error(w, "update failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "read"})
	}
}

func exportCSVHandler(rs RequirementStore, ps PolicyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		policyID, _, f, err := ParseExportQuery(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), consts.ExportHandlerTimeout)
		defer cancel()

		rows, err := LoadExportMatrix(ctx, rs, f)
		if err != nil {
			if errors.Is(err, ErrExportRowLimit) {
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
				return
			}
			slog.Error("export csv: matrix query failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}

		policyTitle, _ := policyDisplayMeta(ctx, ps, policyID)

		filename := SanitizeExportFilename(policyID, ".csv")
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("Cache-Control", "no-store")

		cw := csv.NewWriter(w)
		_ = cw.Write([]string{
			"catalog_id", "control_id", "control_title",
			"requirement_id", "requirement_text",
			"evidence_count", "latest_evidence", "classification",
		})
		for _, row := range rows {
			_ = cw.Write([]string{
				row.CatalogID, row.ControlID, row.ControlTitle,
				row.RequirementID, row.RequirementText,
				strconv.FormatUint(row.EvidenceCount, 10), row.LatestEvidence, row.Classification,
			})
		}
		cw.Flush()

		slog.Info("csv export complete", "policy_id", policyID, "policy_title", policyTitle,
			"rows", len(rows), "duration_ms", time.Since(started).Milliseconds())
	}
}
