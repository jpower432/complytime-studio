// SPDX-License-Identifier: Apache-2.0

package store

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//go:embed testdata/golden/post_evidence_created.json
var goldenPostEvidenceCreated []byte

func normJSON(t *testing.T, b []byte) string {
	t.Helper()
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestContract_POST_api_evidence_JSONResponseGolden(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake})

	body := `[{"policy_id":"p-golden","target_id":"t","control_id":"c","rule_id":"r","eval_result":"Passed","collected_at":"2026-04-25T12:00:00Z"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/evidence", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	want := normJSON(t, goldenPostEvidenceCreated)
	got := normJSON(t, rec.Body.Bytes())
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestContract_POST_api_evidence_upload_Returns410(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	Register(mux, Stores{})

	req := httptest.NewRequest(
		http.MethodPost, "/api/evidence/upload", nil,
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone, got %d %s", rec.Code, rec.Body.String())
	}
}
