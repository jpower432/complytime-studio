## Context

Studio authenticates via Google OAuth. The gateway extracts identity from the ID token, stores it in an encrypted session cookie, and grants full access to every authenticated user. There are no roles, permissions, or scoped access.

Gemara `Policy.contacts` already declares a RACI matrix — responsible, accountable, consulted, informed — with contact names and emails. This is an existing, artifact-native ACL that Studio ignores today.

The `mapping_entries` pattern (parse YAML at import → structured ClickHouse rows → SQL joins at query time) proved effective for the impact graph. The same approach applies to RACI contacts.

Key constraints:
- Google OAuth (OIDC) is the only identity provider.
- Google Groups is the group membership system.
- ClickHouse is the only data store.
- `policy_id` is already the foreign key on every scoped table (evidence, mappings, audit_logs).

## Goals / Non-Goals

**Goals:**
- Scope data visibility per policy using the Gemara RACI contacts.
- Resolve identity via email and Google Groups membership.
- Enforce authorization at the gateway for all `/api/*` endpoints.
- Expose the user's access set to the frontend for role-aware rendering.

**Non-Goals:**
- Building a user management UI or admin console.
- Agent session isolation (MCP proxy filtering, ClickHouse row policies). Deferred — tracked in the ADR.
- Per-tenant API tokens. Current `STUDIO_API_TOKEN` bypass remains unrestricted.
- Rate limiting, retention, or OCI registry scoping per policy.

## Decisions

### 1. `policy_contacts` table schema

Store RACI contacts as structured rows, same engine pattern as `mapping_entries`.

```sql
CREATE TABLE IF NOT EXISTS policy_contacts (
  policy_id String,
  raci_role LowCardinality(String),
  contact_name String,
  contact_email String DEFAULT '',
  contact_affiliation String DEFAULT '',
  imported_at DateTime64(3) DEFAULT now64(3)
) ENGINE = ReplacingMergeTree(imported_at)
ORDER BY (policy_id, raci_role, contact_name)
```

`ReplacingMergeTree` deduplicates on `(policy_id, raci_role, contact_name)` so re-imports are idempotent.

**Why not a separate `tenants` table?** `policy_id` already exists as the scoping key on every row. Adding a `tenant_id` concept creates a parallel ownership model that diverges from Gemara.

### 2. Parse RACI in `importPolicyHandler`

After inserting the raw policy blob, parse `Policy.contacts` via `go-gemara` types and batch-insert into `policy_contacts`. Log a warning and continue on parse failure — the raw policy is still stored.

Same defensive approach as `importMappingHandler` → `ParseMappingYAML`.

### 3. Groups resolution at login via OIDC claim

Request `groups` scope during Google OAuth. If the ID token contains the `groups` claim, populate `ServerSession.Groups`. If absent (Workspace config issue), fall back to empty groups and email-only matching.

**Why OIDC claim over Directory API?** Fewer moving parts. No additional API key or admin delegation needed. The `groups` claim requires a one-time Google Workspace admin config but is zero-cost at runtime.

**Why not lazy resolve?** Cold cache on every session start. OIDC claim arrives with the login — no extra round-trip.

### 4. Access resolution middleware

New middleware between auth and store handlers. On each request:

1. Extract `session.Email` and `session.Groups` from context.
2. Query `policy_contacts` for matching rows.
3. Build `AccessSet map[string]string` (policy_id → highest RACI role).
4. Inject `AccessSet` into request context.
5. Store handlers read `AccessSet` to filter queries.

The access query:

```sql
SELECT policy_id,
       argMax(raci_role, CASE raci_role
           WHEN 'accountable' THEN 4
           WHEN 'responsible' THEN 3
           WHEN 'consulted' THEN 2
           WHEN 'informed' THEN 1 ELSE 0 END) AS raci_role
FROM policy_contacts FINAL
WHERE contact_name IN (?, ?, ...)
   OR contact_email = ?
GROUP BY policy_id
```

**Cache strategy:** Resolve once per request. No cross-request caching in Phase 1 — avoids stale access after policy re-import.

### 5. `/auth/me` returns access set

Extend the `UserInfo` response with a `policies` field:

```json
{
  "login": "alice@acme.com",
  "name": "Alice",
  "avatar_url": "...",
  "email": "alice@acme.com",
  "policies": {
    "ampel-bp": "responsible",
    "soc2-corp": "consulted"
  }
}
```

The frontend uses this map to drive all conditional rendering.

### 6. Write authorization on evidence ingestion

`POST /api/evidence` and `POST /api/evidence/upload` require `responsible` or `accountable` RACI role for the target `policy_id`. Return `403 Forbidden` if the user's role is `consulted` or `informed`.

### 7. Frontend role-aware rendering

The workbench reads `policies` from `/auth/me` and conditionally renders:
- Import/upload buttons: hidden for `consulted` and `informed`.
- Audit trigger prompts: hidden for `informed`.
- RACI role badge: displayed next to policy name in the policy list.
- Empty state: "No policies visible. Import a policy or contact a policy owner."

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| Google Workspace free tier may not support `groups` claim | Fall back to email-only matching. Document the Workspace admin config requirement. |
| Per-request access resolution adds latency | Query is lightweight (small cardinality table, indexed by contact_name). Monitor via ClickHouse query log. |
| API token bypass (`STUDIO_API_TOKEN`) sees everything | Acceptable for dev/CI. Scoped tokens are deferred. |
| Agent (ClickHouse MCP) bypasses gateway filtering | Explicitly deferred. Agent scoping is a separate concern. |
| Policy importer not in RACI loses visibility after import | Cataloged as an open question. Defer to ADR. |
| `ReplacingMergeTree` eventual consistency | Use `FINAL` in access queries. Same approach as `mapping_entries`. |

## Migration Plan

1. Add `policy_contacts` DDL to ClickHouse schema configmap and `internal/clickhouse/client.go`.
2. Deploy — table is created, no data yet.
3. `PopulatePolicyContacts` runs on startup, backfilling from existing `policies.content`.
4. `importPolicyHandler` starts writing `policy_contacts` on new imports.
5. Access resolution middleware is added but defaults to "allow all" if `policy_contacts` is empty.
6. Frontend reads access set from `/auth/me` and renders accordingly.
7. Rollback: remove middleware. Data in `policy_contacts` is inert without enforcement.
