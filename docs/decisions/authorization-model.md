# Authorization Model: RACI-Scoped Multi-Tenancy

**Status:** Exploratory
**Date:** 2026-04-21

## Context

Studio authenticates users via Google OAuth (OIDC). The gateway extracts name, email, and avatar from the ID token and stores them in an encrypted session cookie. There is no concept of roles, permissions, or scoped access. Every authenticated user has full access to all policies, evidence, and agent capabilities.

This is fine for a single-team deployment. It breaks when:

- Auditors need read-only access to evidence and reports.
- Operators need write access to policies but not to system configuration.
- Multiple teams share a Studio instance with different policy scopes.

## Goals

1. Scope data visibility per policy using the Gemara RACI contacts already declared in the artifact.
2. Resolve identity via Google Groups membership.
3. Avoid building a user management system. The Gemara artifact carries its own ACL.

## Design Direction

**The tenant boundary is the policy. The RACI is the ACL. Google Groups is the identity resolution.**

### Layer Visibility

| Gemara Layer | Visibility | Rationale |
|:--|:--|:--|
| L1 GuidanceCatalog | Public | Industry standards, not org-specific |
| L2 ControlCatalog | Public | Control libraries, shared knowledge |
| L3 Policy | RACI-scoped | Org-specific enforcement decisions |
| MappingDocument | Inherits from Policy | Scoped by `policy_id` |
| Evidence | Inherits from Policy | Scoped by `policy_id` |
| MappingEntries | Inherits from Policy | Scoped by `policy_id` |
| AuditLog | Inherits from Policy | Scoped by `policy_id` |

### RACI → Access Mapping

The Gemara `Policy.contacts` RACI declares who interacts with the policy. Studio maps RACI roles to access levels:

| RACI Role | Studio Access |
|:--|:--|
| Responsible | Read + write (import evidence, trigger audits) |
| Accountable | Read + write (manage policy, approve) |
| Consulted | Read + agent interaction (query, analyze) |
| Informed | Read-only (view posture, view audit logs) |

Highest matching role wins when a user matches multiple RACI contacts.

### Identity Resolution

RACI `Contact.name` doubles as a Google Group identifier. Resolution uses two paths, tried in order:

1. **Group match (primary):** `policy_contacts.contact_name` matches one of the user's Google Groups.
2. **Email match (fallback):** `policy_contacts.contact_email` matches the user's session email. For individual overrides.

### Google Groups Resolution Options

| Option | Mechanism | Trade-off |
|:--|:--|:--|
| OIDC groups claim | Groups arrive in ID token at login | Requires Workspace admin config. Claim can be large. |
| Directory API at login | Gateway calls `admin.googleapis.com` after OAuth | Extra API call. Needs Directory API scope + admin delegation. |
| Lazy resolve, cache aggressively | Resolve groups on first policy access, cache for session duration | First access slower. Cold cache on session start. |

### Structured Storage

Parse Policy RACI at import time into a `policy_contacts` table (same pattern as `mapping_entries`):

```
policy_contacts
├── policy_id
├── raci_role        (responsible | accountable | consulted | informed)
├── contact_name     ← Google Group ID or individual name
├── contact_email    ← empty for group contacts, set for individuals
├── contact_affiliation
└── imported_at
```

### Gateway Access Query

```sql
SELECT DISTINCT policy_id,
       max(CASE raci_role
           WHEN 'accountable' THEN 4
           WHEN 'responsible' THEN 3
           WHEN 'consulted' THEN 2
           WHEN 'informed' THEN 1 END) AS access_level
FROM policy_contacts
WHERE contact_name IN ('platform-team', 'engineering-all')
   OR contact_email = 'alice@acme.com'
GROUP BY policy_id
```

Result is a set of `(policy_id, access_level)` pairs cached in the session or resolved per-request. All downstream queries filter by this policy set.

### Self-Service Lifecycle

Importing a policy IS granting access. The artifact carries its own ACL:

1. User imports a policy via `POST /api/policies/import`.
2. Gateway parses RACI contacts → `policy_contacts` rows.
3. User's Google Groups match RACI contacts → policy is visible.
4. Removing access = updating the RACI in the policy and re-importing.

No admin provisioning needed. The policy author decides access.

### Agent Scoping

The assistant talks directly to ClickHouse MCP, bypassing the gateway. Options:

| Option | Mechanism | Trade-off |
|:--|:--|:--|
| MCP proxy filtering | Gateway injects `AND policy_id IN (...)` into every `run_select_query` before forwarding to ClickHouse MCP | Pragmatic. Agent never sees the filter. Requires MCP query rewriting. |
| ClickHouse row policies | `CREATE ROW POLICY` enforced at the database level | Strongest guarantee. Requires per-user or per-session CH credentials. |
| Prompt injection | Instruct agent to only query allowed policies | Fragile. LLM can ignore it. Not a security boundary. |

### What Changes vs. What Doesn't

| Changes | Doesn't Change |
|:--|:--|
| New `policy_contacts` table (parsed at import time) | `evidence` table schema |
| `Session.Groups []string` (populated at login) | `mapping_entries` schema |
| Gateway filter middleware (user → policy_ids → scoped queries) | `audit_logs` schema |
| Agent scoping (MCP proxy or row policies) | No `tenant_id` column needed anywhere |
| | `policy_id` is already the foreign key on every row |

