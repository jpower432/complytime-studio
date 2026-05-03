// SPDX-License-Identifier: Apache-2.0

package store

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

//go:embed testdata/golden/post_evidence_ingest.json
var goldenPostEvidenceIngest []byte

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

const minimalEvalLog = `metadata:
  type: EvaluationLog
  id: test-eval-001
  gemara-version: "1.0.0"
  description: test
  date: "2026-04-25T12:00:00Z"
  author:
    id: test
    name: Test
    type: Software
  mapping-references:
    - id: test-policy
      title: Policy
      version: "1.0.0"
result: Passed
target:
  id: test-target
  name: Test Target
  type: Software
evaluations:
  - name: Test Control
    result: Passed
    message: ok
    control:
      reference-id: test-policy
      entry-id: C-1
    assessment-logs:
      - requirement:
          reference-id: test-policy
          entry-id: C-1.01
        description: test
        result: Passed
        message: passed
        applicability: []
        steps: []
        start: "2026-04-25T12:00:00Z"
`

func TestContract_POST_api_evidence_ingest_GemaraYAML(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	e := echo.New()
	g := e.Group("/api")
	Register(g, Stores{Evidence: fake})

	req := httptest.NewRequest(http.MethodPost, "/api/evidence/ingest", strings.NewReader(minimalEvalLog))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	want := normJSON(t, goldenPostEvidenceIngest)
	got := normJSON(t, rec.Body.Bytes())
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	if len(fake.inserted) != 1 {
		t.Fatalf("expected 1 inserted record, got %d", len(fake.inserted))
	}
	rec0 := fake.inserted[0]
	if rec0.PolicyID != "test-policy" {
		t.Errorf("PolicyID = %q, want %q", rec0.PolicyID, "test-policy")
	}
	if rec0.TargetID != "test-target" {
		t.Errorf("TargetID = %q, want %q", rec0.TargetID, "test-target")
	}
	if rec0.EvalResult != "Passed" {
		t.Errorf("EvalResult = %q, want %q", rec0.EvalResult, "Passed")
	}
}

const minimalEnfLog = `metadata:
  type: EnforcementLog
  id: test-enf-001
  gemara-version: "1.0.0"
  description: test enforcement
  date: "2026-04-25T12:00:00Z"
  author:
    id: enforcer
    name: Enforcer
    type: Software
  mapping-references:
    - id: test-policy
      title: Policy
      version: "1.0.0"
result: Passed
target:
  id: test-target
  name: Test Target
  type: Software
actions:
  - name: Block unsigned image
    disposition: Enforced
    start: "2026-04-25T12:00:00Z"
    method:
      reference-id: test-policy
      entry-id: C-1
    justification:
      reason: Policy violation
      exceptions: []
      assessments:
        - result: Passed
          requirement:
            reference-id: test-policy
            entry-id: C-1.01
`

func TestContract_POST_api_evidence_ingest_EnforcementLog(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	e := echo.New()
	g := e.Group("/api")
	Register(g, Stores{Evidence: fake})

	req := httptest.NewRequest(http.MethodPost, "/api/evidence/ingest", strings.NewReader(minimalEnfLog))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["type"] != "EnforcementLog" {
		t.Errorf("type = %v, want EnforcementLog", resp["type"])
	}
	if resp["policy_id"] != "test-policy" {
		t.Errorf("policy_id = %v, want test-policy", resp["policy_id"])
	}
	if len(fake.inserted) == 0 {
		t.Fatal("expected at least 1 inserted record")
	}
}

func TestContract_POST_api_evidence_ingest_EmptyBody(t *testing.T) {
	t.Parallel()
	e := echo.New()
	g := e.Group("/api")
	Register(g, Stores{Evidence: &fakeEvidenceStore{}})

	req := httptest.NewRequest(http.MethodPost, "/api/evidence/ingest", strings.NewReader(""))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestContract_POST_api_evidence_ingest_InvalidYAML(t *testing.T) {
	t.Parallel()
	e := echo.New()
	g := e.Group("/api")
	Register(g, Stores{Evidence: &fakeEvidenceStore{}})

	req := httptest.NewRequest(http.MethodPost, "/api/evidence/ingest", strings.NewReader("not: valid: yaml: ["))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestContract_POST_api_evidence_ingest_UnsupportedType(t *testing.T) {
	t.Parallel()
	e := echo.New()
	g := e.Group("/api")
	Register(g, Stores{Evidence: &fakeEvidenceStore{}})

	body := "metadata:\n  type: ThreatCatalog\n  id: bad\n  gemara-version: \"1.0.0\"\n"
	req := httptest.NewRequest(http.MethodPost, "/api/evidence/ingest", strings.NewReader(body))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unsupported artifact type") {
		t.Errorf("expected unsupported type error, got %s", rec.Body.String())
	}
}

func TestContract_POST_api_evidence_ingest_InsertFailure(t *testing.T) {
	t.Parallel()
	e := echo.New()
	g := e.Group("/api")
	Register(g, Stores{Evidence: &failingEvidenceStore{}})

	req := httptest.NewRequest(http.MethodPost, "/api/evidence/ingest", strings.NewReader(minimalEvalLog))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d %s", rec.Code, rec.Body.String())
	}
}
