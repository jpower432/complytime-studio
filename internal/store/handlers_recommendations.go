// SPDX-License-Identifier: Apache-2.0

package store

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/complytime/complytime-studio/internal/auth"
	"github.com/complytime/complytime-studio/internal/identity"
)

func registerRecommendationRoutes(g *echo.Group, s Stores) {
	if s.Recommender == nil || s.Users == nil {
		return
	}
	g.GET("/programs/:id/recommendations", listRecommendationsHandler(s.Recommender, s.Users))
	g.POST("/programs/:id/recommendations/:policyId/dismiss", dismissRecommendationHandler(s.Recommender, s.Programs))
	g.DELETE("/programs/:id/recommendations/:policyId/dismiss", undismissRecommendationHandler(s.Recommender, s.Programs))
	g.POST("/programs/:id/recommendations/:policyId/attach", attachRecommendedPolicyHandler(s.Programs))
}

func listRecommendationsHandler(e Recommender, users identity.UserStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if auth.RejectUnlessWriterOrAdmin(c, users) {
			return nil
		}
		programID := c.Param("id")
		if programID == "" {
			return jsonError(c, http.StatusBadRequest, "missing program id")
		}
		recs, err := e.ForProgram(c.Request().Context(), programID)
		if err != nil {
			if errors.Is(err, ErrProgramNotFound) {
				return jsonError(c, http.StatusNotFound, "not found")
			}
			slog.Error("recommendations failed", "error", err, "program_id", programID)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if recs == nil {
			recs = []Recommendation{}
		}
		return c.JSON(http.StatusOK, recs)
	}
}

func dismissRecommendationHandler(e Recommender, programs ProgramStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		programID := c.Param("id")
		policyID := c.Param("policyId")
		if programID == "" || policyID == "" {
			return jsonError(c, http.StatusBadRequest, "program id and policy id required")
		}
		if _, err := programs.GetProgram(c.Request().Context(), programID); err != nil {
			if errors.Is(err, ErrProgramNotFound) {
				return jsonError(c, http.StatusNotFound, "not found")
			}
			slog.Error("dismiss recommendation: load program", "error", err)
			return jsonError(c, http.StatusInternalServerError, "internal error")
		}
		userID := authSessionEmail(c.Request().Context())
		if err := e.Dismiss(c.Request().Context(), programID, policyID, userID); err != nil {
			slog.Error("dismiss recommendation failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "update failed")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "dismissed"})
	}
}

func undismissRecommendationHandler(e Recommender, programs ProgramStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		programID := c.Param("id")
		policyID := c.Param("policyId")
		if programID == "" || policyID == "" {
			return jsonError(c, http.StatusBadRequest, "program id and policy id required")
		}
		if _, err := programs.GetProgram(c.Request().Context(), programID); err != nil {
			if errors.Is(err, ErrProgramNotFound) {
				return jsonError(c, http.StatusNotFound, "not found")
			}
			slog.Error("undismiss recommendation: load program", "error", err)
			return jsonError(c, http.StatusInternalServerError, "internal error")
		}
		if err := e.Undismiss(c.Request().Context(), programID, policyID); err != nil {
			slog.Error("undismiss recommendation failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "update failed")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

func attachRecommendedPolicyHandler(programs ProgramStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		programID := c.Param("id")
		policyID := c.Param("policyId")
		if programID == "" || policyID == "" {
			return jsonError(c, http.StatusBadRequest, "program id and policy id required")
		}
		p, err := programs.GetProgram(c.Request().Context(), programID)
		if err != nil {
			if errors.Is(err, ErrProgramNotFound) {
				return jsonError(c, http.StatusNotFound, "not found")
			}
			slog.Error("attach policy: load program", "error", err)
			return jsonError(c, http.StatusInternalServerError, "internal error")
		}
		for _, existing := range p.PolicyIDs {
			if existing == policyID {
				return c.JSON(http.StatusOK, map[string]string{"status": "already_attached"})
			}
		}
		p.PolicyIDs = append(p.PolicyIDs, policyID)
		if err := programs.UpdateProgram(c.Request().Context(), *p); err != nil {
			if errors.Is(err, ErrProgramVersionConflict) {
				return jsonError(c, http.StatusConflict, "version conflict")
			}
			slog.Error("attach policy: update program", "error", err)
			return jsonError(c, http.StatusInternalServerError, "update failed")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "attached"})
	}
}
