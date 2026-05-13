// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"testing"

	sdk "github.com/gemaraproj/go-gemara"
)

func TestResolveGuidanceRefsFromPolicy_ReturnsMatchedIDs(t *testing.T) {
	pol := sdk.Policy{
		Metadata: sdk.Metadata{
			MappingReferences: []sdk.MappingReference{
				{Id: "soc2-2024"},
				{Id: "nist-csf-2.0"},
			},
		},
		Imports: sdk.Imports{
			Guidance: []sdk.GuidanceImport{
				{ReferenceId: "soc2-2024"},
				{ReferenceId: "nist-csf-2.0"},
			},
		},
	}
	ids := ResolveGuidanceRefsFromPolicy(pol)
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
	if ids[0] != "soc2-2024" || ids[1] != "nist-csf-2.0" {
		t.Errorf("unexpected ids: %v", ids)
	}
}

func TestResolveGuidanceRefsFromPolicy_SkipsUnmatchedRefs(t *testing.T) {
	pol := sdk.Policy{
		Metadata: sdk.Metadata{
			MappingReferences: []sdk.MappingReference{
				{Id: "soc2-2024"},
			},
		},
		Imports: sdk.Imports{
			Guidance: []sdk.GuidanceImport{
				{ReferenceId: "soc2-2024"},
				{ReferenceId: "nonexistent-ref"},
			},
		},
	}
	ids := ResolveGuidanceRefsFromPolicy(pol)
	if len(ids) != 1 {
		t.Fatalf("expected 1 id (unmatched skipped), got %d", len(ids))
	}
	if ids[0] != "soc2-2024" {
		t.Errorf("expected soc2-2024, got %q", ids[0])
	}
}

func TestResolveGuidanceRefsFromPolicy_EmptyPolicy(t *testing.T) {
	ids := ResolveGuidanceRefsFromPolicy(sdk.Policy{})
	if ids != nil {
		t.Errorf("expected nil for empty policy, got %v", ids)
	}
}

func TestResolveGuidanceRefs_ValidPolicyYAML(t *testing.T) {
	content := `
metadata:
  type: Policy
  mapping-references:
    - id: soc2-2024
      title: SOC 2
    - id: fedramp-2024
      title: FedRAMP
imports:
  guidance:
    - reference-id: soc2-2024
    - reference-id: fedramp-2024
`
	ids := ResolveGuidanceRefs(content)
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
	if ids[0] != "soc2-2024" || ids[1] != "fedramp-2024" {
		t.Errorf("unexpected ids: %v", ids)
	}
}

func TestResolveGuidanceRefs_NonPolicyType(t *testing.T) {
	content := `
metadata:
  type: ControlCatalog
imports:
  guidance:
    - reference-id: soc2-2024
`
	ids := ResolveGuidanceRefs(content)
	if ids != nil {
		t.Errorf("expected nil for non-Policy type, got %v", ids)
	}
}

func TestResolveGuidanceRefs_NoGuidanceImports(t *testing.T) {
	content := `
metadata:
  type: Policy
  mapping-references:
    - id: soc2-2024
imports:
  catalogs:
    - reference-id: nist-800-53
`
	ids := ResolveGuidanceRefs(content)
	if ids != nil {
		t.Errorf("expected nil when no guidance imports, got %v", ids)
	}
}

func TestResolveGuidanceRefs_InvalidYAML(t *testing.T) {
	ids := ResolveGuidanceRefs("not: [valid yaml")
	if ids != nil {
		t.Errorf("expected nil for invalid YAML, got %v", ids)
	}
}

func TestResolveGuidanceRefs_EmptyContent(t *testing.T) {
	ids := ResolveGuidanceRefs("")
	if ids != nil {
		t.Errorf("expected nil for empty content, got %v", ids)
	}
}
