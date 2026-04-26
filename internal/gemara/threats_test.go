// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"testing"
)

func TestParseThreatCatalog_Valid(t *testing.T) {
	yaml := `
title: Test Threats
metadata:
  id: tc-test
  type: ThreatCatalog
  gemara-version: "1.0.0"
  description: Test threats
  author:
    id: test
    name: Test
    type: Human
groups:
  - id: g1
    title: Group 1
    description: First group
threats:
  - id: T-1
    title: Threat One
    description: First threat
    group: g1
    capabilities:
      - reference-id: cap-ref
        entries:
          - reference-id: CAP-1
`
	rows, err := ParseThreatCatalog(context.Background(), yaml, "", "pol-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 threat row, got %d", len(rows))
	}
	if rows[0].CatalogID != "tc-test" || rows[0].ThreatID != "T-1" || rows[0].PolicyID != "pol-1" {
		t.Errorf("row: %+v", rows[0])
	}
	if rows[0].Title != "Threat One" || rows[0].Description != "First threat" || rows[0].GroupID != "g1" {
		t.Errorf("row fields: %+v", rows[0])
	}
}

func TestParseThreatCatalog_EmptyThreats(t *testing.T) {
	yaml := `
title: No Threats
metadata:
  id: tc-empty
  type: ThreatCatalog
  gemara-version: "1.0.0"
  description: Empty
  author:
    id: test
    name: Test
    type: Human
threats: []
`
	rows, err := ParseThreatCatalog(context.Background(), yaml, "", "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestParseThreatCatalog_InvalidYAML(t *testing.T) {
	_, err := ParseThreatCatalog(context.Background(), "not: [valid: yaml: {{", "", "")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
