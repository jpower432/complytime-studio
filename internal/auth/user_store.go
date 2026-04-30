// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"github.com/complytime/complytime-studio/internal/identity"
)

// Re-export identity types so existing callers within auth (and callers
// referencing auth.User, auth.RoleChange, auth.ErrUserNotFound, auth.UserStore)
// continue to compile without changing import paths.
type User = identity.User
type RoleChange = identity.RoleChange
type UserStore = identity.UserStore

var ErrUserNotFound = identity.ErrUserNotFound
