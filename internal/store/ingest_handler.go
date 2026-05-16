// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/httputil"
	"github.com/complytime/complytime-studio/internal/ingest"
)

// toEvidenceRecords converts ingest EvidenceRows to store EvidenceRecords.
func toEvidenceRecords(rows []ingest.EvidenceRow) []EvidenceRecord {
	records := make([]EvidenceRecord, len(rows))
	for i, row := range rows {
		records[i] = EvidenceRecord{
			EvidenceID:           row.EvidenceID,
			PolicyID:             derefStr(row.PolicyID),
			TargetID:             row.TargetID,
			TargetName:           derefStr(row.TargetName),
			TargetType:           derefStr(row.TargetType),
			TargetEnv:            derefStr(row.TargetEnv),
			EngineName:           derefStr(row.EngineName),
			EngineVersion:        derefStr(row.EngineVersion),
			RuleID:               row.RuleID,
			RuleName:             derefStr(row.RuleName),
			RuleURI:              derefStr(row.RuleURI),
			EvalResult:           row.EvalResult,
			EvalMessage:          derefStr(row.EvalMessage),
			ControlID:            derefStr(row.ControlID),
			ControlCatalogID:     derefStr(row.ControlCatalogID),
			ControlCategory:      derefStr(row.ControlCategory),
			ControlApplicability: row.ControlApplicability,
			RequirementID:        derefStr(row.RequirementID),
			PlanID:               derefStr(row.PlanID),
			Confidence:           derefStr(row.Confidence),
			StepsExecuted:        derefUint16(row.StepsExecuted),
			ComplianceStatus:     row.ComplianceStatus,
			RiskLevel:            derefStr(row.RiskLevel),
			Frameworks:           row.Frameworks,
			Requirements:         row.Requirements,
			RemediationAction:    derefStr(row.RemediationAction),
			RemediationStatus:    derefStr(row.RemediationStatus),
			RemediationDesc:      derefStr(row.RemediationDesc),
			ExceptionID:          derefStr(row.ExceptionID),
			ExceptionActive:      row.ExceptionActive,
			EnrichmentStatus:     row.EnrichmentStatus,
			AttestationRef:       derefStr(row.AttestationRef),
			SourceRegistry:       derefStr(row.SourceRegistry),
			BlobRef:              derefStr(row.BlobRef),
			Certified:            row.Certified,
			CollectedAt:          row.CollectedAt,
		}
	}
	return records
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefUint16(p *uint16) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

// IngestRawPublisher publishes raw YAML for async processing via NATS.
type IngestRawPublisher interface {
	PublishIngestRaw(jobID string, yaml []byte) error
}

// IngestAsyncHandler returns an http.HandlerFunc that accepts raw Gemara
// YAML, assigns a job ID, publishes it to NATS for async processing, and
// returns 202 Accepted with the job ID for polling.
func IngestAsyncHandler(pub IngestRawPublisher, tracker *IngestTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, consts.MaxRequestBody))
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		if len(body) == 0 {
			httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
				"errors": []string{"request body is empty — expected Gemara YAML"},
			})
			return
		}

		jobID := generateJobID()
		tracker.Create(jobID)

		if err := pub.PublishIngestRaw(jobID, body); err != nil {
			tracker.Fail(jobID, fmt.Sprintf("publish failed: %v", err))
			slog.Error("async ingest publish failed", "job_id", jobID, "error", err)
			httputil.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
				"errors": []string{"event bus unavailable — try again later"},
			})
			return
		}

		httputil.WriteJSON(w, http.StatusAccepted, map[string]any{
			"job_id": jobID,
			"status": "pending",
		})
	}
}

// IngestJobStatusHandler returns an echo handler for polling async ingest jobs.
func IngestJobStatusHandler(tracker *IngestTracker) echo.HandlerFunc {
	return func(c echo.Context) error {
		jobID := c.Param("job_id")
		if jobID == "" {
			return jsonError(c, http.StatusBadRequest, "missing job_id")
		}
		status := tracker.Get(jobID)
		if status == nil {
			return jsonError(c, http.StatusNotFound, "job not found")
		}
		return c.JSON(http.StatusOK, status)
	}
}

// detectArtifactType does a lightweight header parse to determine the type.
func detectArtifactType(data []byte) (gemara.ArtifactType, error) {
	var hdr struct {
		Metadata gemara.Metadata `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(data, &hdr); err != nil {
		return gemara.InvalidArtifact, fmt.Errorf("parse YAML header: %w", err)
	}
	if hdr.Metadata.Type == gemara.InvalidArtifact {
		return gemara.InvalidArtifact, fmt.Errorf("missing or invalid metadata.type field")
	}
	return hdr.Metadata.Type, nil
}

func flattenEvaluation(ctx context.Context, data []byte) ([]ingest.EvidenceRow, string, error) {
	f := &bytesFetcher{data: data}
	evalLog, err := gemara.Load[gemara.EvaluationLog](ctx, f, "upload.yaml")
	if err != nil {
		return nil, "", fmt.Errorf("parse EvaluationLog: %w", err)
	}
	policyID := derivePolicyID(evalLog.Metadata.MappingReferences)
	rows, err := ingest.FlattenEvaluationLog(evalLog, policyID)
	return rows, policyID, err
}

func flattenEnforcement(ctx context.Context, data []byte) ([]ingest.EvidenceRow, string, error) {
	f := &bytesFetcher{data: data}
	enfLog, err := gemara.Load[gemara.EnforcementLog](ctx, f, "upload.yaml")
	if err != nil {
		return nil, "", fmt.Errorf("parse EnforcementLog: %w", err)
	}
	policyID := derivePolicyID(enfLog.Metadata.MappingReferences)
	rows, err := ingest.FlattenEnforcementLog(enfLog, policyID)
	return rows, policyID, err
}

// bytesFetcher satisfies gemara.Fetcher for in-memory YAML.
type bytesFetcher struct {
	data []byte
}

func (b *bytesFetcher) Fetch(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.data)), nil
}

func generateJobID() string {
	return uuid.New().String()
}

// derivePolicyID extracts a policy reference from mapping-references.
func derivePolicyID(refs []gemara.MappingReference) string {
	for _, r := range refs {
		if r.Title == "Policy" || r.Id == "policy" {
			return r.Id
		}
	}
	if len(refs) > 0 {
		return refs[0].Id
	}
	return ""
}
