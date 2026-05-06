// SPDX-License-Identifier: Apache-2.0

package recommend

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestApplicabilityOverlap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		apps     []string
		evidence map[string]struct{}
		want     float64
	}{
		{name: "empty_program_apps", apps: nil, evidence: map[string]struct{}{"a": {}}, want: 0},
		{name: "empty_evidence_tags", apps: []string{"a"}, evidence: nil, want: 0},
		{name: "full_overlap", apps: []string{"x", "y"}, evidence: map[string]struct{}{"x": {}, "y": {}}, want: 1},
		{
			name:     "partial_overlap",
			apps:     []string{"a", "b", "c", "d"},
			evidence: map[string]struct{}{"a": {}, "b": {}},
			want:     0.5,
		},
		{name: "no_overlap", apps: []string{"a"}, evidence: map[string]struct{}{"z": {}}, want: 0},
		{
			name:     "empty_strings_in_apps_no_extra_matches",
			apps:     []string{"", "hit", ""},
			evidence: map[string]struct{}{"hit": {}},
			want:     float64(1) / float64(3),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := applicabilityOverlap(tt.apps, tt.evidence)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestBuildReason(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mapStr    float64
		title     string
		evCount   int
		wantPref  string
		wantPct   int
		wantEvStr string
	}{
		{name: "strong", mapStr: 0.7, title: "P1", evCount: 3, wantPref: "Strong", wantPct: 70, wantEvStr: "3"},
		{name: "strong_above", mapStr: 0.85, title: "T", evCount: 0, wantPref: "Strong", wantPct: 85, wantEvStr: "0"},
		{name: "moderate_low", mapStr: 0.4, title: "M", evCount: 5, wantPref: "Moderate", wantPct: 40, wantEvStr: "5"},
		{name: "moderate_mid", mapStr: 0.55, title: "M2", evCount: 1, wantPref: "Moderate", wantPct: 55, wantEvStr: "1"},
		{name: "light", mapStr: 0.39, title: "L", evCount: 10, wantPref: "Light", wantPct: 39, wantEvStr: "10"},
		{name: "round_half", mapStr: 0.445, title: "R", evCount: 2, wantPref: "Moderate", wantPct: 45, wantEvStr: "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildReason(tt.mapStr, tt.title, tt.evCount)
			if !strings.HasPrefix(got, tt.wantPref) {
				t.Fatalf("prefix %q got %q", tt.wantPref, got)
			}
			if !strings.Contains(got, strconv.Itoa(tt.wantPct)+"%") {
				t.Fatalf("percentage %d%% missing in %q", tt.wantPct, got)
			}
			if !strings.Contains(got, tt.wantEvStr+" evidence items") {
				t.Fatalf("evidence count %q missing in %q", tt.wantEvStr, got)
			}
		})
	}
}

func compositeScore(peakStrength float64, evCount int, appOverlap float64) float64 {
	mapStrength := math.Min(math.Max(peakStrength/strengthScale, 0), 1)
	evFactor := math.Min(float64(evCount)/evidenceNorm, 1)
	return weightMapping*mapStrength + weightEvidence*evFactor + weightAppOverlap*appOverlap
}

func TestScoreCalculation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		peakStrength float64
		evCount      int
		appOverlap   float64
		want         float64
	}{
		{
			name: "mid_values", peakStrength: 50, evCount: 5, appOverlap: 1,
			want: 0.6*0.5 + 0.3*0.5 + 0.1*1,
		},
		{
			name:         "map_clamped_high",
			peakStrength: 150,
			evCount:      0,
			appOverlap:   0.5,
			want:         0.6*1 + 0.3*0 + 0.1*0.5,
		},
		{
			name:         "map_clamped_low",
			peakStrength: -20,
			evCount:      10,
			appOverlap:   0,
			want:         0.6*0 + 0.3*1 + 0,
		},
		{
			name:         "evidence_cap",
			peakStrength: 0,
			evCount:      25,
			appOverlap:   0,
			want:         0.3 * 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compositeScore(tt.peakStrength, tt.evCount, tt.appOverlap)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestRecommendationSorting(t *testing.T) {
	t.Parallel()
	in := []Recommendation{
		{PolicyID: "a", Score: 0.2},
		{PolicyID: "b", Score: 0.9},
		{PolicyID: "c", Score: 0.52},
		{PolicyID: "d", Score: 0.51},
		{PolicyID: "e", Score: 1.0},
		{PolicyID: "f", Score: 0.0},
	}
	out := append([]Recommendation(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	for i := 0; i < len(out)-1; i++ {
		if out[i].Score < out[i+1].Score {
			t.Fatalf("order at %d: %v before %v", i, out[i].Score, out[i+1].Score)
		}
	}
	wantIDs := []string{"e", "b", "c", "d", "a", "f"}
	for i, id := range wantIDs {
		if out[i].PolicyID != id {
			t.Fatalf("index %d got %q want %q", i, out[i].PolicyID, id)
		}
	}

	many := make([]Recommendation, topRecommendN+5)
	for i := range many {
		many[i] = Recommendation{PolicyID: strconv.Itoa(i), Score: float64(i)}
	}
	sort.Slice(many, func(i, j int) bool { return many[i].Score > many[j].Score })
	if len(many) > topRecommendN {
		many = many[:topRecommendN]
	}
	if len(many) != topRecommendN {
		t.Fatalf("len %d want %d", len(many), topRecommendN)
	}
	for i := 0; i < len(many)-1; i++ {
		if many[i].Score < many[i+1].Score {
			t.Fatalf("truncated slice not descending at %d", i)
		}
	}
	if many[0].PolicyID != strconv.Itoa(topRecommendN + 4) {
		t.Fatalf("top score id got %q", many[0].PolicyID)
	}
}
