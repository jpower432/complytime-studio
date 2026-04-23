## 1. Consolidated Skill

- [ ] 1.1 Create `skills/studio-audit/SKILL.md` with frontmatter (`name: studio-audit`)
- [ ] 1.2 Write classification criteria section (Strength/Finding/Gap/Observation table from audit-methodology)
- [ ] 1.3 Write satisfaction determination section (Satisfied/Partially/Not Satisfied/Not Applicable)
- [ ] 1.4 Write coverage-mapping rules section (join logic, status derivation table, multi-mapping resolution)
- [ ] 1.5 Write compact table reference — one line per table: `table_name: col1, col2, ...` for all ClickHouse tables
- [ ] 1.6 Write MCP tools reference (validate_gemara_artifact params, migrate_gemara_artifact params) — compact, no workflow prose
- [ ] 1.7 Verify total skill size is under 4,000 chars

## 2. Prompt Rewrite

- [ ] 2.1 Rewrite `agents/assistant/prompt.md` workflow section — remove MappingDocument from user inputs, add auto-query step
- [ ] 2.2 Inline AuditLog YAML template under "Author AuditLog" step with `reference-id` annotation
- [ ] 2.3 Add `DESCRIBE TABLE` hint for schema discovery
- [ ] 2.4 Remove references to loading skills or schema resources (agent no longer needs to do this manually)
- [ ] 2.5 Verify prompt.md is under 5,000 chars

## 3. Resource Preloading

- [ ] 3.1 Modify `_fetch_gemara_resources()` in `agents/assistant/main.py` to fetch only `gemara://lexicon`, skipping `gemara://schema/definitions`
- [ ] 3.2 Verify lexicon still appears in startup logs

## 4. Skill Cleanup

- [ ] 4.1 Delete `skills/audit-methodology/` directory
- [ ] 4.2 Delete `skills/evidence-schema/` directory
- [ ] 4.3 Delete `skills/coverage-mapping/` directory
- [ ] 4.4 Delete `skills/gemara-mcp/` directory

## 5. Verification

- [ ] 5.1 Run `make sync-skills` and verify `agents/assistant/skills/studio-audit/SKILL.md` exists
- [ ] 5.2 Run `make deploy` and verify assistant pod starts
- [ ] 5.3 Check assistant logs: `prompt_version` changed, lexicon preloaded, no schema/definitions preloaded
- [ ] 5.4 Verify total system prompt under 15K chars (log or exec into pod to measure)
- [ ] 5.5 Validate a sample AuditLog YAML against `#AuditLog` schema via `gemara-mcp-validate_gemara_artifact`
