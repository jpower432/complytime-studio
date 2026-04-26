// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/complytime/complytime-studio/internal/auth"
	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/store"
	"github.com/xuri/excelize/v2"
)

func TestParseExportQuery(t *testing.T) {
	t.Parallel()
	q := url.Values{
		"policy_id":   {"p-1"},
		"audit_start": {"2026-01-01"},
		"audit_end":   {"2026-01-31"},
	}
	policy, auditID, f, err := store.ParseExportQuery(q)
	if err != nil {
		t.Fatal(err)
	}
	if policy != "p-1" || auditID != "" {
		t.Fatalf("policy/audit: %q %q", policy, auditID)
	}
	if f.PolicyID != "p-1" || f.Start.IsZero() || f.End.IsZero() {
		t.Fatalf("filter: %+v", f)
	}

	_, _, _, err = store.ParseExportQuery(url.Values{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseExportQuery_auditEndBeforeStart(t *testing.T) {
	t.Parallel()
	_, _, _, err := store.ParseExportQuery(url.Values{
		"policy_id":   {"p"},
		"audit_start": {"2026-02-01"},
		"audit_end":   {"2026-01-01"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitizeExportFilenamePart(t *testing.T) {
	t.Parallel()
	s := store.SanitizeExportFilenamePart(`ab/cd\ef*g`)
	if strings.ContainsAny(s, `/\*`) {
		t.Fatalf("got %q", s)
	}
}

func TestBuildExportSummaryAgg_andGapRows(t *testing.T) {
	t.Parallel()
	rows := []store.RequirementRow{
		{RequirementID: "r1", EvidenceCount: 2, Classification: "Healthy"},
		{RequirementID: "r2", EvidenceCount: 0, Classification: "Unassessed"},
		{RequirementID: "r3", EvidenceCount: 1, Classification: "No Evidence"},
	}
	agg := store.BuildExportSummaryAgg(rows, "T", "v1")
	if agg.TotalRequirements != 3 || agg.RequirementsWithEvidence != 2 {
		t.Fatalf("agg: %+v", agg)
	}
	if !store.IsGapRow(rows[1]) || !store.IsGapRow(rows[2]) {
		t.Fatal("expected gap rows")
	}
	if store.IsGapRow(rows[0]) {
		t.Fatal("healthy with evidence is not gap")
	}
}

func TestLoadExportMatrix_rowLimit(t *testing.T) {
	t.Parallel()
	m := &mockRequirementStore{}
	m.matrixHook = func(ctx context.Context, f store.RequirementFilter) ([]store.RequirementRow, error) {
		out := make([]store.RequirementRow, consts.MaxExportRequirementRows+1)
		return out, nil
	}
	_, err := store.LoadExportMatrix(context.Background(), m, store.RequirementFilter{PolicyID: "p"})
	if !errors.Is(err, store.ErrExportRowLimit) {
		t.Fatalf("got %v", err)
	}
}

func TestExportCSV_smokeParse(t *testing.T) {
	t.Parallel()
	m := &mockRequirementStore{}
	m.matrixHook = func(ctx context.Context, f store.RequirementFilter) ([]store.RequirementRow, error) {
		if f.PolicyID != "pol1" {
			t.Fatalf("policy: %+v", f)
		}
		return []store.RequirementRow{
			{CatalogID: "c", ControlID: "ctl", ControlTitle: "T",
				RequirementID: "r1", RequirementText: "text",
				EvidenceCount: 1, LatestEvidence: "2026-01-15T00:00:00Z", Classification: "Healthy",
			},
		}, nil
	}
	mux := http.NewServeMux()
	store.Register(mux, testStoresWithRequirements(m))
	req := httptest.NewRequest(http.MethodGet, "/api/export/csv?policy_id=pol1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	r := csv.NewReader(strings.NewReader(body))
	headers, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}
	if headers[0] != "catalog_id" {
		t.Fatalf("headers: %v", headers)
	}
	row, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}
	if row[3] != "r1" {
		t.Fatalf("row: %v", row)
	}
	cd := rec.Header().Get("Content-Disposition")
	if !strings.HasPrefix(cd, "attachment; filename=") {
		t.Fatalf("Content-Disposition: %q", cd)
	}
}

func TestExportExcel_smokeParseXLSX(t *testing.T) {
	t.Parallel()
	m := &mockRequirementStore{}
	m.matrixHook = func(ctx context.Context, f store.RequirementFilter) ([]store.RequirementRow, error) {
		return []store.RequirementRow{
			{ControlID: "C1", RequirementID: "R1", EvidenceCount: 0, Classification: "No Evidence"},
		}, nil
	}
	st := testStoresWithRequirements(m)
	st.Evidence = stubEvidenceStore{}
	st.Policies = stubPolicyStore{}
	st.AuditLogs = stubAuditLogStore{}

	mux := http.NewServeMux()
	store.Register(mux, st)
	req := httptest.NewRequest(http.MethodGet, "/api/export/excel?policy_id=p1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	f, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	idx, err := f.GetSheetIndex("Executive Summary")
	if err != nil || idx == -1 {
		t.Fatalf("Executive Summary sheet: idx=%d err=%v", idx, err)
	}
}

func TestExportPDF_startsWithPDFMagic(t *testing.T) {
	t.Parallel()
	m := &mockRequirementStore{}
	m.matrixHook = func(ctx context.Context, f store.RequirementFilter) ([]store.RequirementRow, error) {
		return []store.RequirementRow{
			{ControlID: "C1", RequirementID: "R1"},
		}, nil
	}
	st := testStoresWithRequirements(m)
	st.Policies = stubPolicyStore{}
	st.AuditLogs = stubAuditLogStore{}

	mux := http.NewServeMux()
	store.Register(mux, st)
	req := httptest.NewRequest(http.MethodGet, "/api/export/pdf?policy_id=p1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	b := rec.Body.Bytes()
	if len(b) < 4 || string(b[:4]) != "%PDF" {
		t.Fatalf("not PDF: %q", b[:min(8, len(b))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type stubEvidenceStore struct{}

func (stubEvidenceStore) InsertEvidence(ctx context.Context, records []store.EvidenceRecord) (int, error) {
	return 0, nil
}

func (stubEvidenceStore) QueryEvidence(ctx context.Context, f store.EvidenceFilter) ([]store.EvidenceRecord, error) {
	return nil, nil
}

type stubAuditLogStore struct{}

func (stubAuditLogStore) InsertAuditLog(ctx context.Context, a store.AuditLog) error {
	return nil
}

func (stubAuditLogStore) ListAuditLogs(ctx context.Context, policyID string, start, end time.Time, limit int) ([]store.AuditLog, error) {
	return nil, nil
}

func (stubAuditLogStore) GetAuditLog(ctx context.Context, auditID string) (*store.AuditLog, error) {
	return nil, errors.New("not found")
}

func TestExportContentDisposition_filenameSanitized(t *testing.T) {
	t.Parallel()
	m := &mockRequirementStore{}
	m.matrixHook = func(ctx context.Context, f store.RequirementFilter) ([]store.RequirementRow, error) {
		return nil, nil
	}
	st := testStoresWithRequirements(m)
	st.Evidence = stubEvidenceStore{}
	st.Policies = stubPolicyStore{}
	st.AuditLogs = stubAuditLogStore{}

	mux := http.NewServeMux()
	store.Register(mux, st)
	rawID := `weird/id\*name`
	req := httptest.NewRequest(http.MethodGet, "/api/export/csv?policy_id="+rawID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	cd := rec.Header().Get("Content-Disposition")
	if strings.Contains(cd, `/`) || strings.Contains(cd, `*`) {
		t.Fatalf("filename not sanitized: %q", cd)
	}
}

func testAuthKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		t.Fatal(err)
	}
	return k
}

func TestExportAuth_unauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h, err := auth.NewHandler(auth.Config{ClientID: "x"}, testAuthKey(t), auth.NewMemorySessionStore())
	if err != nil {
		t.Fatal(err)
	}
	m := &mockRequirementStore{}
	m.matrixHook = func(ctx context.Context, f store.RequirementFilter) ([]store.RequirementRow, error) {
		return nil, nil
	}
	st := testStoresWithRequirements(m)
	st.Evidence = stubEvidenceStore{}
	st.Policies = stubPolicyStore{}
	st.AuditLogs = stubAuditLogStore{}

	mux := http.NewServeMux()
	store.Register(mux, st)
	wrapped := h.Middleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/export/csv?policy_id=p1", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d body %s", rec.Code, rec.Body.String())
	}
}

func TestSelectExportAuditLog_explicitID(t *testing.T) {
	t.Parallel()
	als := &stubAuditGetStore{log: &store.AuditLog{PolicyID: "p1", Summary: `{"gaps":1}`}}
	log, err := store.SelectExportAuditLog(context.Background(), als, "p1", "aid", time.Time{}, time.Time{})
	if err != nil || log == nil || log.Summary == "" {
		t.Fatalf("log=%v err=%v", log, err)
	}
	_, err = store.SelectExportAuditLog(context.Background(), als, "other", "aid", time.Time{}, time.Time{})
	if !errors.Is(err, store.ErrExportAuditPolicyMismatch) {
		t.Fatalf("want mismatch got %v", err)
	}
}

type stubAuditGetStore struct {
	log *store.AuditLog
}

func (s *stubAuditGetStore) InsertAuditLog(ctx context.Context, a store.AuditLog) error {
	return nil
}

func (s *stubAuditGetStore) ListAuditLogs(ctx context.Context, policyID string, start, end time.Time, limit int) ([]store.AuditLog, error) {
	return nil, nil
}

func (s *stubAuditGetStore) GetAuditLog(ctx context.Context, auditID string) (*store.AuditLog, error) {
	return s.log, nil
}
