## 1. Create `internal/gemara/` Package

- [x] 1.1 Create `internal/gemara/` directory with `doc.go` (package comment + SPDX header)
- [x] 1.2 Define `AuditLogSummary` struct in `internal/gemara/auditlog.go`
- [x] 1.3 Implement `ParseAuditLog(content string) (*AuditLogSummary, error)` using `go-gemara` `AuditLog` type and `goccy/go-yaml`
- [x] 1.4 Write unit tests for `ParseAuditLog` in `internal/gemara/auditlog_test.go` (valid YAML, invalid YAML, missing results, mixed classifications)

## 2. Relocate Existing Parsers

- [x] 2.1 Move `PolicyContact` struct and `ParsePolicyContacts` from `internal/store/contacts_parser.go` to `internal/gemara/contacts.go`
- [x] 2.2 Move `MappingEntry` struct and `ParseMappingYAML` (rename to `ParseMappingEntries`) from `internal/store/mapping_parser.go` to `internal/gemara/mappings.go`
- [x] 2.3 Move existing tests from `internal/store/contacts_parser_test.go` to `internal/gemara/contacts_test.go`
- [x] 2.4 Move existing tests from `internal/store/mapping_parser_test.go` to `internal/gemara/mappings_test.go`
- [x] 2.5 Delete old files: `internal/store/contacts_parser.go`, `internal/store/contacts_parser_test.go`, `internal/store/mapping_parser.go`, `internal/store/mapping_parser_test.go`

## 3. Update Store Layer Imports

- [x] 3.1 Update `internal/store/store.go` to use `gemara.PolicyContact` and `gemara.MappingEntry` types from `internal/gemara/`
- [x] 3.2 Update `internal/store/handlers.go` to import parser functions from `internal/gemara/`
- [x] 3.3 Update `internal/store/populate.go` to import `ParsePolicyContacts` from `internal/gemara/`
- [x] 3.4 Verify all existing tests pass after import path changes

## 4. Gateway AuditLog Enrichment

- [x] 4.1 Simplify `createAuditLogHandler` request struct to require only `policy_id` and `content`
- [x] 4.2 Call `gemara.ParseAuditLog(content)` in the handler to extract dates, target, and classification counts
- [x] 4.3 Compute JSON summary string `{"strengths":N,"findings":N,"gaps":N,"observations":N}` from parsed result
- [x] 4.4 Populate `AuditLog` struct fields (`AuditStart`, `AuditEnd`, `Summary`, `Framework`) from parsed result before insert
- [x] 4.5 Return `400 Bad Request` with descriptive error if `ParseAuditLog` fails

## 5. Frontend Simplification

- [x] 5.1 Update `saveAuditLog` in `chat-assistant.tsx` to send only `policy_id` and `content` (remove `audit_start`, `audit_end`, `summary`)
- [x] 5.2 Add error handling to display gateway `400` errors to the user

## 6. Agent Prompt Update

- [x] 6.1 Update `agents/assistant/prompt.md` step 5 to mandate reading `gemara://schema/definitions` resource before authoring AuditLog
- [x] 6.2 Ensure step 5 explicitly states to call `validate_gemara_artifact` MCP tool with `definition: "#AuditLog"` on each artifact
- [x] 6.3 Sync prompt to Helm chart: run `make sync-prompts` or manually copy to `charts/complytime-studio/agents/assistant/prompt.md`

## 7. Verification

- [x] 7.1 Run `go build ./...` to confirm compilation
- [x] 7.2 Run `go test ./internal/gemara/...` to confirm all parser tests pass
- [x] 7.3 Run `go test ./internal/store/...` to confirm store tests pass with updated imports
- [x] 7.4 Run `go vet ./...` to confirm no issues
