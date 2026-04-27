// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"fmt"

	sdk "github.com/gemaraproj/go-gemara"
)

// ControlRow represents a single control parsed from a ControlCatalog.
type ControlRow struct {
	CatalogID string `json:"catalog_id"`
	ControlID string `json:"control_id"`
	Title     string `json:"title"`
	Objective string `json:"objective"`
	GroupID   string `json:"group_id"`
	State     string `json:"state"`
	PolicyID  string `json:"policy_id"`
}

// AssessmentRequirementRow represents a single assessment requirement parsed from a ControlCatalog.
type AssessmentRequirementRow struct {
	CatalogID      string   `json:"catalog_id"`
	ControlID      string   `json:"control_id"`
	RequirementID  string   `json:"requirement_id"`
	Text           string   `json:"text"`
	Applicability  []string `json:"applicability"`
	Recommendation string   `json:"recommendation"`
	State          string   `json:"state"`
}

// ControlThreatRow represents a control-to-threat cross-reference.
type ControlThreatRow struct {
	CatalogID         string `json:"catalog_id"`
	ControlID         string `json:"control_id"`
	ThreatReferenceID string `json:"threat_reference_id"`
	ThreatEntryID     string `json:"threat_entry_id"`
}

// ParseControlCatalog extracts controls, assessment requirements, and control-to-threat links
// from a ControlCatalog YAML body.
func ParseControlCatalog(ctx context.Context, content, catalogID, policyID string) (
	[]ControlRow,
	[]AssessmentRequirementRow,
	[]ControlThreatRow,
	error,
) {
	f := NewMemoryFetcher(map[string][]byte{artifactSource: []byte(content)})
	catalog, err := sdk.Load[sdk.ControlCatalog](ctx, f, artifactSource)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load control catalog: %w", err)
	}

	resolvedID := catalogID
	if resolvedID == "" {
		resolvedID = catalog.Metadata.Id
	}

	var controls []ControlRow
	var requirements []AssessmentRequirementRow
	var threats []ControlThreatRow

	for _, c := range catalog.Controls {
		state := lifecycleString(c.State)
		controls = append(controls, ControlRow{
			CatalogID: resolvedID,
			ControlID: c.Id,
			Title:     c.Title,
			Objective: c.Objective,
			GroupID:   c.Group,
			State:     state,
			PolicyID:  policyID,
		})
		for _, ar := range c.AssessmentRequirements {
			requirements = append(requirements, AssessmentRequirementRow{
				CatalogID:      resolvedID,
				ControlID:      c.Id,
				RequirementID:  ar.Id,
				Text:           ar.Text,
				Applicability:  ar.Applicability,
				Recommendation: ar.Recommendation,
				State:          lifecycleString(ar.State),
			})
		}
		for _, mem := range c.Threats {
			for _, ent := range mem.Entries {
				threats = append(threats, ControlThreatRow{
					CatalogID:         resolvedID,
					ControlID:         c.Id,
					ThreatReferenceID: mem.ReferenceId,
					ThreatEntryID:     ent.ReferenceId,
				})
			}
		}
	}
	return controls, requirements, threats, nil
}

func lifecycleString(l sdk.Lifecycle) string {
	s := l.String()
	if s == "" {
		return "Active"
	}
	return s
}
