// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/complytime/complytime-studio/internal/consts"
)

// RegisterUserAPI mounts user management endpoints on the given mux.
// All endpoints require authentication (enforced by the auth middleware).
// Write operations (PATCH) are further gated by the admin guard in writeProtect.
func (h *Handler) RegisterUserAPI(mux *http.ServeMux) {
	if h.users == nil {
		return
	}
	mux.HandleFunc("GET /api/users", h.handleListUsers)
	mux.HandleFunc("PATCH /api/users/{email}/role", h.handleSetRole)
	mux.HandleFunc("GET /api/role-changes", h.handleListRoleChanges)
	mux.HandleFunc("GET /api/setup-status", h.handleSetupStatus)
	mux.HandleFunc("POST /api/bootstrap", h.handleBootstrap)
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	u, err := h.users.GetUser(r.Context(), sess.Email)
	if err != nil || u.Role != consts.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}

	users, err := h.users.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list users"})
		slog.Error("list users failed", "error", err)
		return
	}
	if users == nil {
		users = []User{}
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) handleSetRole(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	caller, err := h.users.GetUser(r.Context(), sess.Email)
	if err != nil || caller.Role != consts.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}

	targetEmail := r.PathValue("email")
	if targetEmail == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email path parameter required"})
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if body.Role != consts.RoleAdmin && body.Role != consts.RoleReviewer {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role must be 'admin' or 'reviewer'"})
		return
	}

	oldRole, err := h.users.SetRole(r.Context(), targetEmail, body.Role)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		slog.Error("set role failed", "target", targetEmail, "error", err)
		return
	}

	if oldRole != body.Role {
		change := RoleChange{
			ChangedBy:   sess.Email,
			TargetEmail: targetEmail,
			OldRole:     oldRole,
			NewRole:     body.Role,
		}
		if err := h.users.InsertRoleChange(r.Context(), change); err != nil {
			slog.Error("role change audit log failed", "error", err)
		}
		slog.Info("role changed", "changed_by", sess.Email, "target", targetEmail,
			"old_role", oldRole, "new_role", body.Role)
	}

	writeJSON(w, http.StatusOK, map[string]string{"email": targetEmail, "role": body.Role})
}

func (h *Handler) handleListRoleChanges(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	u, err := h.users.GetUser(r.Context(), sess.Email)
	if err != nil || u.Role != consts.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}

	changes, err := h.users.ListRoleChanges(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list role changes"})
		slog.Error("list role changes failed", "error", err)
		return
	}
	if changes == nil {
		changes = []RoleChange{}
	}
	writeJSON(w, http.StatusOK, changes)
}

func (h *Handler) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	count, err := h.users.CountAdmins(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"needs_setup": count == 0})
}

// handleBootstrap promotes the caller to admin when no admins exist.
func (h *Handler) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	count, err := h.users.CountAdmins(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	if count > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "admin already exists"})
		return
	}

	oldRole, err := h.users.SetRole(r.Context(), sess.Email, consts.RoleAdmin)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "promotion failed"})
		slog.Error("bootstrap promotion failed", "email", sess.Email, "error", err)
		return
	}
	_ = h.users.InsertRoleChange(r.Context(), RoleChange{
		ChangedBy:   "bootstrap",
		TargetEmail: sess.Email,
		OldRole:     oldRole,
		NewRole:     consts.RoleAdmin,
	})
	slog.Info("bootstrap: user promoted to admin", "email", sess.Email)
	writeJSON(w, http.StatusOK, map[string]string{"email": sess.Email, "role": consts.RoleAdmin})
}
