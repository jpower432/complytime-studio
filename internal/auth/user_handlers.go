// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/labstack/echo/v4"
)

func (h *Handler) RegisterUserAPI(g *echo.Group) {
	if h.users == nil {
		return
	}
	g.GET("/users", h.handleListUsers)
	g.PATCH("/users/:email/role", h.handleSetRole)
	g.GET("/role-changes", h.handleListRoleChanges)
	g.GET("/setup-status", h.handleSetupStatus)
	g.POST("/bootstrap", h.handleBootstrap)
}

func (h *Handler) handleListUsers(c echo.Context) error {
	sess, ok := SessionFrom(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}
	u, err := h.users.GetUser(c.Request().Context(), sess.Email)
	if err != nil || u.Role != consts.RoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "admin role required"})
	}

	users, err := h.users.ListUsers(c.Request().Context())
	if err != nil {
		slog.Error("list users failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list users"})
	}
	if users == nil {
		users = []User{}
	}
	return c.JSON(http.StatusOK, users)
}

func (h *Handler) handleSetRole(c echo.Context) error {
	sess, ok := SessionFrom(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}
	caller, err := h.users.GetUser(c.Request().Context(), sess.Email)
	if err != nil || caller.Role != consts.RoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "admin role required"})
	}

	rawEmail := c.Param("email")
	targetEmail, _ := url.PathUnescape(rawEmail)
	if targetEmail == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email path parameter required"})
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	if body.Role != consts.RoleAdmin && body.Role != consts.RoleReviewer {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role must be 'admin' or 'reviewer'"})
	}

	oldRole, err := h.users.SetRole(c.Request().Context(), targetEmail, body.Role)
	if err != nil {
		slog.Error("set role failed", "target", targetEmail, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	if oldRole != body.Role {
		change := RoleChange{
			ChangedBy:   sess.Email,
			TargetEmail: targetEmail,
			OldRole:     oldRole,
			NewRole:     body.Role,
		}
		if err := h.users.InsertRoleChange(c.Request().Context(), change); err != nil {
			slog.Error("role change audit log failed", "error", err)
		}
		slog.Info("role changed", "changed_by", sess.Email, "target", targetEmail,
			"old_role", oldRole, "new_role", body.Role)
	}

	return c.JSON(http.StatusOK, map[string]string{"email": targetEmail, "role": body.Role})
}

func (h *Handler) handleListRoleChanges(c echo.Context) error {
	sess, ok := SessionFrom(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}
	u, err := h.users.GetUser(c.Request().Context(), sess.Email)
	if err != nil || u.Role != consts.RoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "admin role required"})
	}

	changes, err := h.users.ListRoleChanges(c.Request().Context())
	if err != nil {
		slog.Error("list role changes failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list role changes"})
	}
	if changes == nil {
		changes = []RoleChange{}
	}
	return c.JSON(http.StatusOK, changes)
}

func (h *Handler) handleSetupStatus(c echo.Context) error {
	count, err := h.users.CountAdmins(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
	}
	return c.JSON(http.StatusOK, map[string]any{"needs_setup": count == 0})
}

func (h *Handler) handleBootstrap(c echo.Context) error {
	sess, ok := SessionFrom(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	oldRole, err := h.users.BootstrapAdmin(c.Request().Context(), sess.Email)
	if err != nil {
		if err == ErrAdminExists {
			return c.JSON(http.StatusConflict, map[string]string{"error": "admin already exists"})
		}
		slog.Error("bootstrap promotion failed", "email", sess.Email, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "promotion failed"})
	}
	_ = h.users.InsertRoleChange(c.Request().Context(), RoleChange{
		ChangedBy:   "bootstrap",
		TargetEmail: sess.Email,
		OldRole:     oldRole,
		NewRole:     consts.RoleAdmin,
	})
	slog.Info("bootstrap: user promoted to admin", "email", sess.Email)
	return c.JSON(http.StatusOK, map[string]string{"email": sess.Email, "role": consts.RoleAdmin})
}
