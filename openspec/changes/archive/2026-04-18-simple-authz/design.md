## Context

The current raci-authz implementation (~500 lines across 15+ files) maps Gemara RACI contacts to access control. It was never verified, uses a fail-open design, and conflates communication roles with permissions. The actual need is simpler: admins write, auditors read.

Separately, the conversation-memory feature includes "pins" — a consumable bookmark mechanism that injects into context once then disappears. Pins duplicate checkpoint behavior and confuse users. Sticky notes and checkpoints remain.

## Goals / Non-Goals

**Goals:**
- Replace raci-authz with admin/viewer roles using a `values.yaml` email allowlist
- Remove all raci-authz code (full revert)
- Remove pin UI and pin storage from conversation memory
- Keep authorization simple enough to verify by inspection

**Non-Goals:**
- Per-policy access scoping (future, with proper design)
- Self-service role management via UI (future: first-user-is-admin pattern)
- Users table or database-backed role storage
- Modifying sticky notes or checkpoints

## Decisions

### 1. Role resolution via email allowlist

Admins are listed in `values.yaml` under `auth.admins`. The gateway reads this as an environment variable (`ADMIN_EMAILS`, comma-separated). Any authenticated user whose email is in the list gets `admin` role. Everyone else gets `viewer`.

**Alternatives considered:**
- Database-backed roles: requires a users table, migration, admin UI. Overkill for current scale.
- Group-based (Google org): ties authorization to identity provider structure. Not portable.

### 2. Write protection at the route level

Instead of per-handler role checks (the raci-authz approach), protect write operations at the route level with a single `RequireAdmin` middleware applied to mutating endpoints.

```
RequireAdmin applied to:
  POST /api/policies/import
  POST /api/evidence
  POST /api/evidence/upload
  POST /api/audit-logs
  POST /api/catalogs/import
  POST /api/mappings/import
  POST /api/publish

NOT applied to (viewer-accessible):
  GET  /api/*  (all reads)
  POST /api/a2a/*  (agent chat)
  GET  /auth/me
```

**Alternatives considered:**
- Per-handler checks: granular but error-prone (one missed check = bypass). The raci-authz approach had this problem.
- Blanket POST protection: too broad — would block agent chat (`POST /api/a2a/*`).

### 3. /auth/me includes role

The `/auth/me` response adds a `role` field (`"admin"` or `"viewer"`). The frontend uses this to hide/show write controls globally (import buttons, upload buttons, audit triggers). Replaces the per-policy `policies` access map and `authz_active` flag.

### 4. Full raci-authz removal

Delete all raci-authz artifacts. No gradual deprecation.

| File | Action |
|:--|:--|
| `internal/auth/access.go` | Delete |
| `internal/auth/access_test.go` | Delete |
| `internal/gemara/contacts.go` | Delete |
| `internal/gemara/contacts_test.go` | Delete |
| `internal/auth/auth.go` | Remove `AuthzActive`, access set from `handleMe` |
| `internal/store/store.go` | Remove `PolicyContactStore` interface and impl |
| `internal/store/handlers.go` | Remove RACI parsing from import handler |
| `internal/store/populate.go` | Remove `PopulatePolicyContacts` |
| `cmd/gateway/main.go` | Remove `AccessMiddleware` wiring, contacts populate |
| `internal/clickhouse/client.go` | Remove `policy_contacts` DDL |
| Helm configmap | Remove `policy_contacts` DDL |
| `workbench/src/app.tsx` | Remove authz banner |
| `workbench/src/components/policies-view.tsx` | Remove role badges, conditional UI |
| `workbench/src/api/auth.ts` | Remove `authz_active`, add `role` |
| `skills/evidence-schema/SKILL.md` | Remove access resolution queries, `policy_contacts` schema |

### 5. Pin removal from conversation memory

Delete pin toggle UI, `savePinnedCache`/`loadAndClearPinnedCache` helpers, `pinned` field from `ChatMessage`, pin notice state, and pin references in `buildInjectedContext`. The "New Session" handler no longer saves pins — it just clears messages and nulls the task ID.

## Risks / Trade-offs

- **[No per-policy scoping]** → Acceptable for current single-team usage. Revisit when multi-team need is validated. Captured as future direction.
- **[Redeploy to change admins]** → Mitigated by ConfigMap hot-reload in future iteration. For now, `helm upgrade` is the change mechanism.
- **[Stale `policy_contacts` table in ClickHouse]** → Orphaned but harmless. No migration needed. Can be dropped manually if desired.

## Future Direction: First-User-Is-Admin

When multi-team usage is validated, the authorization model should evolve to:

1. `users` table in ClickHouse (email, role, created_at)
2. First OAuth login auto-becomes admin
3. Admin can promote/demote users via UI settings panel
4. `values.yaml` allowlist becomes a bootstrap seed, not the source of truth

This requires a settings/admin UI that doesn't exist yet. Don't build it until the need is real.
