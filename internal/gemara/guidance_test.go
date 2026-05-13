// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"testing"
)

func TestParseGuidanceCatalog_Valid(t *testing.T) {
	content := `
title: SOC 2 Trust Services Criteria
metadata:
  type: GuidanceCatalog
  id: soc2-2024
  gemara-version: 1.0.0
  version: 2024.1
  description: SOC 2 criteria
guidelines:
  - id: CC6.1
    title: Logical Access
    objective: Restrict logical access
    group: CC6
  - id: CC6.2
    title: Authentication
    objective: Authenticate users
    group: CC6
  - id: CC7.1
    title: System Operations
    group: CC7
`
	rows, err := ParseGuidanceCatalog(context.Background(), content, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	r := rows[0]
	if r.CatalogID != "soc2-2024" {
		t.Errorf("expected catalog_id 'soc2-2024', got %q", r.CatalogID)
	}
	if r.GuidelineID != "CC6.1" || r.Title != "Logical Access" {
		t.Errorf("unexpected first row: %+v", r)
	}
	if r.Objective != "Restrict logical access" {
		t.Errorf("expected objective, got %q", r.Objective)
	}
	if r.GroupID != "CC6" {
		t.Errorf("expected group CC6, got %q", r.GroupID)
	}
	if r.State != "Active" {
		t.Errorf("expected state Active, got %q", r.State)
	}
}

func TestParseGuidanceCatalog_ExplicitCatalogID(t *testing.T) {
	content := `
title: Test
metadata:
  type: GuidanceCatalog
  id: original-id
  gemara-version: 1.0.0
  description: Test
guidelines:
  - id: G1
    title: Guideline 1
    group: grp
`
	rows, err := ParseGuidanceCatalog(context.Background(), content, "override-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].CatalogID != "override-id" {
		t.Errorf("expected override-id, got %q", rows[0].CatalogID)
	}
}

func TestParseGuidanceCatalog_NoID(t *testing.T) {
	content := `
title: No ID
metadata:
  type: GuidanceCatalog
  gemara-version: 1.0.0
  description: Missing ID
guidelines:
  - id: G1
    title: G1
    group: grp
`
	_, err := ParseGuidanceCatalog(context.Background(), content, "")
	if err == nil {
		t.Fatal("expected error for missing catalog id")
	}
}

func TestParseGuidanceCatalog_SkipsEmptyID(t *testing.T) {
	content := `
title: Mixed
metadata:
  type: GuidanceCatalog
  id: mixed
  gemara-version: 1.0.0
  description: Mixed
guidelines:
  - id: G1
    title: Valid
    group: grp
  - id: ""
    title: Invalid
    group: grp
  - id: G2
    title: Also Valid
    group: grp
`
	rows, err := ParseGuidanceCatalog(context.Background(), content, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (skipping empty id), got %d", len(rows))
	}
	if rows[0].GuidelineID != "G1" || rows[1].GuidelineID != "G2" {
		t.Errorf("expected G1 and G2, got %q and %q", rows[0].GuidelineID, rows[1].GuidelineID)
	}
}

func TestParseGuidanceCatalog_EmptyGuidelines(t *testing.T) {
	content := `
title: Empty
metadata:
  type: GuidanceCatalog
  id: empty
  gemara-version: 1.0.0
  description: No guidelines
guidelines: []
`
	rows, err := ParseGuidanceCatalog(context.Background(), content, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestParseGuidanceCatalog_InvalidYAML(t *testing.T) {
	_, err := ParseGuidanceCatalog(context.Background(), "not: [valid yaml", "")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
