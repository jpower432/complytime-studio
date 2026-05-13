// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"testing"
)

func TestParseMappingEntries_ValidMapping(t *testing.T) {
	content := `
title: Test Mapping
metadata:
  type: MappingDocument
  id: test-map
  gemara-version: 1.0.0
  version: 1.0.0
  description: Test
  author:
    id: test
    name: Test
    type: Software Assisted
source-reference:
  entry-type: Control
  reference-id: src
target-reference:
  entry-type: AssessmentRequirement
  reference-id: tgt
mappings:
  - id: bp1-map
    source: BP-1
    relationship: supports
    targets:
      - entry-id: CC8.1
        strength: 9
        confidence-level: High
      - entry-id: CC6.1
        strength: 6
        confidence-level: Medium
  - id: bp2-map
    source: BP-2
    relationship: supports
    targets:
      - entry-id: CC8.1
        strength: 8
        confidence-level: High
`
	entries, err := ParseMappingEntries(content, "map-1", "soc2-2024", "ccc-v4", "SOC 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	e := entries[0]
	if e.GuidelineID != "BP-1" {
		t.Errorf("expected guideline_id 'BP-1' (m.Source), got %q", e.GuidelineID)
	}
	if e.ControlID != "CC8.1" {
		t.Errorf("expected control_id 'CC8.1' (t.EntryId), got %q", e.ControlID)
	}
	if e.Reference != "CC8.1" || e.Strength != 9 || e.Confidence != "High" {
		t.Errorf("unexpected first entry: %+v", e)
	}
	if e.MappingID != "map-1" || e.SourceCatalogID != "soc2-2024" || e.TargetCatalogID != "ccc-v4" || e.Framework != "SOC 2" {
		t.Errorf("unexpected metadata on entry: %+v", e)
	}
	if e.RequirementID != "bp1-map" {
		t.Errorf("expected requirement_id 'bp1-map', got %q", e.RequirementID)
	}

	e2 := entries[1]
	if e2.GuidelineID != "BP-1" || e2.ControlID != "CC6.1" || e2.Reference != "CC6.1" || e2.Strength != 6 {
		t.Errorf("unexpected second entry: %+v", e2)
	}

	e3 := entries[2]
	if e3.GuidelineID != "BP-2" || e3.ControlID != "CC8.1" || e3.RequirementID != "bp2-map" {
		t.Errorf("unexpected third entry: %+v", e3)
	}
}

func TestParseMappingEntries_MissingOptionalFields(t *testing.T) {
	content := `
title: Minimal
metadata:
  type: MappingDocument
  id: min
  gemara-version: 1.0.0
  description: Minimal mapping
  author:
    id: test
    name: Test
    type: Software Assisted
source-reference:
  entry-type: Control
  reference-id: src
target-reference:
  entry-type: AssessmentRequirement
  reference-id: tgt
mappings:
  - id: m1
    source: CTL-1
    relationship: relates-to
    targets:
      - entry-id: A.8.9
`
	entries, err := ParseMappingEntries(content, "map-2", "iso27001", "target-cat-1", "ISO 27001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.GuidelineID != "CTL-1" {
		t.Errorf("expected guideline_id 'CTL-1' (m.Source), got %q", e.GuidelineID)
	}
	if e.ControlID != "A.8.9" {
		t.Errorf("expected control_id 'A.8.9' (t.EntryId), got %q", e.ControlID)
	}
	if e.Strength != 0 {
		t.Errorf("expected strength 0, got %d", e.Strength)
	}
	if e.Confidence != "Undetermined" {
		t.Errorf("expected 'Undetermined' confidence, got %q", e.Confidence)
	}
}

func TestParseMappingEntries_InvalidYAML(t *testing.T) {
	_, err := ParseMappingEntries("not: [valid: yaml", "m", "s", "t", "f")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseMappingEntries_EmptyMappings(t *testing.T) {
	content := `
title: Empty
metadata:
  type: MappingDocument
  id: empty
  gemara-version: 1.0.0
  description: No mappings
  author:
    id: test
    name: Test
    type: Software Assisted
source-reference:
  entry-type: Control
  reference-id: src
target-reference:
  entry-type: AssessmentRequirement
  reference-id: tgt
mappings: []
`
	entries, err := ParseMappingEntries(content, "m", "s", "t", "f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseMappingEntries_NoMappingsKey(t *testing.T) {
	content := `
title: Some document
metadata:
  type: MappingDocument
  id: nomaps
  gemara-version: 1.0.0
  description: No mappings key
  author:
    id: test
    name: Test
    type: Software Assisted
source-reference:
  entry-type: Control
  reference-id: src
target-reference:
  entry-type: AssessmentRequirement
  reference-id: tgt
`
	entries, err := ParseMappingEntries(content, "m", "s", "t", "f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}
