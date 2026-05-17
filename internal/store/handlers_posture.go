// SPDX-License-Identifier: Apache-2.0

package store

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/complytime-labs/complytime-core/internal/consts"
	gemarapkg "github.com/complytime-labs/complytime-core/internal/gemara"
)

func registerPostureAndRequirementRoutes(g *echo.Group, s Stores) {
	// Posture is a cross-record aggregation that belongs in the workbench
	// (ADR 0039). Kept here temporarily as an internal-only endpoint until
	// the workbench implements its own aggregation logic.
	if s.Posture != nil {
		g.GET("/posture", listPostureHandler(s.Posture))
	}
	if s.Requirements != nil {
		g.GET("/requirements", listRequirementMatrixHandler(s.Requirements))
		g.GET("/requirements/:id/evidence", listRequirementEvidenceHandler(s.Requirements))
	}
}

func registerThreatAndRiskRoutes(g *echo.Group, s Stores) {
	if s.Threats != nil {
		g.GET("/threats", listThreatsHandler(s.Threats))
		g.GET("/control-threats", listControlThreatsHandler(s.Threats))
	}
	if s.Risks != nil {
		g.GET("/risks", listRisksHandler(s.Risks))
		// Risk severity is a cross-record derivation that belongs in the
		// workbench (ADR 0039). Kept here temporarily until the workbench
		// implements its own aggregation logic.
		g.GET("/risks/severity", riskSeverityHandler(s.Risks))
		g.GET("/risk-threats", listRiskThreatsHandler(s.Risks))
	}
}

func listPostureHandler(s PostureStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		start, end, err := parseOptionalTimeRange(c)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, err.Error())
		}
		rows, err := s.ListPosture(c.Request().Context(), start, end)
		if err != nil {
			slog.Error("list posture failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []PostureRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

// parseQueryLimit extracts an optional "limit" query parameter and clamps it
// to the project-wide range [DefaultQueryLimit, MaxQueryLimit].
func parseQueryLimit(c echo.Context) int {
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return consts.ClampLimit(n)
		}
	}
	return consts.ClampLimit(0)
}

// parseOptionalTimeRange extracts optional start/end query parameters.
// Accepts date-only (2006-01-02) or RFC 3339 formats.
func parseOptionalTimeRange(c echo.Context) (start, end time.Time, err error) {
	if v := c.QueryParam("start"); v != "" {
		start, err = parseFlexibleTime(v, false)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidStart
		}
	}
	if v := c.QueryParam("end"); v != "" {
		end, err = parseFlexibleTime(v, true)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidEnd
		}
	}
	return start, end, nil
}

var (
	errInvalidStart = errors.New("invalid start parameter")
	errInvalidEnd   = errors.New("invalid end parameter")
)

// parseFlexibleTime parses RFC 3339 or date-only (YYYY-MM-DD) strings.
// Date-only values are treated as end-of-day (next day at 00:00 minus 1ns)
// when used as an upper bound so that the full calendar day is included.
func parseFlexibleTime(s string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if endOfDay {
			t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}
		return t, nil
	}
	return time.Time{}, errInvalidDateFormat
}

var errInvalidDateFormat = errors.New("expected YYYY-MM-DD or RFC 3339 format")

func listRequirementMatrixHandler(s RequirementStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		policyID := c.QueryParam("policy_id")
		if policyID == "" {
			return jsonError(c, http.StatusBadRequest, "policy_id required")
		}

		f := RequirementFilter{PolicyID: policyID}

		if v := c.QueryParam("audit_start"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err != nil {
				return jsonError(c, http.StatusBadRequest, "invalid audit_start format")
			}
			f.Start = t
		}
		if v := c.QueryParam("audit_end"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err != nil {
				return jsonError(c, http.StatusBadRequest, "invalid audit_end format")
			}
			f.End = t
		}
		if !f.Start.IsZero() && !f.End.IsZero() && f.End.Before(f.Start) {
			return jsonError(c, http.StatusBadRequest, "audit_end must be >= audit_start")
		}

		f.Classification = c.QueryParam("classification")
		f.ControlFamily = c.QueryParam("control_family")

		if v := c.QueryParam("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Limit = n
			}
		}
		if v := c.QueryParam("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Offset = n
			}
		}
		f.Limit = consts.ClampLimit(f.Limit)

		rows, err := s.ListRequirementMatrix(c.Request().Context(), f)
		if err != nil {
			slog.Error("list requirement matrix failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []RequirementRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listRequirementEvidenceHandler(s RequirementStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqID := c.Param("id")
		if reqID == "" {
			return jsonError(c, http.StatusBadRequest, "missing requirement id")
		}
		policyID := c.QueryParam("policy_id")
		if policyID == "" {
			return jsonError(c, http.StatusBadRequest, "policy_id required")
		}

		f := RequirementFilter{PolicyID: policyID}
		if v := c.QueryParam("audit_start"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err == nil {
				f.Start = t
			}
		}
		if v := c.QueryParam("audit_end"); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
			}
			if err == nil {
				f.End = t
			}
		}
		if v := c.QueryParam("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Limit = n
			}
		}
		if v := c.QueryParam("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.Offset = n
			}
		}
		f.Limit = consts.ClampLimit(f.Limit)

		rows, err := s.ListRequirementEvidence(c.Request().Context(), reqID, f)
		if err != nil {
			if errors.Is(err, ErrRequirementNotFound) {
				return jsonError(c, http.StatusNotFound, "not found")
			}
			slog.Error("list requirement evidence failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []RequirementEvidenceRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listThreatsHandler(s ThreatStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		catalogID := c.QueryParam("catalog_id")
		policyID := c.QueryParam("policy_id")
		limit := parseQueryLimit(c)
		rows, err := s.QueryThreats(c.Request().Context(), catalogID, policyID, limit)
		if err != nil {
			slog.Error("query threats failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []gemarapkg.ThreatRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listControlThreatsHandler(s ThreatStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		catalogID := c.QueryParam("catalog_id")
		controlID := c.QueryParam("control_id")
		limit := parseQueryLimit(c)
		rows, err := s.QueryControlThreats(c.Request().Context(), catalogID, controlID, limit)
		if err != nil {
			slog.Error("query control threats failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []gemarapkg.ControlThreatRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listRisksHandler(s RiskStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		catalogID := c.QueryParam("catalog_id")
		policyID := c.QueryParam("policy_id")
		limit := parseQueryLimit(c)
		rows, err := s.QueryRisks(c.Request().Context(), catalogID, policyID, limit)
		if err != nil {
			slog.Error("query risks failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []gemarapkg.RiskRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func listRiskThreatsHandler(s RiskStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		catalogID := c.QueryParam("catalog_id")
		riskID := c.QueryParam("risk_id")
		limit := parseQueryLimit(c)
		rows, err := s.QueryRiskThreats(c.Request().Context(), catalogID, riskID, limit)
		if err != nil {
			slog.Error("query risk threats failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []gemarapkg.RiskThreatRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func riskSeverityHandler(s RiskStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		policyID := c.QueryParam("policy_id")
		if policyID == "" {
			return jsonError(c, http.StatusBadRequest, "policy_id required")
		}
		rows, err := s.GetPolicyRiskSeverity(c.Request().Context(), policyID)
		if err != nil {
			slog.Error("risk severity query failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []RiskSeverityRow{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}
