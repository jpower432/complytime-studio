// SPDX-License-Identifier: Apache-2.0

package store

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/httputil"
)

// Stores groups all domain store interfaces for handler registration.
type Stores struct {
	Policies  PolicyStore
	Mappings  MappingStore
	Evidence  EvidenceStore
	AuditLogs AuditLogStore
}

// Register mounts all store API endpoints on the mux.
func Register(mux *http.ServeMux, s Stores) {
	mux.HandleFunc("GET /api/policies", listPoliciesHandler(s.Policies))
	mux.HandleFunc("GET /api/policies/{id}", getPolicyHandler(s.Policies, s.Mappings))
	mux.HandleFunc("POST /api/policies/import", importPolicyHandler(s.Policies))
	mux.HandleFunc("POST /api/mappings/import", importMappingHandler(s.Mappings))
	mux.HandleFunc("GET /api/evidence", queryEvidenceHandler(s.Evidence))
	mux.HandleFunc("POST /api/evidence", ingestEvidenceHandler(s.Evidence))
	mux.HandleFunc("POST /api/evidence/upload", uploadEvidenceHandler(s.Evidence))
	mux.HandleFunc("GET /api/audit-logs/{id}", getAuditLogHandler(s.AuditLogs))
	mux.HandleFunc("GET /api/audit-logs", listAuditLogsHandler(s.AuditLogs))
	mux.HandleFunc("POST /api/audit-logs", createAuditLogHandler(s.AuditLogs))
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
			PolicyID:  req.PolicyID,
			Framework: req.Framework,
			Content:   req.Content,
		}
		if err := s.InsertMapping(r.Context(), m); err != nil {
			slog.Error("insert mapping failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, map[string]string{"status": "imported", "mapping_id": m.MappingID})
	}
}

func ingestEvidenceHandler(s EvidenceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var records []EvidenceRecord
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&records); err != nil {
			http.Error(w, "invalid json array", http.StatusBadRequest)
			return
		}
		var errors []string
		for i, rec := range records {
			if rec.PolicyID == "" || rec.TargetID == "" || rec.ControlID == "" || rec.CollectedAt.IsZero() {
				errors = append(errors, fmt.Sprintf("row %d: missing required fields (policy_id, target_id, control_id, collected_at)", i))
			}
		}
		if len(errors) > 0 {
			httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{"errors": errors})
			return
		}
		count, err := s.InsertEvidence(r.Context(), records)
		if err != nil {
			slog.Error("insert evidence failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, map[string]int{"inserted": count})
	}
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
		defer file.Close()

		var records []EvidenceRecord
		var parseErrors []string

		if strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
			records, parseErrors = parseCSVEvidence(file)
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

		httputil.WriteJSON(w, http.StatusOK, map[string]any{
			"inserted": inserted,
			"failed":   len(parseErrors),
			"errors":   parseErrors,
		})
	}
}

func parseCSVEvidence(r io.Reader) ([]EvidenceRecord, []string) {
	reader := csv.NewReader(r)
	headers, err := reader.Read()
	if err != nil {
		return nil, []string{"failed to read CSV header"}
	}

	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	required := []string{"policy_id", "target_id", "control_id", "collected_at"}
	for _, req := range required {
		if _, ok := colIdx[req]; !ok {
			return nil, []string{fmt.Sprintf("missing required column: %s", req)}
		}
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

		t, tErr := time.Parse(time.RFC3339, strings.TrimSpace(row[colIdx["collected_at"]]))
		if tErr != nil {
			errors = append(errors, fmt.Sprintf("line %d: invalid collected_at timestamp", lineNum))
			continue
		}

		rec := EvidenceRecord{
			PolicyID:    strings.TrimSpace(row[colIdx["policy_id"]]),
			TargetID:    strings.TrimSpace(row[colIdx["target_id"]]),
			ControlID:   strings.TrimSpace(row[colIdx["control_id"]]),
			CollectedAt: t,
		}
		if idx, ok := colIdx["evidence_id"]; ok && idx < len(row) {
			rec.EvidenceID = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIdx["rule_id"]; ok && idx < len(row) {
			rec.RuleID = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIdx["eval_result"]; ok && idx < len(row) {
			rec.EvalResult = strings.TrimSpace(row[idx])
		}
		records = append(records, rec)
	}
	return records, errors
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
			Framework:     q.Get("framework"),
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
		if f.Limit == 0 {
			f.Limit = 100
		}

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
		logs, err := s.ListAuditLogs(r.Context(), policyID, start, end)
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
	return func(w http.ResponseWriter, r *http.Request) {
		var a AuditLog
		if err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxRequestBody)).Decode(&a); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if a.PolicyID == "" || a.Content == "" || a.AuditStart.IsZero() || a.AuditEnd.IsZero() {
			http.Error(w, "policy_id, content, audit_start, and audit_end required", http.StatusBadRequest)
			return
		}
		if err := s.InsertAuditLog(r.Context(), a); err != nil {
			slog.Error("insert audit log failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, map[string]string{"status": "stored", "audit_id": a.AuditID})
	}
}
