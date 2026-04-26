// SPDX-License-Identifier: Apache-2.0

package store

import (
	"strings"
	"testing"
	"time"
)

func TestParseCSVEvidence_TableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		csv          string
		wantRows     int
		wantErrSubs  []string
		wantWarnSubs []string
	}{
		{
			name: "missing required header",
			csv:  "policy_id,eval_result\np1,Passed\n",
			wantErrSubs: []string{
				"missing required column: collected_at",
			},
		},
		{
			name: "bad timestamp row",
			csv: strings.Join([]string{
				"policy_id,eval_result,collected_at,target_id,control_id,rule_id",
				"p1,Passed,not-a-date,t1,c1,r1",
				"p1,Passed,2026-04-25T12:00:00Z,t1,c1,r1",
			}, "\n"),
			wantRows:    1,
			wantErrSubs: []string{"invalid collected_at timestamp"},
		},
		{
			name: "good partial rows only required headers plus ids",
			csv: strings.Join([]string{
				"policy_id,eval_result,collected_at,target_id,control_id,rule_id",
				"p1,Passed,2026-04-25T12:00:00Z,t1,c1,r1",
				"p2,Failed,2026-04-25T13:00:00Z,t2,c2,r2",
			}, "\n"),
			wantRows: 2,
		},
		{
			name: "warn when requirement_id column absent",
			csv: strings.Join([]string{
				"policy_id,eval_result,collected_at,target_id,control_id,rule_id",
				"p1,Unknown,2026-04-25T12:00:00Z,t1,c1,r1",
			}, "\n"),
			wantRows:     1,
			wantWarnSubs: []string{"recommended column 'requirement_id' not in header"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rows, errs, warns := parseCSVEvidence(strings.NewReader(tt.csv))
			if len(rows) != tt.wantRows {
				t.Fatalf("rows: got %d want %d (errs=%v warns=%v)", len(rows), tt.wantRows, errs, warns)
			}
			joinedErr := strings.Join(errs, " ")
			for _, sub := range tt.wantErrSubs {
				if !strings.Contains(joinedErr, sub) {
					t.Fatalf("errors %v missing %q", errs, sub)
				}
			}
			joinedWarn := strings.Join(warns, " ")
			for _, sub := range tt.wantWarnSubs {
				if !strings.Contains(joinedWarn, sub) {
					t.Fatalf("warnings %v missing %q", warns, sub)
				}
			}
			if tt.wantRows > 0 {
				if rows[0].PolicyID == "" || rows[0].CollectedAt.IsZero() {
					t.Fatalf("first row incomplete: %+v", rows[0])
				}
				if tt.name == "good partial rows only required headers plus ids" {
					if rows[0].EvalResult != "Passed" || rows[1].EvalResult != "Failed" {
						t.Fatalf("eval results: %+v", rows)
					}
					if !rows[0].CollectedAt.Equal(time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)) {
						t.Fatalf("time %v", rows[0].CollectedAt)
					}
				}
			}
		})
	}
}
