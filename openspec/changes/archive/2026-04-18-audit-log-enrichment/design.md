## Context

The assistant produces `AuditLog` YAML per the Gemara `#AuditLog` schema. The current pipeline has three gaps:

1. **Agent skips validation** — the prompt says to validate, but the agent has been producing Kubernetes CRD-style YAML (`apiVersion`, `kind`, `spec`) instead of the `#AuditLog` schema structure. It does not read the schema resource or call `validate_gemara_artifact`.
2. **Frontend overwrites metadata** — `saveAuditLog` hardcodes `audit_start`/`audit_end` to `new Date()` and sends `{"saved_from":"chat"}` as the summary, discarding the actual audit period and classification data.
3. **Gateway stores blindly** — `createAuditLogHandler` accepts pre-formed JSON and inserts it directly. No parsing of the `content` YAML occurs.

Existing parsers (`ParsePolicyContacts`, `ParseMappingYAML`) live in `internal/store/`, tightly coupled to store types. The user requires Gemara parsing in a dedicated package.

## Goals / Non-Goals

**Goals:**
- Consolidate all Gemara YAML parsing into `internal/gemara/` using `go-gemara` + `goccy/go-yaml`
- Gateway enriches `AuditLog` on ingest: extract dates, compute classification summary, extract target/framework
- Frontend simplified to send `policy_id` + `content` only
- Agent prompt explicitly requires reading `gemara://schema/definitions` and calling `validate_gemara_artifact` MCP tool before returning AuditLog artifacts

**Non-Goals:**
- Changing the ClickHouse `audit_logs` table schema
- Modifying the `#AuditLog` CUE schema itself
- Adding new frontend views or UI components

## Decisions

### 1. Dedicated `internal/gemara/` package

Move all Gemara YAML parsing out of `internal/store/`. The package exports pure functions that accept YAML content strings and return typed results. No database or HTTP dependencies.

| Function | Input | Output |
|:--|:--|:--|
| `ParseAuditLog(content)` | AuditLog YAML | `AuditLogSummary` (dates, target, framework, classification counts) |
| `ParsePolicyContacts(content, policyID)` | Policy YAML | `[]PolicyContact` |
| `ParseMappingEntries(content, mappingID, policyID, framework)` | MappingDocument YAML | `[]MappingEntry` |

**Rationale**: Separation of concerns. Parsing is domain logic, not storage logic. A dedicated package enables reuse without importing the store.

**Alternative considered**: Keep parsers in `internal/store/` and add `ParseAuditLog` there. Rejected because the user explicitly requested a dedicated package and the parsers have no store dependency.

### 2. Gateway-side enrichment in `createAuditLogHandler`

The handler calls `gemara.ParseAuditLog(content)` and uses the returned summary to populate `audit_start`, `audit_end`, `summary`, `framework`, and the target ID. The frontend no longer sends these fields.

**Rationale**: Follows the existing pattern where `importPolicyHandler` parses RACI contacts and `importMappingHandler` parses mapping entries. Server-side enrichment is the single source of truth. The frontend cannot produce incorrect metadata.

**Alternative considered**: Parse on the frontend before sending. Rejected because JS-side YAML parsing adds a dependency and duplicates logic.

### 3. `AuditLogSummary` struct

```go
type AuditLogSummary struct {
    AuditStart   time.Time
    AuditEnd     time.Time
    TargetID     string
    Framework    string
    Strengths    int
    Findings     int
    Gaps         int
    Observations int
}
```

The `summary` column stores `{"strengths":N,"findings":N,"gaps":N,"observations":N}` — matching what `PostureCard` and `AuditHistoryView` already expect from `JSON.parse(summary)`.

### 4. Agent prompt: mandatory schema read + MCP validation

The prompt step 5 is strengthened to:
1. Read `gemara://schema/definitions` to get the `#AuditLog` definition before authoring
2. Call `validate_gemara_artifact` with `definition: "#AuditLog"` on each generated artifact
3. Fix and re-validate (max 3 attempts)

This makes the schema read an explicit prerequisite, not just the validation call.

### 5. Graceful fallback on parse failure

If `ParseAuditLog` fails (malformed YAML, missing fields), the handler returns `400 Bad Request` with an error message indicating which field is missing or invalid. The frontend displays the error to the user.

**Rationale**: Fail fast. If the agent produced invalid YAML, the user should know immediately rather than storing garbage data.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| `go-gemara` `AuditLog` type may not have `AuditResult.Type` exposed | Confirmed: `ResultType` enum with `ResultStrength`, `ResultFinding`, `ResultGap`, `ResultObservation` exists in `go-gemara` enums |
| Moving parsers changes import paths across multiple files | Mechanical refactor — update imports in `handlers.go`, `populate.go`, and test files |
| Agent may still skip validation despite prompt changes | Gateway rejects invalid YAML server-side, so bad data never reaches the database regardless of agent behavior |
| Existing stored audit logs have `{"saved_from":"chat"}` summaries | No migration needed — old data remains as-is. New logs get correct summaries. Users can re-run audits to get correct data. |

## Open Questions

None — all decisions are grounded in confirmed `go-gemara` types and existing codebase patterns.
