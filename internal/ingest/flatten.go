// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"fmt"
	"time"

	gemara "github.com/gemaraproj/go-gemara"
)

// parseDatetime converts a gemara.Datetime (ISO 8601 string) to *time.Time.
// Returns nil when the input is empty.
func parseDatetime(dt gemara.Datetime) (*time.Time, error) {
	s := string(dt)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("parse datetime %q: %w", s, err)
	}
	return &t, nil
}

// FlattenEvaluationLog produces one EvalRow per AssessmentLog entry.
// policyID is derived by the caller from metadata.mapping-references.
func FlattenEvaluationLog(log *gemara.EvaluationLog, policyID string) ([]EvalRow, error) {
	if len(log.Evaluations) == 0 {
		return nil, fmt.Errorf("evaluation log %s has no evaluations", log.Metadata.Id)
	}

	var rows []EvalRow
	for _, eval := range log.Evaluations {
		if eval == nil {
			continue
		}
		for _, al := range eval.AssessmentLogs {
			if al == nil {
				continue
			}

			start, err := parseDatetime(al.Start)
			if err != nil {
				return nil, err
			}
			end, err := parseDatetime(al.End)
			if err != nil {
				return nil, err
			}

			var collectedAt time.Time
			if start != nil {
				collectedAt = *start
			}

			row := EvalRow{
				LogID:            log.Metadata.Id,
				TargetID:         log.Target.Id,
				TargetEnv:        log.Target.Environment,
				PolicyID:         policyID,
				CatalogRefID:     eval.Control.ReferenceId,
				ControlID:        eval.Control.EntryId,
				ControlName:      eval.Name,
				ControlResult:    eval.Result.String(),
				RequirementID:    al.Requirement.EntryId,
				AssessmentResult: al.Result.String(),
				Message:          al.Message,
				Description:      al.Description,
				Applicability:    al.Applicability,
				ConfidenceLevel:  al.ConfidenceLevel.String(),
				CollectedAt:      collectedAt,
				CompletedAt:      end,
			}

			if al.Plan != nil {
				id := al.Plan.EntryId
				row.PlanID = &id
			}
			if al.StepsExecuted > 0 {
				row.StepsExecuted = uint16(al.StepsExecuted)
			}
			if al.Recommendation != "" {
				row.Recommendation = &al.Recommendation
			}

			rows = append(rows, row)
		}
	}
	return rows, nil
}

// FlattenEnforcementLog produces one EnforcementRow per AssessmentFinding.
// policyID is derived by the caller from metadata.mapping-references.
func FlattenEnforcementLog(log *gemara.EnforcementLog, policyID string) ([]EnforcementRow, error) {
	if len(log.Actions) == 0 {
		return nil, fmt.Errorf("enforcement log %s has no actions", log.Metadata.Id)
	}

	var rows []EnforcementRow
	for _, action := range log.Actions {
		if action == nil {
			continue
		}

		exceptionRefs := make([]string, 0, len(action.Justification.Exceptions))
		for _, exc := range action.Justification.Exceptions {
			exceptionRefs = append(exceptionRefs, exc.ReferenceId)
		}
		hasException := len(exceptionRefs) > 0

		var msg *string
		if action.Message != nil && *action.Message != "" {
			msg = action.Message
		}

		start, err := parseDatetime(action.Start)
		if err != nil {
			return nil, err
		}
		end, err := parseDatetime(action.End)
		if err != nil {
			return nil, err
		}

		var startedAt time.Time
		if start != nil {
			startedAt = *start
		}

		for _, af := range action.Justification.Assessments {
			controlID := af.Requirement.ReferenceId
			requirementID := af.Requirement.EntryId

			rows = append(rows, EnforcementRow{
				LogID:            log.Metadata.Id,
				TargetID:         log.Target.Id,
				TargetEnv:        log.Target.Environment,
				PolicyID:         policyID,
				CatalogRefID:     action.Method.ReferenceId,
				ControlID:        controlID,
				RequirementID:    requirementID,
				Disposition:      action.Disposition.String(),
				MethodID:         action.Method.EntryId,
				AssessmentResult: af.Result.String(),
				EvalLogRef:       af.Log.EntryId,
				Message:          msg,
				HasException:     hasException,
				ExceptionRefs:    exceptionRefs,
				StartedAt:        startedAt,
				CompletedAt:      end,
			})
		}
	}
	return rows, nil
}
