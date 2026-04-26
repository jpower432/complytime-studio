## Why

The assistant agent's system prompt is ~80K chars (3.6K prompt + 25K skills + 44K CUE schema + 2K few-shot). This causes three observed failures:

1. **Literal template copying** — SQL query patterns with placeholder syntax (`{policy_id}`, `<POLICY_ID>`) get sent verbatim to ClickHouse, producing "Context variable not found" errors.
2. **Field name confusion** — The 44K CUE schema dump contains multiple similar mapping types (`ArtifactMapping.reference-id` vs `EntryMapping.entry-id`). The LLM cannot reliably extract the correct field from a wall of CUE, producing invalid AuditLogs.
3. **Attention dilution** — Critical rules (use `reference-id` not `entry-id`, auto-query mappings) are buried in noise. The LLM ignores them.

The previous `prompt-reliability` change added the skills and schema preloading. This change reverses the "stuff everything in the prompt" approach and replaces it with a compact prompt + tool-based discovery.

## What Changes

- **Remove** the 44K raw CUE schema dump from the system prompt
- **Remove** SQL query template patterns from `evidence-schema` skill (the agent knows SQL; it needs table metadata, not copy-paste templates)
- **Add** a compact AuditLog template with explicit field rules (the one structural constraint the LLM cannot infer)
- **Replace** skill-based table documentation with a tool-discoverable schema (agent runs `DESCRIBE TABLE` or equivalent)
- **Consolidate** four separate skills into a single focused skill that fits under 4K chars
- **Keep** few-shot classification examples (small, high-value)
- **Keep** Gemara lexicon preload (~6K, provides domain vocabulary)
- **Remove** `gemara://schema/definitions` preload (~44K, low signal-to-noise for LLM)

Target: system prompt under 15K chars total (from ~80K).

## Capabilities

### New Capabilities
- `lean-agent-prompt`: Consolidated prompt with compact table metadata, AuditLog template, and workflow — replacing four separate skills and CUE schema preloading

### Modified Capabilities
- `agent-spec-skills`: Skill loading mechanism changes from 4 separate skills to 1 consolidated skill
- `agent-context-injection`: Schema resource preloading reduced to lexicon-only (drop 44K CUE definitions)

## Impact

- `agents/assistant/prompt.md` — rewritten
- `skills/` — four skills consolidated into one (`skills/studio-audit/SKILL.md`)
- `skills/audit-methodology/SKILL.md` — removed (content merged)
- `skills/evidence-schema/SKILL.md` — removed (replaced with compact table list + DESCRIBE TABLE)
- `skills/coverage-mapping/SKILL.md` — removed (content merged)
- `skills/gemara-mcp/SKILL.md` — removed (content merged)
- `agents/assistant/main.py` — remove `gemara://schema/definitions` preload, keep `gemara://lexicon`
- `agents/assistant/prompts/few-shot/*.yaml` — kept as-is
