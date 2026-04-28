## Why

Studio's authorization model is a static `ADMIN_EMAILS` environment variable parsed at startup. Admins are a hardcoded allowlist; everyone else is "viewer." This creates three problems:

1. **Bootstrap friction** — deploying Studio requires knowing admin emails before anyone has logged in. There is no self-service path for the first user.
2. **No runtime management** — adding or removing an admin requires restarting the gateway with an updated env var. There is no UI, no API, no audit trail.
3. **Role naming mismatch** — the "viewer" role does not communicate what the persona actually does. The primary read-only user is a compliance reviewer or external auditor, not a passive viewer.

The QE validation review identified that the auditor persona has no dedicated experience, and the compliance manager workflow lacks role-aware gating. Both depend on a real RBAC foundation.

## What Changes

- **Two roles: `admin` and `reviewer`** — admin has full access (import, promote, upload, manage users). Reviewer is read-only (view posture, evidence, history, exports).
- **First-user-is-admin** — the first user to complete OAuth login is automatically assigned `admin`. All subsequent users default to `reviewer`.
- **Persistent user store** — a `users` table in ClickHouse tracks email, display name, role, and signup timestamp.
- **Role change audit log** — a `role_changes` table records who changed whose role and when. Immutable append-only.
- **Admin API** — `GET /api/users` (list), `PATCH /api/users/:email/role` (promote/demote). Admin-only.
- **Admin UI** — "Users" sidebar item (admin-only) showing a user table with role toggles.
- **`ADMIN_EMAILS` deprecated** — kept as optional bootstrap override for disaster recovery (pre-seed admins before first login). Runtime store takes precedence once populated.
- **Frontend role gating** — reviewer sees all read views. Write actions (import, promote, upload, "Save to History") hidden or disabled with explanation.

## Capabilities

### New Capabilities
- `user-store`: Persistent user records in ClickHouse with email, name, role, created_at
- `first-user-admin`: Automatic admin assignment on first OAuth callback when zero users exist
- `role-management-api`: Admin-only endpoints to list users and change roles
- `role-change-audit`: Immutable `role_changes` table recording all role mutations with actor, target, old/new role, timestamp
- `admin-panel-ui`: Sidebar "Users" view for admins to manage roles

### Modified Capabilities
- `auth-middleware`: `RoleForEmail` replaced by store lookup; `RequireAdmin` reads from persistent store
- `write-protect`: Unchanged behavior, but role source shifts from env var to database
- `workbench-role-gating`: Frontend hides write actions for reviewer role with explanatory tooltips

## Impact

- **ClickHouse**: Two new tables (`users`, `role_changes`), additive migrations
- **Gateway**: New user upsert on OAuth callback, new `/api/users` endpoints, `RoleForEmail` replaced
- **Workbench**: "Users" sidebar item for admins, role-aware gating on write actions, reviewer tooltips
- **Helm**: `ADMIN_EMAILS` env var becomes optional bootstrap-only; new migration version
- **Auth tests**: Updated to test store-based roles, first-user logic, role change endpoints

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS

Role management is self-contained within Studio. No external RBAC service dependency. The user store is a ClickHouse table following the same migration pattern as all other Studio state.

### II. Composability First

**Assessment**: PASS

`ADMIN_EMAILS` remains as a bootstrap fallback. Deployments can seed admins via env var before first login. Once the store is populated, runtime management takes over. No breaking change to existing deployments.

### III. Observable Quality

**Assessment**: PASS

Every role change is immutably logged with actor, target, old role, new role, and timestamp. The auditor persona can verify RBAC enforcement through the `role_changes` table.

### IV. Testability

**Assessment**: PASS

First-user-is-admin, role promotion, role demotion, and audit logging are all testable with in-memory ClickHouse store and the existing auth test harness.
