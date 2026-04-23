## 1. Consolidated Skill

- [x] 1.1 Create `skills/studio-audit/SKILL.md` with frontmatter (`name: studio-audit`)
- [x] 1.2 Write classification criteria section (Strength/Finding/Gap/Observation table from audit-methodology)
- [x] 1.3 Write satisfaction determination section (Satisfied/Partially/Not Satisfied/Not Applicable)
- [x] 1.4 Write coverage-mapping rules section (join logic, status derivation table, multi-mapping resolution)
- [x] 1.5 Write compact table reference — one line per table: `table_name: col1, col2, ...` for all ClickHouse tables
- [x] 1.6 Write MCP tools reference (validate_gemara_artifact params, migrate_gemara_artifact params) — compact, no workflow prose
- [x] 1.7 Verify total skill size is under 4,000 chars

## 2. Prompt Rewrite

- [x] 2.1 Rewrite `agents/assistant/prompt.md` workflow section — remove MappingDocument from user inputs, add auto-query step
- [x] 2.2 Inline AuditLog YAML template under "Author AuditLog" step with `reference-id` annotation
- [x] 2.3 Add `DESCRIBE TABLE` hint for schema discovery
- [x] 2.4 Remove references to loading skills or schema resources (agent no longer needs to do this manually)
- [x] 2.5 Verify prompt.md is under 5,000 chars

## 3. Resource Preloading

- [x] 3.1 Modify `_fetch_gemara_resources()` in `agents/assistant/main.py` to fetch only `gemara://lexicon`, skipping `gemara://schema/definitions`
- [x] 3.2 Verify lexicon still appears in startup logs

## 4. Skill Cleanup

- [x] 4.1 Delete `skills/audit-methodology/` directory
- [x] 4.2 Delete `skills/evidence-schema/` directory
- [x] 4.3 Delete `skills/coverage-mapping/` directory
- [x] 4.4 Delete `skills/gemara-mcp/` directory

## 5. Verification

- [x] 5.1 Run `make sync-skills` and verify `agents/assistant/skills/studio-audit/SKILL.md` exists
- [x] 5.2 Run `make deploy` and verify assistant pod starts
- [x] 5.3 Check assistant logs: `prompt_version` changed, lexicon preloaded, no schema/definitions preloaded
- [x] 5.4 Verify total system prompt under 15K chars (log or exec into pod to measure)
- [x] 5.5 Validate a sample AuditLog YAML against `#AuditLog` schema via `gemara-mcp-validate_gemara_artifact`
