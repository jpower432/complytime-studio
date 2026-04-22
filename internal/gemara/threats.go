// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"fmt"

	sdk "github.com/gemaraproj/go-gemara"
	goyaml "github.com/goccy/go-yaml"
)

// ThreatRow represents a single threat parsed from a ThreatCatalog.
type ThreatRow struct {
	CatalogID   string `json:"catalog_id"`
	ThreatID    string `json:"threat_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	GroupID     string `json:"group_id"`
	PolicyID    string `json:"policy_id"`
}

// ParseThreatCatalog extracts threat rows from a ThreatCatalog YAML body.
func ParseThreatCatalog(content, catalogID, policyID string) ([]ThreatRow, error) {
	var catalog sdk.ThreatCatalog
	if err := goyaml.Unmarshal([]byte(content), &catalog); err != nil {
		return nil, fmt.Errorf("parse threat catalog YAML: %w", err)
	}

	resolvedID := catalogID
	if resolvedID == "" {
		resolvedID = catalog.Metadata.Id
	}

	var rows []ThreatRow
	for _, t := range catalog.Threats {
		rows = append(rows, ThreatRow{
			CatalogID:   resolvedID,
			ThreatID:    t.Id,
			Title:       t.Title,
			Description: t.Description,
			GroupID:     t.Group,
			PolicyID:    policyID,
		})
	}
	return rows, nil
}
