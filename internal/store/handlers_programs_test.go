// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// --- fakes ---

type fakeJobStore struct {
	jobs            []Job
	listErr         error
	createErr       error
	updateStatusErr error
	lastStatusID    string
	lastProgramID   string
	lastStatus      string
}

func (f *fakeJobStore) ListJobs(_ context.Context, _ string) ([]Job, error) {
	return f.jobs, f.listErr
}
func (f *fakeJobStore) CreateJob(_ context.Context, j Job) (*Job, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	j.ID = "job-1"
	j.CreatedAt = time.Now()
	j.UpdatedAt = time.Now()
	return &j, nil
}
func (f *fakeJobStore) UpdateJobStatus(_ context.Context, id, programID, status string) error {
	f.lastStatusID = id
	f.lastProgramID = programID
	f.lastStatus = status
	return f.updateStatusErr
}

type fakeMemberStore struct {
	members   []ProgramMember
	listErr   error
	upsertErr error
	removeErr error
}

func (f *fakeMemberStore) ListMembers(_ context.Context, _ string) ([]ProgramMember, error) {
	return f.members, f.listErr
}
func (f *fakeMemberStore) UpsertMember(_ context.Context, _ ProgramMember) error {
	return f.upsertErr
}
func (f *fakeMemberStore) RemoveMember(_ context.Context, _, _ string) error {
	return f.removeErr
}

type fakeFindingStore struct {
	findings        []ProgramFinding
	listErr         error
	createErr       error
	updateStatusErr error
}

func (f *fakeFindingStore) ListFindings(_ context.Context, _, _ string) ([]ProgramFinding, error) {
	return f.findings, f.listErr
}
func (f *fakeFindingStore) CreateFinding(_ context.Context, pf ProgramFinding) (*ProgramFinding, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	pf.ID = "find-1"
	pf.CreatedAt = time.Now()
	pf.UpdatedAt = time.Now()
	return &pf, nil
}
func (f *fakeFindingStore) UpdateFindingStatus(_ context.Context, _, _, _ string) error {
	return f.updateStatusErr
}

func newProgramTestServer(s Stores) *echo.Echo {
	srv := echo.New()
	g := srv.Group("/api")
	Register(g, s)
	return srv
}

// --- job status tests ---

func TestUpdateJobStatus_Success(t *testing.T) {
	t.Parallel()
	fake := &fakeJobStore{}
	srv := newProgramTestServer(Stores{Jobs: fake, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodPatch, "/api/programs/prog-1/jobs/job-1/status",
		strings.NewReader(`{"status":"completed"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if fake.lastProgramID != "prog-1" {
		t.Fatalf("program_id not passed: got %q", fake.lastProgramID)
	}
}

func TestUpdateJobStatus_NotFound(t *testing.T) {
	t.Parallel()
	fake := &fakeJobStore{updateStatusErr: ErrJobNotFound}
	srv := newProgramTestServer(Stores{Jobs: fake, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodPatch, "/api/programs/prog-1/jobs/missing/status",
		strings.NewReader(`{"status":"failed"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestUpdateJobStatus_MissingBody(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Jobs: &fakeJobStore{}, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodPatch, "/api/programs/prog-1/jobs/job-1/status",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

// --- member tests ---

func TestListMembers_Success(t *testing.T) {
	t.Parallel()
	fake := &fakeMemberStore{members: []ProgramMember{
		{ProgramID: "p1", UserEmail: "a@co.com", Role: "owner"},
	}}
	srv := newProgramTestServer(Stores{Members: fake, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodGet, "/api/programs/p1/members", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var got []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 member, got %d", len(got))
	}
}

func TestListMembers_Empty(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Members: &fakeMemberStore{}, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodGet, "/api/programs/p1/members", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var got []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty array, got %d", len(got))
	}
}

func TestUpsertMember_Success(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Members: &fakeMemberStore{}, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/programs/p1/members",
		strings.NewReader(`{"user_email":"a@co.com","role":"contributor"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpsertMember_MissingFields(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Members: &fakeMemberStore{}, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/programs/p1/members",
		strings.NewReader(`{"user_email":"a@co.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestRemoveMember_NotFound(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Members: &fakeMemberStore{removeErr: ErrMemberNotFound}, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodDelete, "/api/programs/p1/members/nobody@co.com", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestRemoveMember_InternalError(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Members: &fakeMemberStore{removeErr: errors.New("db down")}, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodDelete, "/api/programs/p1/members/a@co.com", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
}

// --- finding tests ---

func TestListFindings_Success(t *testing.T) {
	t.Parallel()
	fake := &fakeFindingStore{findings: []ProgramFinding{
		{ID: "f1", ProgramID: "p1", PolicyID: "pol", Source: "audit_log", Type: "Finding", Title: "x", Status: "open"},
	}}
	srv := newProgramTestServer(Stores{Findings: fake, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodGet, "/api/programs/p1/findings", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var got []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
}

func TestCreateFinding_Success(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Findings: &fakeFindingStore{}, Programs: &fakeProgramStore{}})
	body := `{"policy_id":"pol","source":"audit_log","type":"Finding","title":"gap"}`
	req := httptest.NewRequest(http.MethodPost, "/api/programs/p1/findings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateFinding_MissingFields(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Findings: &fakeFindingStore{}, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/programs/p1/findings",
		strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestUpdateFindingStatus_NotFound(t *testing.T) {
	t.Parallel()
	fake := &fakeFindingStore{updateStatusErr: ErrFindingNotFound}
	srv := newProgramTestServer(Stores{Findings: fake, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodPatch, "/api/programs/p1/findings/missing/status",
		strings.NewReader(`{"status":"resolved"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestUpdateFindingStatus_Success(t *testing.T) {
	t.Parallel()
	srv := newProgramTestServer(Stores{Findings: &fakeFindingStore{}, Programs: &fakeProgramStore{}})
	req := httptest.NewRequest(http.MethodPatch, "/api/programs/p1/findings/f1/status",
		strings.NewReader(`{"status":"in_progress"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- fake program store for routing setup ---

type fakeProgramStore struct{}

func (f *fakeProgramStore) ListPrograms(_ context.Context) ([]Program, error)   { return nil, nil }
func (f *fakeProgramStore) GetProgram(_ context.Context, _ string) (*Program, error) {
	return nil, ErrProgramNotFound
}
func (f *fakeProgramStore) CreateProgram(_ context.Context, p Program) (*Program, error) {
	return &p, nil
}
func (f *fakeProgramStore) UpdateProgram(_ context.Context, _ Program) error { return nil }
func (f *fakeProgramStore) DeleteProgram(_ context.Context, _ string) error  { return nil }
