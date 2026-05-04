// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"fmt"

	sdk "github.com/gemaraproj/go-gemara"
)

// GuidanceRow represents a single guideline parsed from a GuidanceDocument.
type GuidanceRow struct {
	CatalogID   string `json:"catalog_id"`
	GuidelineID string `json:"guideline_id"`
	Title       string `json:"title"`
	Objective   string `json:"objective"`
	GroupID     string `json:"group_id"`
	State       string `json:"state"`
}

// ParseGuidanceCatalog extracts guideline rows from a GuidanceCatalog YAML body.
func ParseGuidanceCatalog(ctx context.Context, content, catalogID string) ([]GuidanceRow, error) {
	f := NewMemoryFetcher(map[string][]byte{artifactSource: []byte(content)})
	doc, err := sdk.Load[sdk.GuidanceCatalog](ctx, f, artifactSource)
	if err != nil {
		return nil, fmt.Errorf("load guidance catalog: %w", err)
	}

	resolvedID := catalogID
	if resolvedID == "" {
		resolvedID = doc.Metadata.Id
	}
	if resolvedID == "" {
		return nil, fmt.Errorf("guidance catalog has no id and none was provided")
	}

	var rows []GuidanceRow
	for _, g := range doc.Guidelines {
		if g.Id == "" {
			continue
		}
		rows = append(rows, GuidanceRow{
			CatalogID:   resolvedID,
			GuidelineID: g.Id,
			Title:       g.Title,
			Objective:   g.Objective,
			GroupID:     g.Group,
			State:       "Active",
		})
	}
	return rows, nil
}
