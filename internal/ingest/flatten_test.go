// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"testing"
	"time"

	gemara "github.com/gemaraproj/go-gemara"
)

func TestParseDatetime_ValidISO8601(t *testing.T) {
	dt := gemara.Datetime("2025-06-15T10:30:00Z")
	got, err := parseDatetime(dt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil time")
	}
	want := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseDatetime_EmptyString(t *testing.T) {
	got, err := parseDatetime(gemara.Datetime(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for empty datetime, got %v", got)
	}
}

func TestParseDatetime_InvalidFormat(t *testing.T) {
	_, err := parseDatetime(gemara.Datetime("not-a-date"))
	if err == nil {
		t.Fatal("expected error for invalid datetime format")
	}
}

func TestParseDatetime_WithOffset(t *testing.T) {
	dt := gemara.Datetime("2025-01-01T00:00:00+05:00")
	got, err := parseDatetime(dt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil time")
	}
	if got.UTC().Hour() != 19 {
		t.Errorf("expected UTC hour 19, got %d", got.UTC().Hour())
	}
}
