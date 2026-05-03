// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"os"
	"testing"
	"time"

	pgstore "github.com/complytime/complytime-studio/internal/postgres"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("POSTGRES_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_TEST_URL not set — skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := pgstore.New(ctx, pgstore.Config{URL: url})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := client.EnsureSchema(ctx); err != nil {
		client.Close()
		t.Fatalf("schema: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		pool := client.Pool()
		_, _ = pool.Exec(bg, "DELETE FROM evidence_assessments")
		_, _ = pool.Exec(bg, "DELETE FROM evidence")
		_, _ = pool.Exec(bg, "DELETE FROM policies")
		client.Close()
	})
	return New(client.Pool())
}

func TestIntegration_InsertAndQueryEvidence(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	records := []EvidenceRecord{
		{
			EvidenceID:       "ev-int-1",
			PolicyID:         "pol-int",
			TargetID:         "tgt-1",
			TargetName:       "Web App",
			TargetType:       "Software",
			RuleID:           "r-1",
			ControlID:        "C-1",
			RequirementID:    "C-1.01",
			EvalResult:       "Passed",
			ComplianceStatus: "Compliant",
			EnrichmentStatus: "Success",
			CollectedAt:      time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			EvidenceID:       "ev-int-2",
			PolicyID:         "pol-int",
			TargetID:         "tgt-1",
			ControlID:        "C-2",
			RequirementID:    "C-2.01",
			RuleID:           "r-2",
			EvalResult:       "Failed",
			ComplianceStatus: "Non-Compliant",
			EnrichmentStatus: "Success",
			CollectedAt:      time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC),
		},
	}

	n, err := st.InsertEvidence(ctx, records)
	if err != nil {
		t.Fatalf("InsertEvidence: %v", err)
	}
	if n != 2 {
		t.Fatalf("inserted %d, want 2", n)
	}

	got, err := st.QueryEvidence(ctx, EvidenceFilter{PolicyIDs: []string{"pol-int"}, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvidence: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("queried %d records, want 2", len(got))
	}

	byControl := make(map[string]EvidenceRecord)
	for _, r := range got {
		byControl[r.ControlID] = r
	}
	if r, ok := byControl["C-1"]; !ok || r.EvalResult != "Passed" {
		t.Errorf("C-1: got %+v, want Passed", byControl["C-1"])
	}
	if r, ok := byControl["C-2"]; !ok || r.EvalResult != "Failed" {
		t.Errorf("C-2: got %+v, want Failed", byControl["C-2"])
	}
}

func TestIntegration_InsertEvidence_Upsert(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	original := []EvidenceRecord{{
		EvidenceID:       "ev-upsert",
		PolicyID:         "pol-upsert",
		TargetID:         "tgt-1",
		ControlID:        "C-1",
		RequirementID:    "C-1.01",
		RuleID:           "r-1",
		EvalResult:       "Failed",
		ComplianceStatus: "Non-Compliant",
		EnrichmentStatus: "Success",
		CollectedAt:      time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}}
	if _, err := st.InsertEvidence(ctx, original); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	updated := []EvidenceRecord{{
		EvidenceID:       "ev-upsert",
		PolicyID:         "pol-upsert",
		TargetID:         "tgt-1",
		ControlID:        "C-1",
		RequirementID:    "C-1.01",
		RuleID:           "r-1",
		EvalResult:       "Passed",
		ComplianceStatus: "Compliant",
		EnrichmentStatus: "Success",
		CollectedAt:      time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC),
	}}
	if _, err := st.InsertEvidence(ctx, updated); err != nil {
		t.Fatalf("upsert insert: %v", err)
	}

	got, err := st.QueryEvidence(ctx, EvidenceFilter{PolicyIDs: []string{"pol-upsert"}, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvidence: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 record after upsert, got %d", len(got))
	}
	if got[0].EvalResult != "Passed" {
		t.Errorf("expected upserted result Passed, got %q", got[0].EvalResult)
	}
}

func TestIntegration_InsertAndListPolicies(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	p := Policy{
		PolicyID:     "pol-test-1",
		Title:        "Test Policy",
		Version:      "1.0.0",
		OCIReference: "oci://example.com/policies/test:1.0.0",
		Content:      "policy: content",
	}
	if err := st.InsertPolicy(ctx, p); err != nil {
		t.Fatalf("InsertPolicy: %v", err)
	}

	policies, err := st.ListPolicies(ctx)
	if err != nil {
		t.Fatalf("ListPolicies: %v", err)
	}
	found := false
	for _, pol := range policies {
		if pol.PolicyID == "pol-test-1" {
			found = true
			if pol.Title != "Test Policy" {
				t.Errorf("Title = %q, want %q", pol.Title, "Test Policy")
			}
		}
	}
	if !found {
		t.Fatal("inserted policy not found in ListPolicies")
	}

	got, err := st.GetPolicy(ctx, "pol-test-1")
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if got.OCIReference != p.OCIReference {
		t.Errorf("OCIReference = %q, want %q", got.OCIReference, p.OCIReference)
	}
}

func TestIntegration_QueryEvidence_Filters(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	records := []EvidenceRecord{
		{
			EvidenceID: "ev-filt-1", PolicyID: "pol-a", TargetID: "tgt-1",
			ControlID: "C-1", RequirementID: "C-1.01", RuleID: "r-1",
			EvalResult: "Passed", ComplianceStatus: "Compliant",
			EnrichmentStatus: "Success",
			CollectedAt:      time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			EvidenceID: "ev-filt-2", PolicyID: "pol-b", TargetID: "tgt-2",
			ControlID: "C-2", RequirementID: "C-2.01", RuleID: "r-2",
			EvalResult: "Failed", ComplianceStatus: "Non-Compliant",
			EnrichmentStatus: "Success",
			CollectedAt:      time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC),
		},
	}
	if _, err := st.InsertEvidence(ctx, records); err != nil {
		t.Fatalf("InsertEvidence: %v", err)
	}

	got, err := st.QueryEvidence(ctx, EvidenceFilter{PolicyIDs: []string{"pol-a"}, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvidence pol-a: %v", err)
	}
	if len(got) != 1 || got[0].EvidenceID != "ev-filt-1" {
		t.Fatalf("expected ev-filt-1 only, got %d records", len(got))
	}

	got, err = st.QueryEvidence(ctx, EvidenceFilter{ControlID: "C-2", Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvidence C-2: %v", err)
	}
	if len(got) != 1 || got[0].EvidenceID != "ev-filt-2" {
		t.Fatalf("expected ev-filt-2 only, got %d records", len(got))
	}
}
