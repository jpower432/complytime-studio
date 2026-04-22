## 1. Revert raci-authz Backend

- [x] 1.1 Delete `internal/auth/access.go` and `internal/auth/access_test.go`
- [x] 1.2 Delete `internal/gemara/contacts.go` and `internal/gemara/contacts_test.go`
- [x] 1.3 Remove `PolicyContactStore` interface, `InsertPolicyContacts`, `ResolveAccess`, `CountPolicyContacts`, `HasAnyPolicyContacts` from `internal/store/store.go`
- [x] 1.4 Remove RACI parsing call from `importPolicyHandler` in `internal/store/handlers.go`
- [x] 1.5 Remove `PopulatePolicyContacts` from `internal/store/populate.go`
- [x] 1.6 Remove `AccessMiddleware` wiring and `PopulatePolicyContacts` call from `cmd/gateway/main.go`
- [x] 1.7 Remove `policy_contacts` DDL from `internal/clickhouse/client.go`
- [x] 1.8 Remove `policy_contacts` DDL from `charts/complytime-studio/templates/clickhouse-schema-configmap.yaml`
- [x] 1.9 Remove `policy_contacts` table schema and access resolution queries from `skills/evidence-schema/SKILL.md`

## 2. Revert raci-authz Frontend

- [x] 2.1 Remove `authz_active` from `UserInfo` in `workbench/src/api/auth.ts`
- [x] 2.2 Remove authz-inactive banner from `workbench/src/app.tsx`
- [x] 2.3 Remove `.authz-banner` styles from `workbench/src/global.css`
- [x] 2.4 Remove RACI role badges and conditional write-control hiding from `workbench/src/components/policies-view.tsx`

## 3. Revert raci-authz Auth

- [x] 3.1 Remove `AuthzActive` field and access set logic from `handleMe` in `internal/auth/auth.go`
- [x] 3.2 Remove RACI authz-inactive startup warning from `cmd/gateway/main.go`

## 4. Add Admin/Viewer Roles

- [x] 4.1 Add `auth.admins` list to `values.yaml` (default empty = all admin)
- [x] 4.2 Add `ADMIN_EMAILS` env var to gateway deployment template in `charts/complytime-studio/templates/gateway.yaml`
- [x] 4.3 Parse `ADMIN_EMAILS` on startup in `cmd/gateway/main.go`, store as `map[string]bool`
- [x] 4.4 Add `RoleForEmail(email string) string` function to `internal/auth/auth.go` returning `"admin"` or `"viewer"`
- [x] 4.5 Add `role` field to `UserInfo` struct in `internal/auth/auth.go`, populate in `handleMe`
- [x] 4.6 Create `RequireAdmin` middleware in `internal/auth/auth.go` that returns 403 for viewer users
- [x] 4.7 Apply `RequireAdmin` to write endpoints in `cmd/gateway/main.go`
- [x] 4.8 Write tests for `RoleForEmail` and `RequireAdmin` middleware

## 5. Frontend Role Awareness

- [x] 5.1 Add `role: string` to `UserInfo` in `workbench/src/api/auth.ts`
- [x] 5.2 Hide import/upload buttons when `role === "viewer"` in `policies-view.tsx`
- [x] 5.3 Hide evidence upload controls when `role === "viewer"` in `evidence-view.tsx`

## 6. Remove Pins from Conversation Memory

- [x] 6.1 Remove `pinned` field from `ChatMessage` interface in `chat-assistant.tsx`
- [x] 6.2 Delete `PINNED_CACHE_KEY`, `MAX_PINS`, `PIN_TRUNCATE_CHARS` constants
- [x] 6.3 Delete `savePinnedCache`, `loadAndClearPinnedCache` functions
- [x] 6.4 Delete `togglePin` handler and `pinNotice` state
- [x] 6.5 Remove pin toggle button and pinned class from message rendering
- [x] 6.6 Simplify `handleNewSession` — clear messages and null taskIdRef only
- [x] 6.7 Remove `pinnedCache` parameter from `buildInjectedContext`, update callers
- [x] 6.8 Remove `.chat-msg-pinned`, `.chat-pin-btn`, `.pin-notice` styles from `global.css`

## 7. Verification

- [x] 7.1 `go build ./...` passes
- [x] 7.2 `go test ./...` passes
- [x] 7.3 TypeScript `tsc --noEmit` passes
- [ ] 7.4 Deploy to cluster, verify gateway starts without warnings
- [ ] 7.5 Verify `/auth/me` returns `role` field
- [ ] 7.6 Verify viewer gets 403 on `POST /api/policies/import`
- [ ] 7.7 Verify viewer can `GET /api/policies` and chat with agent
