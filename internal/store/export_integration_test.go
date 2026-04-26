// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	chclient "github.com/complytime/complytime-studio/internal/clickhouse"
	"github.com/complytime/complytime-studio/internal/store"
	"github.com/google/uuid"
	tcch "github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

// TestAuditorExport_ClickHouse runs against a real ClickHouse instance via
// testcontainers when RUN_CLICKHOUSE_INTEGRATION=1 (requires Docker).
func TestAuditorExport_ClickHouse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if os.Getenv("RUN_CLICKHOUSE_INTEGRATION") == "" {
		t.Skip("set RUN_CLICKHOUSE_INTEGRATION=1 to run (requires Docker)")
	}

	ctx := context.Background()
	ctr, err := tcch.Run(ctx, "clickhouse/clickhouse-server:24.8-alpine")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = ctr.Terminate(ctx)
	})

	addr, err := ctr.ConnectionHost(ctx)
	if err != nil {
		t.Fatal(err)
	}

	cli, err := chclient.New(ctx, chclient.Config{
		Addr:     addr,
		Database: ctr.DbName,
		User:     ctr.User,
		Password: ctr.Password,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.EnsureSchema(ctx, 1); err != nil {
		t.Fatal(err)
	}
	st := store.New(cli.Conn())

	policyID := uuid.New().String()
	if err := st.InsertPolicy(ctx, store.Policy{
		PolicyID:     policyID,
		Title:        "Integration policy",
		OCIReference: "oci://test",
		Content:      "metadata:\n  type: Policy\n",
	}); err != nil {
		t.Fatal(err)
	}

	winStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	winEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)

	// Empty window: evidence only outside the audit range.
	outside := winStart.Add(-48 * time.Hour)
	if _, err := st.InsertEvidence(ctx, []store.EvidenceRecord{{
		PolicyID:    policyID,
		TargetID:    "tgt-out",
		ControlID:   "AC-1",
		RuleID:      "rule-1",
		EvalResult:  "Passed",
		CollectedAt: outside,
	}}); err != nil {
		t.Fatal(err)
	}
	emptyRows, err := st.ListRequirementMatrix(ctx, store.RequirementFilter{
		PolicyID: policyID, Start: winStart, End: winEnd, Limit: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(emptyRows) != 0 {
		t.Fatalf("empty window: got %d rows", len(emptyRows))
	}

	// Mixed evidence inside the window.
	inside := winStart.Add(2 * time.Hour)
	eidMix := uuid.New().String()
	if _, err := st.InsertEvidence(ctx, []store.EvidenceRecord{{
		EvidenceID:    eidMix,
		PolicyID:      policyID,
		TargetID:      "tgt-in",
		ControlID:     "AC-2",
		RuleID:        "rule-2",
		RequirementID: "req-mix",
		EvalResult:    "Passed",
		CollectedAt:   inside,
	}}); err != nil {
		t.Fatal(err)
	}
	mixed, err := st.ListRequirementMatrix(ctx, store.RequirementFilter{
		PolicyID: policyID, Start: winStart, End: winEnd, Limit: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mixed) < 1 {
		t.Fatalf("mixed: want rows, got %d", len(mixed))
	}

	// All classified as gaps for one control (evidence present but "No Evidence").
	eidGap := uuid.New().String()
	if _, err := st.InsertEvidence(ctx, []store.EvidenceRecord{{
		EvidenceID:  eidGap,
		PolicyID:    policyID,
		TargetID:    "tgt-gap",
		ControlID:   "AC-GAP",
		RuleID:      "rule-gap",
		EvalResult:  "Failed",
		CollectedAt: inside,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertEvidenceAssessments(ctx, []store.EvidenceAssessment{{
		EvidenceID:     eidGap,
		PolicyID:       policyID,
		Classification: "No Evidence",
		AssessedAt:     time.Now().UTC(),
		AssessedBy:     "integration",
	}}); err != nil {
		t.Fatal(err)
	}
	gapView, err := st.ListRequirementMatrix(ctx, store.RequirementFilter{
		PolicyID: policyID, Start: winStart, End: winEnd, Limit: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	var gapRows int
	for _, r := range gapView {
		if store.IsGapRow(r) {
			gapRows++
		}
	}
	if gapRows < 1 {
		t.Fatalf("expected at least one gap-classified row, matrix=%+v", gapView)
	}

	// Large batch within export limits (smoke).
	var batch []store.EvidenceRecord
	for i := 0; i < 120; i++ {
		batch = append(batch, store.EvidenceRecord{
			PolicyID:    policyID,
			TargetID:    "bulk",
			ControlID:   "BC-1",
			RuleID:      "br",
			EvalResult:  "Passed",
			CollectedAt: inside.Add(time.Duration(i) * time.Minute),
		})
	}
	if _, err := st.InsertEvidence(ctx, batch); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	store.Register(mux, store.Stores{
		Policies:     st,
		Mappings:     st,
		Evidence:     st,
		AuditLogs:    st,
		Requirements: st,
	})

	q := "/api/export/csv?policy_id=" + policyID +
		"&audit_start=" + winStart.Format(time.RFC3339) +
		"&audit_end=" + winEnd.Format(time.RFC3339)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, q, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("csv export: %d %s", rec.Code, rec.Body.String())
	}
	cr := csv.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if _, err := cr.Read(); err != nil { // header
		t.Fatal(err)
	}
	if _, err := cr.Read(); err != nil {
		t.Fatalf("csv data row: %v", err)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/export/excel?policy_id="+policyID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("excel: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/export/pdf?policy_id="+policyID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("pdf: %d", rec.Code)
	}
	b := rec.Body.Bytes()
	if len(b) < 4 || string(b[:4]) != "%PDF" {
		t.Fatal("pdf magic missing")
	}
}
