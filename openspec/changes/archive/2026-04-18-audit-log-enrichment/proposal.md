## Why

The assistant generates `AuditLog` artifacts that do not conform to the Gemara `#AuditLog` CUE schema. The frontend `saveAuditLog` function hardcodes `audit_start`/`audit_end` to `new Date()` and sends `{"saved_from":"chat"}` as the summary. This means the Posture View and Audit History View display incorrect dates and zero counts for strengths/findings/gaps. The gateway blindly stores whatever JSON it receives without parsing the `content` YAML.

Additionally, all Gemara YAML parsing logic (`contacts_parser.go`, `mapping_parser.go`) lives in `internal/store/`. A dedicated `internal/gemara/` package consolidates parsing, enables reuse across handlers, and aligns with the user's request for separation of concerns.

## What Changes

- Create `internal/gemara/` package to centralize all Gemara YAML parsing using `go-gemara` and `goccy/go-yaml`
- Move existing parsers (`ParsePolicyContacts`, `ParseMappingYAML`) from `internal/store/` to `internal/gemara/`
- Add `ParseAuditLog` function that parses `#AuditLog` YAML and extracts dates, target, and classification counts
- Modify `createAuditLogHandler` to parse `content` server-side: extract `audit_start`/`audit_end` from the YAML, compute a structured summary `{"strengths":N,"findings":N,"gaps":N,"observations":N}`, and extract `target.id` + `framework` metadata
- Simplify frontend `saveAuditLog` to only send `policy_id` and `content` — the gateway derives everything else
- Update the assistant agent prompt to mandate reading `gemara://schema/definitions` for the `#AuditLog` schema and calling the `validate_gemara_artifact` MCP tool before returning any `AuditLog` artifact to the user

## Capabilities

### New Capabilities
- `gemara-parsing`: Dedicated Go package (`internal/gemara/`) for schema-aware Gemara YAML parsing using `go-gemara` types
- `audit-log-gateway-enrichment`: Server-side parsing and enrichment of AuditLog content in `createAuditLogHandler`

### Modified Capabilities
- `streaming-chat`: Frontend `saveAuditLog` simplified to send only `policy_id` + `content`
- `agent-spec-skills`: Agent prompt updated to require `gemara://schema/definitions` read and `validate_gemara_artifact` MCP tool call before returning AuditLog artifacts

## Impact

- **Backend**: `internal/gemara/` new package; `internal/store/` loses parser files (re-exports or import path updates); `internal/store/handlers.go` modified
- **Frontend**: `chat-assistant.tsx` `saveAuditLog` simplified
- **Agent**: `agents/studio-gap-analyst/prompt.md` updated to enforce validation
- **Existing parsers**: Import paths change from `internal/store` to `internal/gemara` in `handlers.go` and `populate.go`
