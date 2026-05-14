// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"fmt"

	sdk "github.com/gemaraproj/go-gemara"
	goyaml "github.com/goccy/go-yaml"
)

// GuidanceEntryRow represents a single guideline parsed from a GuidanceCatalog.
type GuidanceEntryRow struct {
	CatalogID     string
	GuidelineID   string
	Title         string
	Objective     string
	GroupID       string
	State         string
	Applicability []string
}

// ParseGuidanceCatalog extracts guideline rows from a GuidanceCatalog YAML body.
func ParseGuidanceCatalog(ctx context.Context, content, catalogID string) ([]GuidanceEntryRow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var catalog sdk.GuidanceCatalog
	if err := goyaml.Unmarshal([]byte(content), &catalog); err != nil {
		return nil, fmt.Errorf("unmarshal guidance catalog: %w", err)
	}

	resolvedID := catalogID
	if resolvedID == "" {
		resolvedID = catalog.Metadata.Id
	}
	if resolvedID == "" {
		return nil, fmt.Errorf("guidance catalog has no id and none was provided")
	}

	var rows []GuidanceEntryRow
	for _, g := range catalog.Guidelines {
		if g.Id == "" {
			continue
		}
		app := g.Applicability
		if app == nil {
			app = []string{}
		}
		rows = append(rows, GuidanceEntryRow{
			CatalogID:     resolvedID,
			GuidelineID:   g.Id,
			Title:         g.Title,
			Objective:     g.Objective,
			GroupID:       g.Group,
			State:         lifecycleString(g.State),
			Applicability: app,
		})
	}
	return rows, nil
}
