// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//go:embed testdata/golden/post_evidence_created.json
var goldenPostEvidenceCreated []byte

//go:embed testdata/golden/post_evidence_upload.json
var goldenPostEvidenceUpload []byte

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

func TestContract_POST_api_evidence_upload_JSONGolden(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake})

	csv := "policy_id,eval_result,collected_at,target_id,control_id,rule_id\n" +
		"pol-g,Passed,2026-04-25T12:00:00Z,tg,cg,rg\n"
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "rows.csv")
	_, _ = io.WriteString(part, csv)
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/evidence/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	want := normJSON(t, goldenPostEvidenceUpload)
	got := normJSON(t, rec.Body.Bytes())
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
