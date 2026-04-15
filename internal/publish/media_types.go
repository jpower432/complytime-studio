// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"fmt"

	gemara "github.com/gemaraproj/go-gemara"
)

const (
	MediaTypeCapabilityCatalog = "application/vnd.gemara.capability-catalog.layer.v1+yaml"
	MediaTypeThreatCatalog     = "application/vnd.gemara.threat-catalog.layer.v1+yaml"
	MediaTypeControlCatalog    = "application/vnd.gemara.control-catalog.layer.v1+yaml"
	MediaTypeGuidanceCatalog   = "application/vnd.gemara.guidance-catalog.layer.v1+yaml"
	MediaTypeVectorCatalog     = "application/vnd.gemara.vector-catalog.layer.v1+yaml"
	MediaTypePrincipleCatalog  = "application/vnd.gemara.principle-catalog.layer.v1+yaml"
	MediaTypeMappingDocument   = "application/vnd.gemara.mapping-document.layer.v1+yaml"
	MediaTypePolicy            = "application/vnd.gemara.policy.layer.v1+yaml"
	MediaTypeRiskCatalog       = "application/vnd.gemara.risk-catalog.layer.v1+yaml"
	MediaTypeAuditLog          = "application/vnd.gemara.audit-log.layer.v1+yaml"
	MediaTypeEvaluationLog     = "application/vnd.gemara.evaluation-log.layer.v1+yaml"
	MediaTypeEnforcementLog    = "application/vnd.gemara.enforcement-log.layer.v1+yaml"
)

var artifactTypeToMediaType = map[gemara.ArtifactType]string{
	gemara.CapabilityCatalogArtifact: MediaTypeCapabilityCatalog,
	gemara.ThreatCatalogArtifact:     MediaTypeThreatCatalog,
	gemara.ControlCatalogArtifact:    MediaTypeControlCatalog,
	gemara.GuidanceCatalogArtifact:   MediaTypeGuidanceCatalog,
	gemara.VectorCatalogArtifact:     MediaTypeVectorCatalog,
	gemara.PrincipleCatalogArtifact:  MediaTypePrincipleCatalog,
	gemara.MappingDocumentArtifact:   MediaTypeMappingDocument,
	gemara.PolicyArtifact:            MediaTypePolicy,
	gemara.RiskCatalogArtifact:       MediaTypeRiskCatalog,
	gemara.AuditLogArtifact:          MediaTypeAuditLog,
	gemara.EvaluationLogArtifact:     MediaTypeEvaluationLog,
	gemara.EnforcementLogArtifact:    MediaTypeEnforcementLog,
}

// MediaTypeForArtifact returns the OCI media type for a Gemara artifact type.
func MediaTypeForArtifact(artifactType gemara.ArtifactType) (string, error) {
	mt, ok := artifactTypeToMediaType[artifactType]
	if !ok {
		return "", fmt.Errorf("unknown Gemara artifact type: %q", artifactType)
	}
	return mt, nil
}

// KnownArtifactTypes returns all artifact type names that have a registered media type.
func KnownArtifactTypes() []string {
	types := make([]string, 0, len(artifactTypeToMediaType))
	for t := range artifactTypeToMediaType {
		types = append(types, t.String())
	}
	return types
}
