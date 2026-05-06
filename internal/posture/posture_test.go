// SPDX-License-Identifier: Apache-2.0

package posture

import "testing"

func Test_classifyHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		score, green, red int
		want              string
	}{
		{score: 100, green: 90, red: 50, want: "green"},
		{score: 90, green: 90, red: 50, want: "green"},
		{score: 89, green: 90, red: 50, want: "yellow"},
		{score: 51, green: 90, red: 50, want: "yellow"},
		{score: 50, green: 90, red: 50, want: "red"},
		{score: 0, green: 90, red: 50, want: "red"},
	}
	for _, tt := range tests {
		got := classifyHealth(tt.score, tt.green, tt.red)
		if got != tt.want {
			t.Fatalf("classifyHealth(%d,%d,%d)=%q want %q", tt.score, tt.green, tt.red, got, tt.want)
		}
	}
}

func Test_rollupTargetResult(t *testing.T) {
	t.Parallel()
	if g := rollupTargetResult(1, 0, 0, 0); g != "pass" {
		t.Fatalf("got %q", g)
	}
	if g := rollupTargetResult(1, 1, 0, 0); g != "fail" {
		t.Fatalf("got %q", g)
	}
	if g := rollupTargetResult(1, 0, 1, 0); g != "error" {
		t.Fatalf("got %q", g)
	}
	if g := rollupTargetResult(1, 0, 0, 1); g != "unknown" {
		t.Fatalf("got %q", g)
	}
	if g := rollupTargetResult(0, 0, 0, 0); g != "unknown" {
		t.Fatalf("got %q", g)
	}
}
