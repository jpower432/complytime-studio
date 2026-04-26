// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeEvidenceStore struct {
	inserted []EvidenceRecord
	query    []EvidenceRecord
}

func (f *fakeEvidenceStore) InsertEvidence(ctx context.Context, records []EvidenceRecord) (int, error) {
	f.inserted = append([]EvidenceRecord{}, records...)
	return len(records), nil
}

func (f *fakeEvidenceStore) QueryEvidence(ctx context.Context, filt EvidenceFilter) ([]EvidenceRecord, error) {
	out := make([]EvidenceRecord, len(f.query))
	copy(out, f.query)
	return out, nil
}

func evidencePOSTBody(includeSourceRegistry bool, sourceRegistry string) string {
	row := map[string]string{
		"policy_id":    "pol-1",
		"target_id":    "tgt-1",
		"control_id":   "ctl-1",
		"rule_id":      "rule-1",
		"eval_result":  "Passed",
		"collected_at": "2026-04-25T12:00:00Z",
	}
	if includeSourceRegistry {
		row["source_registry"] = sourceRegistry
	}
	b, _ := json.Marshal([]map[string]string{row})
	return string(b)
}

func TestIngestEvidenceHandler_SourceRegistryOptional(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake})

	t.Run("with_source_registry", func(t *testing.T) {
		body := evidencePOSTBody(true, "https://registry.example/v2")
		req := httptest.NewRequest(http.MethodPost, "/api/evidence", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
		}
		if len(fake.inserted) != 1 {
			t.Fatalf("got %d rows", len(fake.inserted))
		}
		if fake.inserted[0].SourceRegistry != "https://registry.example/v2" {
			t.Fatalf("SourceRegistry %q", fake.inserted[0].SourceRegistry)
		}
	})

	t.Run("omitted_source_registry", func(t *testing.T) {
		f2 := &fakeEvidenceStore{}
		m2 := http.NewServeMux()
		Register(m2, Stores{Evidence: f2})
		body := evidencePOSTBody(false, "")
		req := httptest.NewRequest(http.MethodPost, "/api/evidence", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		m2.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d", rec.Code)
		}
		if f2.inserted[0].SourceRegistry != "" {
			t.Fatalf("expected empty SourceRegistry, got %q", f2.inserted[0].SourceRegistry)
		}
	})
}

func TestQueryEvidenceHandler_SourceRegistryJSON(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{
		query: []EvidenceRecord{
			{
				EvidenceID:     "ev-1",
				PolicyID:       "pol-1",
				TargetID:       "tgt-1",
				ControlID:      "c1",
				RuleID:         "r1",
				EvalResult:     "Passed",
				SourceRegistry: "oci://boundary.registry/ns/repo",
				CollectedAt:    time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake})

	req := httptest.NewRequest(http.MethodGet, "/api/evidence?policy_id=pol-1&limit=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var got []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len %d", len(got))
	}
	if got[0]["source_registry"] != "oci://boundary.registry/ns/repo" {
		t.Fatalf("source_registry field: %v", got[0]["source_registry"])
	}
}

func TestIngestEvidenceHandler_InvalidEnumRejected(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake})

	body := `[{"policy_id":"p","target_id":"t","control_id":"c","rule_id":"r","eval_result":"Passed","collected_at":"2026-04-25T15:00:00Z","compliance_status":"Partial"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/evidence", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d: %s", w.Code, w.Body.String())
	}
	if len(fake.inserted) != 0 {
		t.Fatalf("expected no insert, got %d", len(fake.inserted))
	}
}

func TestIngestEvidenceHandler_RoundTripSourceRegistryREST(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake})

	body := `[{"policy_id":"p","target_id":"t","control_id":"c","rule_id":"r","eval_result":"Passed","collected_at":"2026-04-25T15:00:00Z","source_registry":"https://reg.test/"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/evidence", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST %d: %s", w.Code, w.Body.String())
	}

	fake.query = fake.inserted
	req2 := httptest.NewRequest(http.MethodGet, "/api/evidence?policy_id=p&limit=5", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET %d", w2.Code)
	}
	payload, _ := io.ReadAll(w2.Body)
	if !strings.Contains(string(payload), "https://reg.test/") {
		t.Fatalf("GET body missing source_registry: %s", payload)
	}
}
