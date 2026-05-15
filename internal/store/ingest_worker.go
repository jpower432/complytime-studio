// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"fmt"
	"log/slog"

	gemara "github.com/gemaraproj/go-gemara"

	"github.com/complytime/complytime-studio/internal/events"
	"github.com/complytime/complytime-studio/internal/ingest"
)

// IngestWorker returns a handler for async ingest events. It flattens the
// raw Gemara YAML, inserts evidence, publishes the evidence event (which
// triggers the certifier), and updates the job tracker.
func IngestWorker(
	ctx context.Context,
	evidence EvidenceStore,
	pub EventPublisher,
	tracker *IngestTracker,
) func(events.IngestRawEvent) {
	return func(evt events.IngestRawEvent) {
		slog.Info("async ingest started", "job_id", evt.JobID)

		artifactType, err := detectArtifactType(evt.YAML)
		if err != nil {
			tracker.Fail(evt.JobID, fmt.Sprintf("invalid artifact: %v", err))
			slog.Warn("async ingest: invalid artifact", "job_id", evt.JobID, "error", err)
			return
		}

		var rows []ingest.EvidenceRow
		var policyID string

		switch artifactType {
		case gemara.EvaluationLogArtifact:
			rows, policyID, err = flattenEvaluation(ctx, evt.YAML)
		case gemara.EnforcementLogArtifact:
			rows, policyID, err = flattenEnforcement(ctx, evt.YAML)
		default:
			tracker.Fail(evt.JobID, fmt.Sprintf("unsupported artifact type: %s", artifactType))
			return
		}
		if err != nil {
			tracker.Fail(evt.JobID, fmt.Sprintf("flatten failed: %v", err))
			slog.Warn("async ingest: flatten failed", "job_id", evt.JobID, "error", err)
			return
		}

		records := toEvidenceRecords(rows)
		count, err := evidence.InsertEvidence(ctx, records)
		if err != nil {
			tracker.Fail(evt.JobID, fmt.Sprintf("insert failed: %v", err))
			slog.Error("async ingest: insert failed", "job_id", evt.JobID, "error", err)
			return
		}

		if pub != nil && count > 0 && policyID != "" {
			pub.PublishEvidence(policyID, count)
		}

		tracker.Complete(evt.JobID, count, policyID)
		slog.Info("async ingest completed",
			"job_id", evt.JobID,
			"inserted", count,
			"policy_id", policyID,
		)
	}
}
