// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"fmt"

	gemara "github.com/gemaraproj/go-gemara"

	"github.com/complytime/complytime-studio/internal/ingest"
)

// ParseAndFlattenEvidence parses Gemara YAML (EvaluationLog or EnforcementLog)
// and returns flattened evidence records ready for insertion. Extracted from the
// HTTP ingest handler so the ConnectRPC service can reuse the same logic.
func ParseAndFlattenEvidence(ctx context.Context, data []byte) ([]EvidenceRecord, string, error) {
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty YAML content")
	}
	artifactType, err := detectArtifactType(data)
	if err != nil {
		return nil, "", err
	}

	var rows []ingest.EvidenceRow
	var policyID string

	switch artifactType {
	case gemara.EvaluationLogArtifact:
		rows, policyID, err = flattenEvaluation(ctx, data)
	case gemara.EnforcementLogArtifact:
		rows, policyID, err = flattenEnforcement(ctx, data)
	default:
		return nil, "", fmt.Errorf(
			"unsupported artifact type %q — expected EvaluationLog or EnforcementLog",
			artifactType,
		)
	}
	if err != nil {
		return nil, "", fmt.Errorf("flatten %s: %w", artifactType, err)
	}

	return toEvidenceRecords(rows), policyID, nil
}
