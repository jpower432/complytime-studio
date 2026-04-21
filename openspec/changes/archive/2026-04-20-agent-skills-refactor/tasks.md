## 1. Delete dead agents and platform.md

- [x] 1.1 Delete `agents/threat-modeler/` directory
- [x] 1.2 Delete `agents/policy-composer/` directory
- [x] 1.3 Delete `agents/platform.md`
- [x] 1.4 Update `agent-prompts-configmap.yaml` — remove `threat-modeler` and `policy-composer` keys, keep `assistant` only
- [x] 1.5 Verify `agent-specialists.yaml` comment is accurate (no references to deleted agents)

## 2. Create internal skills

- [x] 2.1 Create `skills/gemara-mcp/SKILL.md` — layer model (L1-L7), MCP tools (validate, migrate), MCP resources (lexicon, schema/definitions), validation workflow, which layers assistant produces vs consumes
- [x] 2.2 Create `skills/evidence-schema/SKILL.md` — ClickHouse table schemas (evidence, policies, mapping_documents, audit_logs), column types, enum values, standard query patterns (inventory, per-target, cadence)
- [x] 2.3 Create `skills/audit-methodology/SKILL.md` — assessment cadence rules, frequency-to-cycle mapping, classification criteria (Strength/Finding/Gap/Observation), satisfaction determination levels. Informed by `rhaml-23/prompt` `control-assessment-spec.md` and `control-coverage-spec.md`
- [x] 2.4 Create `skills/coverage-mapping/SKILL.md` — cross-framework join logic, strength/confidence table, coverage status derivation, multi-mapping resolution, matrix format. Informed by `rhaml-23/prompt` `control-coverage-spec.md`

## 3. Wire external skills

- [x] 3.1 Update `agents/assistant/agent.yaml` — add external gitRef for `rhaml-23/prompt` `skills/research.md`
- [x] 3.2 Update `agents/assistant/agent.yaml` — add external gitRef for `rhaml-23/prompt` `skills/gemara.md`
- [x] 3.3 Update `agents/assistant/agent.yaml` — replace dead internal skill refs (`gemara-layers`, `gemara-authoring`) with real internal refs (`gemara-mcp`, `evidence-schema`, `audit-methodology`, `coverage-mapping`)
- [x] 3.4 Remove dead external gitRef for `gemaraproj/gemara-skills` `skills/audit-classification`

## 4. Rewrite assistant prompt

- [x] 4.1 Rewrite `agents/assistant/prompt.md` — workflow only (~30-50 lines): identity, tool awareness (gemara-mcp, clickhouse-mcp), workflow steps (gather → inventory → assess → author → validate), output format, constraints from platform.md
- [x] 4.2 Run `make sync-prompts` to copy rewritten prompt into Helm chart

## 5. Verify

- [x] 5.1 `go build ./...` passes (no Go changes expected, but verify ConfigMap template renders)
- [x] 5.2 No remaining references to `gemara-layers`, `gemara-authoring`, `gemaraproj/gemara-skills`, or `openssf/stride-skills` in non-archive files
- [x] 5.3 All internal skill paths in `agent.yaml` resolve to existing `SKILL.md` files
- [x] 5.4 External gitRef repos return 200 (already verified: `rhaml-23/prompt` exists)
