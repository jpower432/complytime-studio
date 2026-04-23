## Context

The assistant agent uses a BYO ADK deployment (`agents/assistant/main.py`) with a system prompt assembled from:

1. `prompt.md` — workflow instructions (~3.6K)
2. `skills/*/SKILL.md` — 4 skill files loaded via `load_skills()` (~25K)
3. `gemara://schema/definitions` — CUE schema preloaded via `_fetch_gemara_resources()` (~44K)
4. `gemara://lexicon` — domain vocabulary preloaded (~6K)
5. `prompts/few-shot/*.yaml` — classification examples (~2K)

Total: ~80K chars. The LLM copies template SQL literally, confuses similar CUE types, and misses critical rules buried in noise.

The ClickHouse MCP server already exposes `list_tables` and `run_select_query` tools. The agent can discover table structure at query time rather than carrying it in prompt.

## Goals / Non-Goals

**Goals:**
- Reduce system prompt to under 15K chars
- Eliminate literal-template-copying errors (no SQL templates in prompt)
- Ensure the AuditLog structure is unambiguous (concrete YAML template)
- Preserve domain vocabulary (lexicon) and classification examples (few-shot)
- Maintain the agent's ability to query all existing ClickHouse tables

**Non-Goals:**
- Changing the agent's workflow steps (load policy → discover targets → assess → author)
- Modifying ClickHouse schema or tables
- Changing the MCP server implementations
- Adding new MCP tools (the existing `run_select_query`, `list_tables`, `validate_gemara_artifact` are sufficient)

## Decisions

### D1: Consolidate four skills into one

**Choice:** Merge `audit-methodology`, `evidence-schema`, `coverage-mapping`, and `gemara-mcp` into a single `skills/studio-audit/SKILL.md`.

**Why:** Four skill files create redundancy (table schemas documented alongside query patterns alongside validation workflow). A single file under 4K chars forces prioritization of what the LLM actually needs.

**Alternative:** Keep four skills but trim each. Rejected because the boundaries between them are arbitrary — the agent needs all four simultaneously during a single workflow run.

### D2: Remove SQL query templates entirely

**Choice:** Remove all `SELECT ... FROM ... WHERE ...` patterns from the skill. Provide only table names and column names in a compact list.

**Why:** The agent is a frontier LLM — it knows SQL. The templates caused literal-copying bugs (`{policy_id}`, `<POLICY_ID>` sent verbatim). Table metadata is sufficient for correct query construction.

**Alternative:** Fix placeholder syntax (tried `<PLACEHOLDER>`, still copied literally). The templates are the problem, not the syntax.

### D3: Drop CUE schema preload, keep lexicon

**Choice:** Remove `gemara://schema/definitions` (~44K) from prompt. Keep `gemara://lexicon` (~6K).

**Why:** The CUE schema contains 20+ type definitions. The agent only authors `AuditLog`. A concrete YAML template (200 chars) conveys the structure better than 44K of CUE. The lexicon provides domain vocabulary the agent can't infer.

**Alternative:** Preload only the `#AuditLog` CUE definition. Rejected because the YAML template is more directly usable and smaller.

### D4: Inline the AuditLog template in prompt.md

**Choice:** Move the AuditLog YAML template from the audit-methodology skill directly into `prompt.md` under the "Author AuditLog" workflow step.

**Why:** The template is the single most critical structural constraint. It must be adjacent to the instruction that uses it, not buried in a skill file among other content.

### D5: Compact table reference format

**Choice:** List tables as `table_name: col1, col2, col3` — one line per table, columns comma-separated. No type annotations, no descriptions.

**Why:** The agent needs column names for correct SQL. Types and descriptions are discoverable via `DESCRIBE TABLE` if needed. One-line-per-table keeps the entire schema reference under 1K chars.

**Alternative:** Include types (e.g., `policy_id String`). Rejected — adds bulk without improving query correctness. The agent infers types from context (IDs are strings, timestamps are dates).

## Risks / Trade-offs

**[Risk] Agent issues bad queries without type info** → Low risk. Column names are strongly typed by convention (`_id` = String, `_at` = DateTime). If the agent needs types, it can run `DESCRIBE TABLE <name>`.

**[Risk] Agent can't author non-AuditLog artifacts** → Acceptable. The prompt explicitly states the agent only authors AuditLogs. Other artifact types are authored by engineers.

**[Risk] Removing the CUE preload breaks validate_gemara_artifact** → No risk. The validation tool calls the MCP server, which has its own schema. The preload was for the LLM's understanding, not the tool's.

**[Risk] Coverage-mapping detail lost** → Low risk. The coverage-mapping skill's join logic and status derivation tables are important but rarely exercised (most audits skip cross-framework analysis). Move the coverage matrix rules to an appendix section at the end of the consolidated skill — present but not front-loaded.

## Migration Plan

1. Create `skills/studio-audit/SKILL.md` with consolidated content
2. Update `prompt.md` with inline AuditLog template and compact table reference
3. Modify `main.py` to skip `gemara://schema/definitions` preload (keep lexicon)
4. Delete old skill directories (`audit-methodology/`, `evidence-schema/`, `coverage-mapping/`, `gemara-mcp/`)
5. Run `make deploy` — single deployment, no data migration
6. Rollback: revert the 4 files + re-add old skills. No state to clean up.
