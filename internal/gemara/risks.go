// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"fmt"

	sdk "github.com/gemaraproj/go-gemara"
)

// RiskRow represents a single risk parsed from a RiskCatalog.
type RiskRow struct {
	CatalogID   string `json:"catalog_id"`
	RiskID      string `json:"risk_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	GroupID     string `json:"group_id"`
	Impact      string `json:"impact"`
	PolicyID    string `json:"policy_id"`
}

// RiskThreatRow represents a risk-to-threat cross-reference.
type RiskThreatRow struct {
	CatalogID         string `json:"catalog_id"`
	RiskID            string `json:"risk_id"`
	ThreatReferenceID string `json:"threat_reference_id"`
	ThreatEntryID     string `json:"threat_entry_id"`
}

// ParseRiskCatalog extracts risk rows and risk-to-threat links from a RiskCatalog YAML body.
func ParseRiskCatalog(ctx context.Context, content, catalogID, policyID string) ([]RiskRow, []RiskThreatRow, error) {
	f := NewMemoryFetcher(map[string][]byte{artifactSource: []byte(content)})
	catalog, err := sdk.Load[sdk.RiskCatalog](ctx, f, artifactSource)
	if err != nil {
		return nil, nil, fmt.Errorf("load risk catalog: %w", err)
	}

	resolvedID := catalogID
	if resolvedID == "" {
		resolvedID = catalog.Metadata.Id
	}

	var riskRows []RiskRow
	var threatRows []RiskThreatRow

	for _, risk := range catalog.Risks {
		riskRows = append(riskRows, RiskRow{
			CatalogID:   resolvedID,
			RiskID:      risk.Id,
			Title:       risk.Title,
			Description: risk.Description,
			Severity:    risk.Severity.String(),
			GroupID:     risk.Group,
			Impact:      risk.Impact,
			PolicyID:    policyID,
		})
		for _, mem := range risk.Threats {
			for _, ent := range mem.Entries {
				threatRows = append(threatRows, RiskThreatRow{
					CatalogID:         resolvedID,
					RiskID:            risk.Id,
					ThreatReferenceID: mem.ReferenceId,
					ThreatEntryID:     ent.ReferenceId,
				})
			}
		}
	}
	return riskRows, threatRows, nil
}
