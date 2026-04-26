// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"testing"
)

func TestParseRiskCatalog(t *testing.T) {
	t.Parallel()

	const multiRiskYAML = `title: Sample Risk Catalog
metadata:
  id: rc-sample
  type: RiskCatalog
risks:
  - id: R-1
    title: First Risk
    description: Desc one
    group: G1
    severity: High
    impact: Operational disruption
    threats:
      - reference-id: tc-ref
        entries:
          - reference-id: T-1
          - reference-id: T-2
  - id: R-2
    title: Second Risk
    description: Desc two
    group: G1
    severity: Critical
    threats:
      - reference-id: tc-ref
        entries:
          - reference-id: T-3
`

	const noThreatsYAML = `title: Empty threats
metadata:
  id: rc-not
  type: RiskCatalog
risks:
  - id: R-solo
    title: Lone Risk
    description: No threats
    group: G0
    severity: Low
`

	tests := []struct {
		name               string
		yaml               string
		catalogID          string
		policyID           string
		wantRiskCount      int
		wantLinkCount      int
		wantSeverity       string
		wantRiskID         string
		wantFirstCatalogID string
	}{
		{
			name:               "multi risk catalog",
			yaml:               multiRiskYAML,
			catalogID:          "",
			policyID:           "pol-1",
			wantRiskCount:      2,
			wantLinkCount:      3,
			wantSeverity:       "High",
			wantRiskID:         "R-1",
			wantFirstCatalogID: "rc-sample",
		},
		{
			name:               "risk with no threats",
			yaml:               noThreatsYAML,
			catalogID:          "",
			policyID:           "",
			wantRiskCount:      1,
			wantLinkCount:      0,
			wantSeverity:       "Low",
			wantRiskID:         "R-solo",
			wantFirstCatalogID: "rc-not",
		},
		{
			name:               "explicit catalog id override",
			yaml:               noThreatsYAML,
			catalogID:          "override-id",
			policyID:           "p",
			wantRiskCount:      1,
			wantLinkCount:      0,
			wantSeverity:       "Low",
			wantRiskID:         "R-solo",
			wantFirstCatalogID: "override-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			risks, links, err := ParseRiskCatalog(tt.yaml, tt.catalogID, tt.policyID)
			if err != nil {
				t.Fatalf("ParseRiskCatalog: %v", err)
			}
			if len(risks) != tt.wantRiskCount {
				t.Fatalf("risk rows: got %d, want %d", len(risks), tt.wantRiskCount)
			}
			if len(links) != tt.wantLinkCount {
				t.Fatalf("risk-threat rows: got %d, want %d", len(links), tt.wantLinkCount)
			}
			if tt.wantRiskCount > 0 {
				first := risks[0]
				if first.Severity != tt.wantSeverity {
					t.Fatalf("severity: got %q, want %q", first.Severity, tt.wantSeverity)
				}
				if first.RiskID != tt.wantRiskID {
					t.Fatalf("risk_id: got %q, want %q", first.RiskID, tt.wantRiskID)
				}
				if first.CatalogID != tt.wantFirstCatalogID {
					t.Fatalf("catalog_id: got %q, want %q", first.CatalogID, tt.wantFirstCatalogID)
				}
				if first.PolicyID != tt.policyID {
					t.Fatalf("policy_id: got %q, want %q", first.PolicyID, tt.policyID)
				}
			}
			if tt.wantLinkCount > 0 {
				if links[0].ThreatReferenceID != "tc-ref" || links[0].ThreatEntryID != "T-1" {
					t.Fatalf("first link: %+v", links[0])
				}
			}
		})
	}
}
