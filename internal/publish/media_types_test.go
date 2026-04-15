// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"testing"

	gemara "github.com/gemaraproj/go-gemara"
)

func TestMediaTypeForArtifact_KnownTypes(t *testing.T) {
	cases := []struct {
		artifactType  gemara.ArtifactType
		wantMediaType string
	}{
		{gemara.ThreatCatalogArtifact, MediaTypeThreatCatalog},
		{gemara.ControlCatalogArtifact, MediaTypeControlCatalog},
		{gemara.PolicyArtifact, MediaTypePolicy},
		{gemara.AuditLogArtifact, MediaTypeAuditLog},
		{gemara.MappingDocumentArtifact, MediaTypeMappingDocument},
		{gemara.GuidanceCatalogArtifact, MediaTypeGuidanceCatalog},
		{gemara.RiskCatalogArtifact, MediaTypeRiskCatalog},
		{gemara.CapabilityCatalogArtifact, MediaTypeCapabilityCatalog},
		{gemara.VectorCatalogArtifact, MediaTypeVectorCatalog},
		{gemara.PrincipleCatalogArtifact, MediaTypePrincipleCatalog},
		{gemara.EvaluationLogArtifact, MediaTypeEvaluationLog},
		{gemara.EnforcementLogArtifact, MediaTypeEnforcementLog},
	}

	for _, tc := range cases {
		t.Run(tc.artifactType.String(), func(t *testing.T) {
			got, err := MediaTypeForArtifact(tc.artifactType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantMediaType {
				t.Errorf("got %q, want %q", got, tc.wantMediaType)
			}
		})
	}
}

func TestMediaTypeForArtifact_UnknownType(t *testing.T) {
	_, err := MediaTypeForArtifact(gemara.InvalidArtifact)
	if err == nil {
		t.Fatal("expected error for unknown artifact type")
	}
}

func TestKnownArtifactTypes(t *testing.T) {
	types := KnownArtifactTypes()
	if len(types) != len(artifactTypeToMediaType) {
		t.Errorf("got %d types, want %d", len(types), len(artifactTypeToMediaType))
	}
}
