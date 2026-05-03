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

// IngestGemaraHandler returns an http.HandlerFunc that accepts raw Gemara
// artifact YAML (EvaluationLog or EnforcementLog), flattens it into evidence
// rows, and inserts them into the store.
func IngestGemaraHandler(s EvidenceStore, pub EventPublisher) http.HandlerFunc {
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

		artifactType, err := detectArtifactType(body)
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
				"errors": []string{fmt.Sprintf("invalid artifact: %v", err)},
			})
			return
		}

		var rows []ingest.EvidenceRow
		var policyID string

		switch artifactType {
		case gemara.EvaluationLogArtifact:
			rows, policyID, err = flattenEvaluation(r.Context(), body)
		case gemara.EnforcementLogArtifact:
			rows, policyID, err = flattenEnforcement(r.Context(), body)
		default:
			httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
				"errors": []string{fmt.Sprintf(
					"unsupported artifact type %q — expected EvaluationLog or EnforcementLog",
					artifactType,
				)},
			})
			return
		}
	if err != nil {
		slog.Error("artifact flatten failed", "type", artifactType, "error", err)
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"errors": []string{"failed to process artifact — check server logs for details"},
		})
		return
	}

		records := toEvidenceRecords(rows)
		count, err := s.InsertEvidence(r.Context(), records)
		if err != nil {
			slog.Error("ingest insert failed", "error", err)
			http.Error(w, "insert failed", http.StatusInternalServerError)
			return
		}

		if pub != nil && count > 0 && policyID != "" {
			pub.PublishEvidence(policyID, count)
		}

		slog.Info("gemara artifact ingested",
			"type", artifactType,
			"policy_id", policyID,
			"rows", count,
		)

		httputil.WriteJSON(w, http.StatusCreated, map[string]any{
			"inserted":  count,
			"type":      artifactType.String(),
			"policy_id": policyID,
		})
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
