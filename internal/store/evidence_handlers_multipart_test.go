// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubBlob struct {
	ref string
	err error
}

func (s *stubBlob) Upload(ctx context.Context, key string, r io.Reader, size int64) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.ref != "" {
		return s.ref, nil
	}
	return "s3://stub-bucket/" + key, nil
}

func TestIngestEvidenceHandler_MultipartFileRequiresBlob(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake, Blob: nil})

	body, ct := multipartEvidenceBody(t, `[{"policy_id":"p","target_id":"t","control_id":"c","rule_id":"r","eval_result":"Passed","collected_at":"2026-04-25T12:00:00Z"}]`, "note.txt", "hello")
	req := httptest.NewRequest(http.MethodPost, "/api/evidence", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "blob storage not configured") {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestIngestEvidenceHandler_MultipartWithBlobSetsBlobRef(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	b := &stubBlob{ref: "s3://testbk/evidence/obj"}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake, Blob: b})

	body, ct := multipartEvidenceBody(t, `[{"policy_id":"p","target_id":"t","control_id":"c","rule_id":"r","eval_result":"Passed","collected_at":"2026-04-25T12:00:00Z"}]`, "a.pdf", "x")
	req := httptest.NewRequest(http.MethodPost, "/api/evidence", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if len(fake.inserted) != 1 || fake.inserted[0].BlobRef != "s3://testbk/evidence/obj" {
		t.Fatalf("inserted %+v", fake.inserted)
	}
}

func TestIngestEvidenceHandler_MultipartAttachmentFieldName(t *testing.T) {
	t.Parallel()
	fake := &fakeEvidenceStore{}
	b := &stubBlob{ref: "s3://bk/k"}
	mux := http.NewServeMux()
	Register(mux, Stores{Evidence: fake, Blob: b})

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("data", `[{"policy_id":"p","target_id":"t","control_id":"c","rule_id":"r","eval_result":"Passed","collected_at":"2026-04-25T12:00:00Z"}]`)
	part, _ := w.CreateFormFile("attachment", "f.txt")
	_, _ = part.Write([]byte("z"))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/evidence", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if fake.inserted[0].BlobRef != "s3://bk/k" {
		t.Fatalf("blob ref %q", fake.inserted[0].BlobRef)
	}
}

func multipartEvidenceBody(t *testing.T, dataJSON, fileName, fileContent string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("data", dataJSON); err != nil {
		t.Fatal(err)
	}
	part, err := w.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(fileContent)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, w.FormDataContentType()
}
