## Why

Every authenticated user sees all policies, evidence, and audit logs. No scoping, no roles. This works for a single team but breaks when multiple teams share a Studio instance or auditors need read-only access.

Gemara policies already declare RACI contacts — who is responsible, accountable, consulted, and informed. Studio should use these contacts as the authorization model instead of building a separate user management system.

## What Changes

- New `policy_contacts` ClickHouse table populated at policy import time (same pattern as `mapping_entries`).
- `Session` extended with Google Groups membership for identity resolution against RACI contact names.
- Gateway middleware resolves `(email, groups) → Set<(policy_id, raci_role)>` and injects `policy_id IN (...)` filters on all scoped API queries.
- `/auth/me` returns the user's access set so the frontend can conditionally render controls.
- `importPolicyHandler` parses `Policy.contacts` RACI into `policy_contacts` rows after blob insert.
- Retroactive population backfills `policy_contacts` from existing `policies.content` on startup.
- Workbench hides write actions (import, upload, audit triggers) for `consulted` and `informed` roles.

## Capabilities

### New Capabilities

- `policy-contacts-storage`: Parse Policy RACI contacts at import time into a structured `policy_contacts` table. Retroactive population on startup.
- `raci-access-resolution`: Resolve a user's allowed policy set and RACI role from session identity (email + Google Groups) against `policy_contacts`.
- `gateway-policy-filtering`: Inject `policy_id IN (...)` filters on all scoped API endpoints. Enforce write authorization based on RACI role.
- `frontend-role-scoping`: Conditionally render UI controls based on the user's RACI role per policy. Role badges, hidden actions, empty state guidance.

### Modified Capabilities

- `github-oauth`: Extend OAuth flow to request Google Groups membership (OIDC `groups` claim or Directory API fallback). Populate `Session.Groups`.
- `evidence-ingestion`: Enforce RACI role check — only `responsible` and `accountable` users can ingest evidence for a given `policy_id`.

## Impact

| Area | Change |
|:--|:--|
| ClickHouse schema | New `policy_contacts` table |
| `internal/auth` | `Session.Groups`, `ServerSession.Groups`, Groups resolution at login |
| `internal/store` | `PolicyContactStore` interface, RACI parser, `PopulatePolicyContacts` |
| `internal/store/handlers.go` | `importPolicyHandler` calls RACI parser after insert |
| `cmd/gateway/main.go` | Access resolution middleware, retroactive population |
| `/auth/me` | Returns `policies: { "<id>": "<raci_role>" }` access map |
| Workbench | Role-aware rendering in policies-view, chat-assistant |
| Helm chart | `policy_contacts` DDL in clickhouse-schema-configmap |
| `skills/evidence-schema` | New `policy_contacts` schema + access query patterns |
