// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func registerProgramRoutes(g *echo.Group, s Stores) {
	g.GET("/programs", listProgramsHandler(s.Programs))
	g.POST("/programs", createProgramHandler(s.Programs))
	if s.Jobs != nil {
		g.GET("/programs/:id/jobs", listJobsHandler(s.Jobs))
		g.POST("/programs/:id/jobs", createJobHandler(s.Jobs))
	}
	g.GET("/programs/:id", getProgramHandler(s.Programs))
	g.PUT("/programs/:id", updateProgramHandler(s.Programs, s.PostureComputer))
	g.DELETE("/programs/:id", deleteProgramHandler(s.Programs))
}

func listProgramsHandler(s ProgramStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := s.ListPrograms(c.Request().Context())
		if err != nil {
			slog.Error("list programs failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []Program{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func getProgramHandler(s ProgramStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return jsonError(c, http.StatusBadRequest, "missing program id")
		}
		p, err := s.GetProgram(c.Request().Context(), id)
		if err != nil {
			if errors.Is(err, ErrProgramNotFound) {
				return jsonError(c, http.StatusNotFound, "not found")
			}
			slog.Error("get program failed", "error", err, "id", id)
			return jsonError(c, http.StatusInternalServerError, "internal error")
		}
		return c.JSON(http.StatusOK, p)
	}
}

func createProgramHandler(s ProgramStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var p Program
		if err := c.Bind(&p); err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid json")
		}
		if p.Name == "" || p.Framework == "" {
			return jsonError(c, http.StatusBadRequest, "name and framework required")
		}
		out, err := s.CreateProgram(c.Request().Context(), p)
		if err != nil {
			slog.Error("create program failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "insert failed")
		}
		return c.JSON(http.StatusCreated, out)
	}
}

func updateProgramHandler(s ProgramStore, pc PostureComputer) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return jsonError(c, http.StatusBadRequest, "missing program id")
		}
		var p Program
		if err := c.Bind(&p); err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid json")
		}
		p.ID = id
		if err := s.UpdateProgram(c.Request().Context(), p); err != nil {
			if errors.Is(err, ErrProgramVersionConflict) {
				return jsonError(c, http.StatusConflict, "version conflict")
			}
			slog.Error("update program failed", "error", err, "id", id)
			return jsonError(c, http.StatusInternalServerError, "update failed")
		}
		if pc != nil && len(p.PolicyIDs) > 0 {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				greenPct, redPct := p.GreenPct, p.RedPct
				if greenPct == 0 {
					greenPct = 90
				}
				if redPct == 0 {
					redPct = 50
				}
				if err := pc.RecomputePosture(ctx, id, p.PolicyIDs, greenPct, redPct); err != nil {
					slog.Warn("posture recompute after update failed", "program_id", id, "error", err)
				}
			}()
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

func deleteProgramHandler(s ProgramStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return jsonError(c, http.StatusBadRequest, "missing program id")
		}
		if err := s.DeleteProgram(c.Request().Context(), id); err != nil {
			if errors.Is(err, ErrProgramNotFound) {
				return jsonError(c, http.StatusNotFound, "not found")
			}
			slog.Error("delete program failed", "error", err, "id", id)
			return jsonError(c, http.StatusInternalServerError, "delete failed")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func listJobsHandler(s JobStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		programID := c.Param("id")
		if programID == "" {
			return jsonError(c, http.StatusBadRequest, "missing program id")
		}
		rows, err := s.ListJobs(c.Request().Context(), programID)
		if err != nil {
			slog.Error("list jobs failed", "error", err, "program_id", programID)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		if rows == nil {
			rows = []Job{}
		}
		return c.JSON(http.StatusOK, rows)
	}
}

func createJobHandler(s JobStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		programID := c.Param("id")
		if programID == "" {
			return jsonError(c, http.StatusBadRequest, "missing program id")
		}
		var j Job
		if err := c.Bind(&j); err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid json")
		}
		j.ProgramID = programID
		if j.Agent == "" || j.UserID == "" {
			return jsonError(c, http.StatusBadRequest, "agent and user_id required")
		}
		out, err := s.CreateJob(c.Request().Context(), j)
		if err != nil {
			slog.Error("create job failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "insert failed")
		}
		return c.JSON(http.StatusCreated, out)
	}
}