## Rejected Approaches

| Approach | Why Not |
|:--|:--|
| `tenant_id` column on every table | Redundant — `policy_id` already serves as the scoping key. Adding a separate tenant concept duplicates what Gemara RACI already provides. |
| Instance per tenant (Model B) | Simpler isolation but more ops burden. Doesn't leverage the Gemara artifact model. |
| Keycloak / Authzed | Full authorization server is overkill when the Gemara artifact carries its own ACL. |
| Custom user management | Studio is not a user management product. Google Groups owns identity. Gemara RACI owns authorization. |
| Kubernetes RBAC mapping | Couples Studio authz to cluster access. |
| Global RBAC without policy scoping | Doesn't solve multi-team sharing. An "auditor" role with no policy scope sees everything. |

## Implementation Phases

### Phase 1: Policy Contacts + RACI Parsing

Parse `Policy.contacts` RACI at import time into a `policy_contacts` ClickHouse table. Same pattern as `mapping_entries` — parse YAML via go-gemara types, batch insert structured rows, retroactive population on startup.

| Deliverable | Description |
|:--|:--|
| `policy_contacts` table | `(policy_id, raci_role, contact_name, contact_email, contact_affiliation, imported_at)` |
| Import-time parsing | `importPolicyHandler` parses RACI after blob insert |
| Retroactive population | Backfill from existing `policies.content` on startup |
| `Session.Groups` | Populate from Google Groups (OIDC claim or Directory API) at login |

Unblocks Phase 3. No user-facing behavior change yet — data is stored but not enforced.

### Phase 2: Agent Session Isolation

The assistant uses `InMemorySessionService` and `InMemoryTaskStore` shared across all users. User A's conversation context is visible to User B if session/task IDs collide. This is a data leak in multi-tenant.

| Deliverable | Description |
|:--|:--|
| Scoped session IDs | Prefix session and task IDs with a user identity hash derived from the A2A request |
| Per-user agent context | Ensure the agent's ClickHouse queries are scoped to the user's allowed policy set |
| MCP proxy filtering | Gateway injects `AND policy_id IN (...)` into `run_select_query` calls forwarded to ClickHouse MCP |

Independent of Phase 1. Can run in parallel.

### Phase 3: Gateway Query Filtering

Enforce RACI-scoped visibility on all API endpoints. The gateway resolves `user → allowed policy_ids` from `policy_contacts` and injects filters into every query.

| Deliverable | Description |
|:--|:--|
| Access resolution middleware | On each request, resolve `(session.email, session.groups) → Set<(policy_id, raci_role)>` |
| `GET /api/policies` filtering | Return only policies the user has RACI access to |
| `GET /api/evidence` filtering | Inject `policy_id IN (...)` filter |
| `POST /api/evidence` authz | Verify caller has responsible/accountable RACI role for the target `policy_id` |
| `POST /api/policies/import` attribution | Enforce `imported_by` from session |
| `/auth/me` access set | Return `{ policies: { "ampel-bp": "responsible", ... } }` for frontend scoping |

### Phase 4: Frontend Scoping

Conditionally render UI controls based on the user's RACI role per policy. The `/auth/me` access set from Phase 3 drives all frontend decisions.

| Deliverable | Description |
|:--|:--|
| Role-aware policy view | Hide import/upload buttons for `consulted` and `informed` roles |
| Role-aware chat assistant | Hide "Run audit" prompts for `informed` users |
| Role badge | Show RACI role next to policy name in the policy list |
| Empty state guidance | When no policies are visible, prompt user to import or contact a policy owner |

### Deferred

| Concern | Status |
|:--|:--|
| API token scoping (per-tenant tokens) | After Phase 4 |
| Audit trail (`audit_events` table) | After Phase 4 |
| Rate limiting per tenant | Operational maturity |
| Retention per policy | Operational maturity |
| OCI registry scoping | Operational maturity |

## Open Questions

- Should the policy importer get implicit access even if they're not in the RACI? (Useful for importing on behalf of another team. Confusing if they lose visibility after import.)
- Does Google Workspace free tier support the `groups` claim in OIDC tokens, or is Directory API the only path?
- Should the API token bypass (`STUDIO_API_TOKEN`) have unrestricted access to all policies, or should tokens be scoped?
- How should the agent be scoped — MCP proxy rewriting or ClickHouse row policies?
- Should there be a "default viewer" escape hatch for org-wide visibility, or is empty-by-default the right posture?
- How does this interact with the `effective_policy` MCP tool (go-gemara#64)? Should it respect RACI scoping?

## Related

- [Impact Graph](impact-graph.md) — `mapping_entries` parsing pattern reused for `policy_contacts`
- [Backend Architecture](backend-architecture.md) — gateway as the auth/proxy chokepoint
- `internal/auth/auth.go` — current session and middleware implementation
- `charts/complytime-studio/values.yaml` — Helm auth configuration
- [gemaraproj/go-gemara](https://github.com/gemaraproj/go-gemara) — `Policy.Contacts` RACI types
