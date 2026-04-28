# Tasks — simple RBAC

## Schema migrations

- [x] Add migration (next version after current max) creating `users` table with `ReplacingMergeTree(version)`, columns: `email`, `name`, `avatar_url`, `role` (default `'reviewer'`), `created_at`, `version`. ORDER BY `(email)`.
- [x] Add migration creating `role_changes` table with `MergeTree`, columns: `changed_by`, `target_email`, `old_role`, `new_role`, `changed_at`. ORDER BY `(changed_at, target_email)`.

## User store layer

- [x] Create `internal/auth/user_store.go` with `UserStore` interface: `UpsertUser(ctx, email, name, avatarURL) error`, `GetUser(ctx, email) (*User, error)`, `ListUsers(ctx) ([]User, error)`, `SetRole(ctx, email, role string) (oldRole string, err error)`, `CountUsers(ctx) (int, error)`.
- [x] Create `User` struct: `Email`, `Name`, `AvatarURL`, `Role`, `CreatedAt`.
- [x] Implement ClickHouse-backed `UserStore` in `internal/store/users.go` on existing `*Store`.
- [x] Implement `InsertRoleChange(ctx, changedBy, targetEmail, oldRole, newRole)` on the store.
- [x] Implement `ListRoleChanges(ctx) ([]RoleChange, error)` on the store.

## Auth handler changes

- [x] Add `UserStore` field to `auth.Handler`; accept via `SetUserStore` setter to avoid breaking signature.
- [x] In `handleCallback`: after successful OAuth, call `UpsertUser` with email, name, avatar. If `CountUsers == 0`, call `SetRole(admin)` and insert a role change record with `changed_by = "system"`.
- [x] Update `handleMe` to read role from the user store instead of `RoleForEmail`.
- [x] Update `RequireAdmin` middleware to accept `UserStore` and query role from store.
- [x] Deprecate `SetAdmins` — convert to a bootstrap seed: on startup, if `ADMIN_EMAILS` is set, upsert those emails as admin in the `users` table. Log deprecation warning.

## API endpoints

- [x] `GET /api/users` — authenticated. Query `ListUsers`, return JSON array.
- [x] `PATCH /api/users/:email/role` — admin-only (via writeProtect). Accept `{"role": "admin"|"reviewer"}`. Validate role value. Call `SetRole`, insert `role_changes` row with `changed_by` from session email. Return updated user.
- [x] `GET /api/role-changes` — authenticated. Query `ListRoleChanges`, return JSON array.
- [x] Wire all three endpoints in `cmd/gateway/main.go`. PATCH gated by `RequireAdmin` via `writeProtect`.

## Frontend — role rename

- [x] Backend returns `"reviewer"` instead of `"viewer"` for non-admin users from the store. Static `RoleForEmail` still returns `"viewer"` as fallback when no store is configured.

## Frontend — admin panel

- [x] Add `UsersView` component: table with email, name, role, created_at. Promote/Demote button calls `PATCH /api/users/:email/role`.
- [x] Add role changes audit tab within UsersView: table showing changed_by, target, old_role, new_role, timestamp.
- [x] Add "Users" item to sidebar nav (after Inbox). Only render when `currentUser.value?.role === "admin"`.
- [x] Add `"users"` to the `View` type union in `app.tsx`. Wire routing.

## Frontend — reviewer gating

- [ ] Add `title` tooltip to disabled/hidden write elements explaining "Admin role required" for reviewer users.
- [ ] Verify existing `isAdmin()` checks cover: policy import, evidence upload, draft promotion ("Save to History"), chat "Save to Audit History" button.

## Gateway startup changes

- [x] If `ADMIN_EMAILS` is set, on startup iterate emails and call `UpsertUser` with role `admin`.
- [x] Log deprecation warning: `"ADMIN_EMAILS is set — bootstrapping seed admins into persistent user store (deprecated: use the admin UI)"`.
- [x] If no store and no admins, log: `"ADMIN_EMAILS is empty and no user store — all authenticated users have admin access"`.

## Tests

- [x] Unit test `UserStore` (in-memory): upsert, get, list, set role, count.
- [x] Unit test role change audit: set role → verify `role_changes` row exists with correct actor/target/old/new.
- [x] Handler test `GET /api/users`: returns list.
- [x] Handler test `PATCH /api/users/:email/role`: role update works, invalid role rejected.
- [x] Handler test `handleMe`: returns role sourced from store not env var.
- [ ] Unit test first-user-is-admin logic via `handleCallback` (requires mocking Google OAuth exchange).
- [ ] Unit test `ADMIN_EMAILS` bootstrap integration.
- [ ] Handler test reviewer gets 403 on PATCH (requires full writeProtect wiring in test).
- [ ] Frontend: admin sees "Users" sidebar item, reviewer does not.

## Documentation

- [ ] Update `README.md` "Quick Start" to mention first-user-is-admin behavior.
- [ ] Add deprecation note for `ADMIN_EMAILS` in README and Helm `values.yaml` comments.
- [ ] Update `docs/use-case.md` "Who It Is For" section to reference admin and reviewer roles.

## Edge case: last-admin guard

- [ ] `PATCH /api/users/:email/role` must reject demoting the last remaining admin. Query `SELECT count() FROM users FINAL WHERE role = 'admin'`. If count is 1 and target is that admin, return 409 with `"cannot demote the last admin"`.
