// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"time"
)

// ErrUserNotFound is returned when a user lookup finds no matching row.
var ErrUserNotFound = errors.New("user not found")

// User represents a registered user with a role assignment.
type User struct {
	Sub       string    `json:"sub"`
	Issuer    string    `json:"issuer"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// RoleChange records a single role mutation for audit purposes.
type RoleChange struct {
	ChangedBy   string    `json:"changed_by"`
	TargetEmail string    `json:"target_email"`
	OldRole     string    `json:"old_role"`
	NewRole     string    `json:"new_role"`
	ChangedAt   time.Time `json:"changed_at"`
}

// UserStore abstracts persistent user and role management.
type UserStore interface {
	// UpsertUser inserts or updates a user keyed on (sub, issuer).
	// email is stored for display and backward-compatible lookups.
	UpsertUser(ctx context.Context, sub, issuer, email, name, avatarURL string) error
	// GetUser retrieves a user by email (used by the user management API).
	GetUser(ctx context.Context, email string) (*User, error)
	// GetUserBySub retrieves a user by (sub, issuer). Returns ErrUserNotFound if absent.
	GetUserBySub(ctx context.Context, sub, issuer string) (*User, error)
	ListUsers(ctx context.Context) ([]User, error)
	SetRole(ctx context.Context, email, role string) (oldRole string, err error)
	CountUsers(ctx context.Context) (int, error)
	CountAdmins(ctx context.Context) (int, error)
	InsertRoleChange(ctx context.Context, change RoleChange) error
	ListRoleChanges(ctx context.Context) ([]RoleChange, error)
}
