# Simple RBAC — design

## Context

Studio uses Google OAuth for authentication. Authorization is a static `ADMIN_EMAILS` env var checked by `RoleForEmail()` at request time. The `writeProtect` middleware gates all mutating API calls behind `RequireAdmin`. The frontend checks `currentUser.value?.role === "admin"` to conditionally render write actions.

This design replaces the static allowlist with a persistent user store while preserving the existing middleware pattern.

## Decision 1: ClickHouse for user storage

**Choice:** Store users and role changes in ClickHouse using the same migration framework as all other Studio tables.

**Rationale:** Studio already runs ClickHouse for all persistent state. Adding a separate database (SQLite, Postgres) for two small tables violates the "reuse existing infrastructure" principle. The `users` table will have low write volume (one insert per new login, rare role changes). ClickHouse's `ReplacingMergeTree` handles upserts naturally.

**Consequences:** User lookups on every authenticated request add one ClickHouse query. Acceptable — the `users` table will have single-digit to low-hundreds rows. If this becomes a concern, add an in-process cache with short TTL.

## Decision 2: First-user-is-admin via atomic check

**Choice:** On OAuth callback, after fetching/creating the user record, check if the `users` table has exactly one row. If yes and that row is the current user, set role to `admin`.

**Rationale:** Simpler than a separate "setup wizard" or "bootstrap mode." The first person to log in owns the instance. Subsequent users get `reviewer` by default. `ADMIN_EMAILS` can pre-seed admin rows before anyone logs in (disaster recovery / air-gapped deploys).

**Consequences:** In a race condition where two users log in simultaneously on a fresh instance, both could see zero rows and both become admin. Acceptable for the two-role model — the first admin can demote the other. ClickHouse does not support row-level locking, so true atomicity requires application-level sequencing (a mutex in the gateway process). For single-replica deployments this is sufficient.

**Failure modes:** If ClickHouse is down during first login, the callback fails with 500. No silent role escalation.

## Decision 3: User upsert on every OAuth callback

**Choice:** On each successful OAuth callback, upsert into `users` (insert if not exists, update `name` and `avatar_url` if exists). Role is never overwritten by login — only by explicit admin action via the role management API.

**Rationale:** Keeps profile data fresh without requiring a separate "sync" mechanism. The `ReplacingMergeTree` engine with `version` column handles this cleanly.

**Consequences:** The `users` table schema uses `ReplacingMergeTree(version)` with `email` as the ordering key. On read, `FINAL` or deduplication query ensures latest row.

## Decision 4: Immutable role_changes audit table

**Choice:** Every role mutation inserts a row into `role_changes` with `changed_by`, `target_email`, `old_role`, `new_role`, `changed_at`. This table uses `MergeTree` (not Replacing) — rows are never updated or deduplicated.

**Rationale:** The auditor persona needs proof that RBAC is enforced. An append-only log satisfies this. The compliance manager can export it as evidence of access control.

**Consequences:** Table grows monotonically. At the expected volume (handful of role changes per quarter), this is negligible. No TTL applied — access control history should be retained indefinitely.

## Decision 5: Replace RoleForEmail with store lookup

**Choice:** `RoleForEmail` is replaced by `RoleForUser(ctx, email)` which queries the `users` table. The `RequireAdmin` middleware and `handleMe` endpoint both use this new function.

**Rationale:** Direct replacement. Same call sites, same return type (`"admin"` or `"reviewer"`), but backed by persistent state instead of a static map.

**Consequences:** The `Handler` struct no longer needs the `admins map[string]bool` field. `SetAdmins` becomes a bootstrap-only path that pre-seeds the `users` table rather than setting an in-memory map.

## Decision 6: ADMIN_EMAILS as bootstrap seed

**Choice:** If `ADMIN_EMAILS` is set, on startup the gateway inserts those emails into the `users` table with role `admin` (no-op if already present). This runs before any OAuth callbacks.

**Rationale:** Preserves backward compatibility. Existing deployments that set `ADMIN_EMAILS` continue to work. New deployments can omit it and rely on first-user-is-admin.

**Consequences:** `ADMIN_EMAILS` no longer gates runtime authorization. It only seeds initial rows. Once an admin exists in the store, the env var is irrelevant. Document this behavioral change in release notes.

## Decision 7: Frontend role gating pattern

**Choice:** The existing `isAdmin()` checks in `chat-assistant.tsx`, `policies-view.tsx`, and `evidence-view.tsx` continue to work unchanged — they read `currentUser.value?.role === "admin"`. The "reviewer" role (previously "viewer") returns `false` for these checks. Add a "Users" sidebar item visible only to admins.

**Rationale:** Minimal frontend change. The `UserInfo.role` field already flows through. Renaming "viewer" to "reviewer" is a backend string change that propagates automatically.

**Consequences:** Reviewers see all read views but write actions are hidden. Add `title` attributes (tooltips) on disabled/hidden elements explaining why they're restricted.

## Schema

### users table

```sql
CREATE TABLE IF NOT EXISTS users (
    email String,
    name String,
    avatar_url String,
    role LowCardinality(String) DEFAULT 'reviewer',
    created_at DateTime64(3) DEFAULT now64(3),
    version UInt64 DEFAULT toUnixTimestamp64Milli(now64(3))
) ENGINE = ReplacingMergeTree(version)
ORDER BY (email)
```

### role_changes table

```sql
CREATE TABLE IF NOT EXISTS role_changes (
    changed_by String,
    target_email String,
    old_role LowCardinality(String),
    new_role LowCardinality(String),
    changed_at DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree
ORDER BY (changed_at, target_email)
```

## API

| Method | Path | Auth | Body | Response |
|--------|------|------|------|----------|
| GET | `/api/users` | admin | — | `[{email, name, avatar_url, role, created_at}]` |
| PATCH | `/api/users/:email/role` | admin | `{"role": "admin"\|"reviewer"}` | `{email, role, changed_by}` |
| GET | `/api/users/role-changes` | admin | — | `[{changed_by, target_email, old_role, new_role, changed_at}]` |

## Related documents

- `internal/auth/auth.go` — current `RoleForEmail`, `RequireAdmin`, `Handler`
- `internal/clickhouse/client.go` — migration framework, existing tables
- `cmd/gateway/main.go` — `ADMIN_EMAILS` parsing, `writeProtect` wiring
