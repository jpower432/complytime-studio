// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"testing"
)

func TestParseControlCatalog_Valid(t *testing.T) {
	yaml := `
title: Test Controls
metadata:
  id: cc-test
  type: ControlCatalog
  gemara-version: "1.0.0"
  description: Test catalog
  author:
    id: test
    name: Test
    type: Human
  applicability-groups:
    - id: all
      title: All
      description: All environments
groups:
  - id: g1
    title: Group 1
    description: First group
controls:
  - id: C-1
    title: Control One
    objective: First objective
    group: g1
    state: Active
    assessment-requirements:
      - id: AR-1.1
        text: Must do X
        applicability: [all]
        state: Active
    threats:
      - reference-id: tc-ref
        entries:
          - reference-id: T-1
  - id: C-2
    title: Control Two
    objective: Second objective
    group: g1
    state: Active
    assessment-requirements:
      - id: AR-2.1
        text: Must do Y
        applicability: [all]
        state: Active
`
	controls, reqs, threats, err := ParseControlCatalog(context.Background(), yaml, "", "pol-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(controls) != 2 {
		t.Fatalf("expected 2 controls, got %d", len(controls))
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 assessment requirements, got %d", len(reqs))
	}
	if len(threats) != 1 {
		t.Fatalf("expected 1 threat link, got %d", len(threats))
	}
	if controls[0].CatalogID != "cc-test" || controls[0].PolicyID != "pol-1" {
		t.Errorf("control[0]: %+v", controls[0])
	}
	if threats[0].ThreatReferenceID != "tc-ref" || threats[0].ThreatEntryID != "T-1" {
		t.Errorf("threat link: %+v", threats[0])
	}
}

func TestParseControlCatalog_EmptyControls(t *testing.T) {
	yaml := `
title: Empty Controls
metadata:
  id: cc-empty
  type: ControlCatalog
  gemara-version: "1.0.0"
  description: Empty
  author:
    id: test
    name: Test
    type: Human
controls: []
`
	controls, reqs, threats, err := ParseControlCatalog(context.Background(), yaml, "", "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(controls) != 0 || len(reqs) != 0 || len(threats) != 0 {
		t.Fatalf("expected empty slices, got %d %d %d", len(controls), len(reqs), len(threats))
	}
}

func TestParseControlCatalog_InvalidYAML(t *testing.T) {
	_, _, _, err := ParseControlCatalog(context.Background(), "not: [valid: yaml: {{", "", "")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
