## Why

The existing raci-authz implementation maps Gemara RACI contacts to read/write permissions across 15+ files. It was never verified, has a contested fail-open design, and conflates a communication model (RACI) with access control. Two roles cover the actual need: **admin** (policy owners, security engineers) and **viewer** (auditors, compliance reviewers). This change reverts raci-authz and replaces it with a simple allowlist-based role system.

## What Changes

- **Revert all raci-authz code**: Remove `AccessMiddleware`, `AccessSet`, `policy_contacts` table, RACI parser integration, frontend role scoping, and the authz-inactive banner.
- **Add admin/viewer role enforcement**: Admin emails configured in `values.yaml`. Anyone authenticated but not listed is a viewer. Admins can import, upload, delete, and modify. Viewers can read everything and chat with the agent.
- **Remove pin-based conversation memory**: Pins are consumable and confusing — they inject once into context then vanish. Sticky notes and checkpoints remain.
- **Future direction (not in scope)**: First-user-is-admin pattern with a `users` table for self-service role management. Captured in design.md for later.

## Capabilities

### New Capabilities
- `role-gate`: Admin/viewer role resolution from allowlist and write-endpoint protection

### Modified Capabilities
- `github-oauth`: `/auth/me` response includes `role` field; remove `authz_active` and `policies` access map
- `pinned-messages`: **REMOVED** — delete pin UI, pin storage, pin injection from context assembly
- `context-assembly`: Remove pinned-cache from context injection (sticky notes + checkpoints only)

## Impact

- **Backend**: `internal/auth/access.go` deleted, `internal/auth/auth.go` simplified, `internal/gemara/contacts.go` deleted, `internal/store/store.go` loses `PolicyContactStore`, `internal/store/handlers.go` loses RACI parsing in import handler, `internal/store/populate.go` loses `PopulatePolicyContacts`, `cmd/gateway/main.go` loses access middleware wiring
- **Frontend**: `workbench/src/app.tsx` loses authz banner, `workbench/src/components/policies-view.tsx` loses role badges, `workbench/src/components/chat-assistant.tsx` loses pin UI and pin storage
- **Helm**: `values.yaml` gets `auth.admins` list, `policy_contacts` DDL removed from schema configmap
- **ClickHouse**: `policy_contacts` table orphaned (no migration needed, just stop creating it)
