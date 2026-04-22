## 1. Schema

- [x] 1.1 Add `CREATE TABLE policy_contacts` DDL to `internal/clickhouse/client.go`
- [x] 1.2 Add `policy_contacts` DDL to `charts/complytime-studio/templates/clickhouse-schema-configmap.yaml`
- [x] 1.3 Add `policy_contacts` table schema to `skills/evidence-schema/SKILL.md`

## 2. Store Layer

- [x] 2.1 Define `PolicyContactStore` interface in `internal/store/store.go` with `InsertPolicyContacts`, `ResolveAccess`, `CountPolicyContacts`
- [x] 2.2 Add `PolicyContact` struct to `internal/store/store.go`
- [x] 2.3 Implement `InsertPolicyContacts` (batch insert into `policy_contacts`)
- [x] 2.4 Implement `ResolveAccess(ctx, email string, groups []string) (map[string]string, error)` returning `policy_id → raci_role`
- [x] 2.5 Implement `CountPolicyContacts(ctx, policyID string) (int, error)`

## 3. RACI Parser

- [x] 3.1 Create `internal/store/contacts_parser.go` with `ParsePolicyContacts(content, policyID string) ([]PolicyContact, error)` using `go-gemara` types and `goccy/go-yaml`
- [x] 3.2 Write tests in `internal/store/contacts_parser_test.go` for: valid RACI, empty contacts, invalid YAML, multiple roles

## 4. Import Handler

- [x] 4.1 Modify `importPolicyHandler` in `internal/store/handlers.go` to accept `PolicyContactStore` (update `Stores` struct)
- [x] 4.2 After policy blob insert, call `ParsePolicyContacts` and `InsertPolicyContacts`. Log warnings on failure, don't fail the HTTP response.

## 5. Retroactive Population

- [x] 5.1 Create `PopulatePolicyContacts` in `internal/store/populate.go` — read all policies, parse RACI, insert contacts for policies with zero existing rows
- [x] 5.2 Call `PopulatePolicyContacts` on gateway startup in `cmd/gateway/main.go`

## 6. Session Groups

- [x] 6.1 Add `Groups []string` to `ServerSession` in `internal/auth/session_store.go`
- [x] 6.2 Add `Groups []string` to `Session` in `internal/auth/auth.go`
- [x] 6.3 Extract `groups` claim from Google ID token in `handleCallback` and populate `ServerSession.Groups`
- [x] 6.4 Propagate `Groups` from `ServerSession` to `Session` in `Middleware`

## 7. Access Resolution Middleware

- [x] 7.1 Create `AccessSet` type and `AccessSetFrom(ctx)` helper in `internal/auth/access.go`
- [x] 7.2 Create `AccessMiddleware` that resolves `(email, groups) → AccessSet` via `ResolveAccess` and injects into context
- [x] 7.3 Add graceful degradation: if `policy_contacts` is empty, allow all policies
- [x] 7.4 Wire `AccessMiddleware` into the handler chain in `cmd/gateway/main.go` after auth middleware
- [x] 7.5 Write tests for access resolution: group match, email match, highest role wins, empty table fallback

## 8. Gateway Filtering

- [x] 8.1 Update `listPoliciesHandler` to filter results by `AccessSet`
- [x] 8.2 Update `getPolicyHandler` to return 404 if policy not in `AccessSet`
- [x] 8.3 Update `queryEvidenceHandler` to inject `policy_id IN (...)` filter from `AccessSet`
- [x] 8.4 Update `listAuditLogsHandler` / `getAuditLogHandler` to filter by `AccessSet`
- [x] 8.5 Add write authorization on `ingestEvidenceHandler` and `uploadEvidenceHandler` — require `responsible` or `accountable` role

## 9. /auth/me Access Set

- [x] 9.1 Extend `UserInfo` with `Policies map[string]string` field
- [x] 9.2 Update `handleMe` to resolve access set and include in response

## 10. Frontend Scoping

- [x] 10.1 Extend `fetchConfig` / user info fetch to store the `policies` access map in app state
- [x] 10.2 Add RACI role badge component next to policy name in the policy list
- [x] 10.3 Conditionally hide import/upload buttons for `consulted` and `informed` roles
- [x] 10.4 Hide audit trigger prompts in chat-assistant for `informed` users
- [x] 10.5 Add empty state guidance when no policies are visible

## 11. Skill Update

- [x] 11.1 Add access resolution query patterns to `skills/evidence-schema/SKILL.md`

## 12. Verification

- [ ] 12.1 Seed a policy with RACI contacts, verify `policy_contacts` rows are populated
- [ ] 12.2 Verify `/auth/me` returns the `policies` access map
- [ ] 12.3 Verify `GET /api/policies` returns only accessible policies
- [ ] 12.4 Verify evidence write returns 403 for `consulted` role
- [ ] 12.5 Verify frontend hides write controls for read-only roles
