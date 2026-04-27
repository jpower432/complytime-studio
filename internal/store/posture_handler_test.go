// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/complytime/complytime-studio/internal/auth"
)

type fakePostureStore struct {
	rows      []PostureRow
	err       error
	lastStart time.Time
	lastEnd   time.Time
}

func (f *fakePostureStore) ListPosture(_ context.Context, start, end time.Time) ([]PostureRow, error) {
	f.lastStart = start
	f.lastEnd = end
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func (f *fakePostureStore) QueryPolicyPosture(_ context.Context, _ string) (uint64, uint64, uint64, error) {
	return 0, 0, 0, nil
}

func testAuthSecretKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func TestListPostureHandler(t *testing.T) {
	t.Parallel()

	seeded := []PostureRow{
		{
			PolicyID:   "pol-1",
			Title:      "Test Policy",
			Version:    "1.0",
			TotalRows:  10,
			PassedRows: 7,
			FailedRows: 2,
			OtherRows:  1,
			LatestAt:   "2026-01-15T12:00:00Z",
		},
	}

	tests := []struct {
		name       string
		store      PostureStore
		wantStatus int
		wantLen    int
		wantFirst  *PostureRow
	}{
		{
			name:       "returns seeded aggregates",
			store:      &fakePostureStore{rows: seeded},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantFirst:  &seeded[0],
		},
		{
			name:       "empty store returns JSON array",
			store:      &fakePostureStore{rows: nil},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "store error returns 500",
			store:      &fakePostureStore{err: io.ErrUnexpectedEOF},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/posture", listPostureHandler(tt.store))
			req := httptest.NewRequest(http.MethodGet, "/api/posture", nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %q", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}

			var got []PostureRow
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode JSON: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len(got) = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantFirst != nil {
				if len(got) == 0 {
					t.Fatal("wantFirst set but got empty slice")
				}
				if got[0] != *tt.wantFirst {
					t.Errorf("got[0] = %+v, want %+v", got[0], *tt.wantFirst)
				}
			}
		})
	}
}

func TestListPostureHandler_AuthMiddleware(t *testing.T) {
	t.Parallel()

	seeded := []PostureRow{{PolicyID: "p1", Title: "Policy One", TotalRows: 3}}
	mock := &fakePostureStore{rows: seeded}

	h, err := auth.NewHandler(auth.Config{}, testAuthSecretKey(t), auth.NewMemorySessionStore())
	if err != nil {
		t.Fatal(err)
	}
	const apiToken = "test-posture-api-token"
	h.SetAPIToken(apiToken)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/posture", listPostureHandler(mock))
	stack := h.Middleware(mux)

	tests := []struct {
		name       string
		setupReq   func(*http.Request)
		wantStatus int
		decodeBody bool
		wantPolicy string
	}{
		{
			name: "unauthenticated returns 401",
			setupReq: func(_ *http.Request) {
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "bearer API token returns posture JSON",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+apiToken)
			},
			wantStatus: http.StatusOK,
			decodeBody: true,
			wantPolicy: "p1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/api/posture", nil)
			tt.setupReq(req)
			rec := httptest.NewRecorder()
			stack.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %q", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if !tt.decodeBody {
				return
			}
			var rows []PostureRow
			if err := json.NewDecoder(rec.Body).Decode(&rows); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(rows) != 1 || rows[0].PolicyID != tt.wantPolicy {
				t.Fatalf("body = %+v, want one row policy_id %q", rows, tt.wantPolicy)
			}
		})
	}
}

func TestListPostureHandler_AuthMiddleware_wrongBearer(t *testing.T) {
	t.Parallel()

	mock := &fakePostureStore{rows: []PostureRow{{PolicyID: "x"}}}
	h, err := auth.NewHandler(auth.Config{}, testAuthSecretKey(t), auth.NewMemorySessionStore())
	if err != nil {
		t.Fatal(err)
	}
	h.SetAPIToken("correct-token")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/posture", listPostureHandler(mock))
	stack := h.Middleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/posture", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	stack.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestListPostureHandler_TimeFilter(t *testing.T) {
	t.Parallel()

	seeded := []PostureRow{{PolicyID: "pol-1", Title: "P", TotalRows: 5, PassedRows: 3}}

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantStart  bool
		wantEnd    bool
		checkEnd   func(*testing.T, time.Time)
	}{
		{
			name:       "no params passes zero times",
			query:      "/api/posture",
			wantStatus: http.StatusOK,
		},
		{
			name:       "date-only start and end are parsed",
			query:      "/api/posture?start=2026-04-01&end=2026-04-26",
			wantStatus: http.StatusOK,
			wantStart:  true,
			wantEnd:    true,
			checkEnd: func(t *testing.T, end time.Time) {
				t.Helper()
				expected := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
				if !end.Equal(expected) {
					t.Errorf("end = %v, want end-of-day %v", end, expected)
				}
			},
		},
		{
			name:       "RFC 3339 start is parsed",
			query:      "/api/posture?start=2026-04-01T00:00:00Z",
			wantStatus: http.StatusOK,
			wantStart:  true,
		},
		{
			name:       "invalid start returns 400",
			query:      "/api/posture?start=not-a-date",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid end returns 400",
			query:      "/api/posture?end=xyz",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := &fakePostureStore{rows: seeded}
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/posture", listPostureHandler(mock))

			req := httptest.NewRequest(http.MethodGet, tt.query, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %q", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			if tt.wantStart && mock.lastStart.IsZero() {
				t.Error("expected non-zero start time")
			}
			if tt.wantEnd && mock.lastEnd.IsZero() {
				t.Error("expected non-zero end time")
			}
			if !tt.wantStart && !mock.lastStart.IsZero() {
				t.Error("expected zero start time")
			}
			if tt.checkEnd != nil {
				tt.checkEnd(t, mock.lastEnd)
			}
		})
	}
}
