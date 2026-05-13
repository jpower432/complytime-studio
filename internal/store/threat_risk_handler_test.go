// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gemarapkg "github.com/complytime/complytime-studio/internal/gemara"
	"github.com/labstack/echo/v4"
)

type fakeThreatStore struct {
	threats        []gemarapkg.ThreatRow
	controlThreats []gemarapkg.ControlThreatRow
	lastLimit      int
}

func (f *fakeThreatStore) InsertThreats(_ context.Context, _ []gemarapkg.ThreatRow) error { return nil }
func (f *fakeThreatStore) CountThreats(_ context.Context, _ string) (int, error)          { return 0, nil }
func (f *fakeThreatStore) InsertControlThreats(_ context.Context, _ []gemarapkg.ControlThreatRow) error {
	return nil
}

func (f *fakeThreatStore) QueryThreats(_ context.Context, _, _ string, limit int) ([]gemarapkg.ThreatRow, error) {
	f.lastLimit = limit
	return f.threats, nil
}

func (f *fakeThreatStore) QueryControlThreats(_ context.Context, _, _ string, limit int) ([]gemarapkg.ControlThreatRow, error) {
	f.lastLimit = limit
	return f.controlThreats, nil
}

type fakeRiskStore struct {
	risks       []gemarapkg.RiskRow
	riskThreats []gemarapkg.RiskThreatRow
	lastLimit   int
}

func (f *fakeRiskStore) InsertRisks(_ context.Context, _ []gemarapkg.RiskRow) error { return nil }
func (f *fakeRiskStore) InsertRiskThreats(_ context.Context, _ []gemarapkg.RiskThreatRow) error {
	return nil
}
func (f *fakeRiskStore) CountRisks(_ context.Context, _ string) (int, error) { return 0, nil }
func (f *fakeRiskStore) GetPolicyRiskSeverity(_ context.Context, _ string) ([]RiskSeverityRow, error) {
	return nil, nil
}

func (f *fakeRiskStore) QueryRisks(_ context.Context, _, _ string, limit int) ([]gemarapkg.RiskRow, error) {
	f.lastLimit = limit
	return f.risks, nil
}

func (f *fakeRiskStore) QueryRiskThreats(_ context.Context, _, _ string, limit int) ([]gemarapkg.RiskThreatRow, error) {
	f.lastLimit = limit
	return f.riskThreats, nil
}

