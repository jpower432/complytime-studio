// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/complytime/complytime-studio/internal/auth"
	"github.com/complytime/complytime-studio/internal/consts"
)

// UpsertUser inserts a new user or updates name/avatar for an existing one.
// Preserves the existing role to avoid ReplacingMergeTree overwriting it
// with the DDL default on every login.
func (s *Store) UpsertUser(ctx context.Context, email, name, avatarURL string) error {
	existing, err := s.GetUser(ctx, email)
	switch {
	case err == nil:
		return s.conn.Exec(ctx, `
			INSERT INTO users (email, name, avatar_url, role)
			VALUES (?, ?, ?, ?)`,
			email, name, avatarURL, existing.Role,
		)
	case errors.Is(err, auth.ErrUserNotFound):
		return s.conn.Exec(ctx, `
			INSERT INTO users (email, name, avatar_url)
			VALUES (?, ?, ?)`,
			email, name, avatarURL,
		)
	default:
		return fmt.Errorf("upsert user %s: %w", email, err)
	}
}

// GetUser retrieves a user by email. Returns auth.ErrSessionNotFound if absent.
func (s *Store) GetUser(ctx context.Context, email string) (*auth.User, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT email, name, avatar_url, role, created_at
		FROM users FINAL
		WHERE email = ?`, email)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", email, err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, fmt.Errorf("get user %s: %w", email, auth.ErrUserNotFound)
	}
	var u auth.User
	if err := rows.Scan(&u.Email, &u.Name, &u.AvatarURL, &u.Role, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("get user %s: %w", email, err)
	}
	return &u, nil
}

// ListUsers returns all registered users.
func (s *Store) ListUsers(ctx context.Context) ([]auth.User, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT email, name, avatar_url, role, created_at
		FROM users FINAL
		ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []auth.User
	for rows.Next() {
		var u auth.User
		if err := rows.Scan(&u.Email, &u.Name, &u.AvatarURL, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// SetRole updates a user's role and returns the previous role.
func (s *Store) SetRole(ctx context.Context, email, role string) (string, error) {
	u, err := s.GetUser(ctx, email)
	if err != nil {
		return "", fmt.Errorf("set role: user %s not found: %w", email, err)
	}
	oldRole := u.Role

	if err := s.conn.Exec(ctx, `
		INSERT INTO users (email, name, avatar_url, role)
		VALUES (?, ?, ?, ?)`,
		u.Email, u.Name, u.AvatarURL, role,
	); err != nil {
		return "", fmt.Errorf("set role for %s: %w", email, err)
	}
	return oldRole, nil
}

// CountUsers returns the number of distinct users.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	row := s.conn.QueryRow(ctx, `SELECT count() FROM users FINAL`)
	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return int(count), nil
}

// CountAdmins returns the number of users with the admin role.
func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	row := s.conn.QueryRow(ctx, `SELECT count() FROM users FINAL WHERE role = ?`, consts.RoleAdmin)
	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return int(count), nil
}

// InsertRoleChange records an immutable role change audit entry.
func (s *Store) InsertRoleChange(ctx context.Context, change auth.RoleChange) error {
	return s.conn.Exec(ctx, `
		INSERT INTO role_changes (changed_by, target_email, old_role, new_role, changed_at)
		VALUES (?, ?, ?, ?, ?)`,
		change.ChangedBy, change.TargetEmail, change.OldRole, change.NewRole, time.Now(),
	)
}

// ListRoleChanges returns all role change audit entries.
func (s *Store) ListRoleChanges(ctx context.Context) ([]auth.RoleChange, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT changed_by, target_email, old_role, new_role, changed_at
		FROM role_changes
		ORDER BY changed_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list role changes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var changes []auth.RoleChange
	for rows.Next() {
		var c auth.RoleChange
		if err := rows.Scan(&c.ChangedBy, &c.TargetEmail, &c.OldRole, &c.NewRole, &c.ChangedAt); err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, rows.Err()
}
