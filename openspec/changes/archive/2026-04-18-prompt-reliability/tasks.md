## 1. Few-Shot Examples

- [x] 1.1 Create `agents/assistant/prompts/few-shot/result-type.yaml` with 6 examples: Strength (clean pass), Strength (remediated finding), Finding (failed eval), Finding (cadence gap), Gap (no evidence), Observation (mixed results)
- [x] 1.2 Create `agents/assistant/prompts/few-shot/satisfaction.yaml` with 5 examples: Satisfied (complete cadence, high confidence), Partially Satisfied (missing cycles), Not Satisfied (failed eval with no remediation), Not Satisfied (critical cadence gaps), Not Applicable (scope exclusion)
- [x] 1.3 Create `agents/assistant/prompts/few-shot/coverage-status.yaml` with 5 examples: Covered (strength+high mapping), Partially Covered (strength+medium mapping), Weakly Covered (low strength), Not Covered (finding), Unmapped (no mapping entry)
- [x] 1.4 Include at least 1 anti-pattern per file (e.g., active exception converting Finding→Strength, remediation_status=Success within audit window, high mapping strength with Gap evidence)

## 2. Prompt Loading

- [x] 2.1 Add `load_few_shot_examples()` function to `main.py` — reads YAML files from `/app/prompts/few-shot/`, formats into a `## Classification Examples` text block
- [x] 2.2 Extend `load_prompt()` to append the few-shot block after skills
- [x] 2.3 Update `Dockerfile` to `COPY prompts/ /app/prompts/`
- [x] 2.4 Update Helm ConfigMap or volume mount to include `prompts/few-shot/` content

## 3. AuditLog Provenance

- [x] 3.1 Add two columns to `audit_logs` DDL in `internal/clickhouse/client.go`: `model Nullable(String)`, `prompt_version Nullable(String)`
- [x] 3.2 Extend `AuditLog` struct in `internal/store/store.go` with `Model` and `PromptVersion` fields
- [x] 3.3 Update `InsertAuditLog` INSERT statement to include `model` and `prompt_version`
- [x] 3.4 Update `ListAuditLogs` and `GetAuditLog` SELECT queries to include new columns
- [x] 3.5 Update `createAuditLogHandler` to accept optional `model` and `prompt_version` fields in the request body
- [x] 3.6 Compute `prompt_version` (SHA256 of merged instruction string) once at startup in `main.py`, store as module-level constant
- [x] 3.7 Extend `after_agent_callback` to attach `model` and `prompt_version` to the save request when AuditLog YAML is detected
- [x] 3.8 Update `skills/evidence-schema/SKILL.md` with new `audit_logs` columns

## 4. Verification

- [x] 4.1 Unit test: `load_few_shot_examples()` returns formatted text containing all examples from `few-shot/` directory
- [x] 4.2 Unit test: `load_prompt()` output includes `## Classification Examples` section
- [x] 4.3 `go test ./internal/store/...` passes with updated `AuditLog` struct and INSERT
- [x] 4.4 `go build ./cmd/gateway/` compiles cleanly
- [x] 4.5 Helm template renders with updated ConfigMap content
- [x] 4.6 End-to-end: run demo audit prompt, verify `audit_logs` row has `model` and `prompt_version` populated