func TestListThreatsHandler(t *testing.T) {
	t.Parallel()
	seeded := []gemarapkg.ThreatRow{
		{CatalogID: "tc-1", ThreatID: "t-1", Title: "Threat One", PolicyID: "pol-1"},
		{CatalogID: "tc-1", ThreatID: "t-2", Title: "Threat Two", PolicyID: "pol-1"},
	}

	tests := []struct {
		name       string
		query      string
		rows       []gemarapkg.ThreatRow
		wantStatus int
		wantLen    int
	}{
		{
			name:       "returns seeded threats",
			query:      "/api/threats?catalog_id=tc-1",
			rows:       seeded,
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "empty result returns JSON array",
			query:      "/api/threats?catalog_id=missing",
			rows:       nil,
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeThreatStore{threats: tt.rows}
			e := echo.New()
			g := e.Group("/api")
			g.GET("/threats", listThreatsHandler(fake))

			req := httptest.NewRequest(http.MethodGet, tt.query, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %q", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var got []gemarapkg.ThreatRow
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestListThreatsHandler_DefaultLimit(t *testing.T) {
	t.Parallel()
	fake := &fakeThreatStore{}
	e := echo.New()
	g := e.Group("/api")
	g.GET("/threats", listThreatsHandler(fake))

	req := httptest.NewRequest(http.MethodGet, "/api/threats", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastLimit != 100 {
		t.Fatalf("limit = %d, want default 100", fake.lastLimit)
	}
}

func TestListControlThreatsHandler(t *testing.T) {
	t.Parallel()
	seeded := []gemarapkg.ControlThreatRow{
		{CatalogID: "cc-1", ControlID: "ctrl-1", ThreatReferenceID: "tc-1", ThreatEntryID: "t-1"},
	}

	fake := &fakeThreatStore{controlThreats: seeded}
	e := echo.New()
	g := e.Group("/api")
	g.GET("/control-threats", listControlThreatsHandler(fake))

	req := httptest.NewRequest(http.MethodGet, "/api/control-threats?catalog_id=cc-1&control_id=ctrl-1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %q", rec.Code, rec.Body.String())
	}
	var got []gemarapkg.ControlThreatRow
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ControlID != "ctrl-1" {
		t.Fatalf("control_id = %q, want ctrl-1", got[0].ControlID)
	}
}

func TestListRisksHandler(t *testing.T) {
	t.Parallel()
	seeded := []gemarapkg.RiskRow{
		{CatalogID: "rc-1", RiskID: "r-1", Title: "Risk One", Severity: "High", PolicyID: "pol-1"},
	}

	tests := []struct {
		name       string
		query      string
		rows       []gemarapkg.RiskRow
		wantStatus int
		wantLen    int
	}{
		{
			name:       "returns seeded risks",
			query:      "/api/risks?catalog_id=rc-1",
			rows:       seeded,
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "empty result returns JSON array",
			query:      "/api/risks",
			rows:       nil,
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeRiskStore{risks: tt.rows}
			e := echo.New()
			g := e.Group("/api")
			g.GET("/risks", listRisksHandler(fake))

			req := httptest.NewRequest(http.MethodGet, tt.query, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %q", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var got []gemarapkg.RiskRow
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestListRisksHandler_DefaultLimit(t *testing.T) {
	t.Parallel()
	fake := &fakeRiskStore{}
	e := echo.New()
	g := e.Group("/api")
	g.GET("/risks", listRisksHandler(fake))

	req := httptest.NewRequest(http.MethodGet, "/api/risks", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastLimit != 100 {
		t.Fatalf("limit = %d, want default 100", fake.lastLimit)
	}
}

func TestListRiskThreatsHandler(t *testing.T) {
	t.Parallel()
	seeded := []gemarapkg.RiskThreatRow{
		{CatalogID: "rc-1", RiskID: "r-1", ThreatReferenceID: "tc-1", ThreatEntryID: "t-1"},
	}

	fake := &fakeRiskStore{riskThreats: seeded}
	e := echo.New()
	g := e.Group("/api")
	g.GET("/risk-threats", listRiskThreatsHandler(fake))

	req := httptest.NewRequest(http.MethodGet, "/api/risk-threats?catalog_id=rc-1&risk_id=r-1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %q", rec.Code, rec.Body.String())
	}
	var got []gemarapkg.RiskThreatRow
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].RiskID != "r-1" {
		t.Fatalf("risk_id = %q, want r-1", got[0].RiskID)
	}
}

func TestListThreatsHandler_CustomLimit(t *testing.T) {
	t.Parallel()
	fake := &fakeThreatStore{}
	e := echo.New()
	g := e.Group("/api")
	g.GET("/threats", listThreatsHandler(fake))

	req := httptest.NewRequest(http.MethodGet, "/api/threats?limit=50", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastLimit != 50 {
		t.Fatalf("limit = %d, want 50", fake.lastLimit)
	}
}

func TestListThreatsHandler_LimitClamped(t *testing.T) {
	t.Parallel()
	fake := &fakeThreatStore{}
	e := echo.New()
	g := e.Group("/api")
	g.GET("/threats", listThreatsHandler(fake))

	req := httptest.NewRequest(http.MethodGet, "/api/threats?limit=9999", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastLimit != 1000 {
		t.Fatalf("limit = %d, want clamped 1000", fake.lastLimit)
	}
}
