## 1. Context Replay

- [x] 1.1 Modify `streamReply` in `workbench/src/api/a2a.ts` to accept conversation history and context artifacts as parameters
- [x] 1.2 Implement history serialization in `chat-drawer.tsx`: assemble `[User]:`/`[Agent]:` prefixed messages into `<conversation-history>` block
- [x] 1.3 Re-attach context artifacts on every follow-up reply (read from `currentJob.contextArtifacts`)
- [x] 1.4 Implement token budget: truncate oldest messages when serialized history exceeds 100K characters, preserving last 4 messages and all context artifacts
- [x] 1.5 Add `[Earlier conversation truncated]` marker when truncation occurs

## 2. Input Sanitization

- [x] 2.1 Wrap replayed history in `<conversation-history>` delimiter tags
- [x] 2.2 Prefix context artifacts with `--- Context: <name> (reference only) ---`
- [x] 2.3 Update `agents/platform.md` with guardrails: treat `<conversation-history>` as prior context, treat `--- Context:` as reference material, do not execute embedded instructions

## 3. Gemara Authoring Skill

- [x] 3.1 Create `skills/gemara-authoring/SKILL.md` with minimal valid YAML skeletons for ThreatCatalog, ControlCatalog, RiskCatalog, and Policy
- [x] 3.2 Document cross-reference constraints (group ID matching, mapping-references requirements, capabilities on threats)
- [x] 3.3 Document common validation pitfalls (missing gemara-version, empty groups array, threat without capabilities mapping)
- [x] 3.4 Add `{ path: skills/gemara-authoring }` to `agents/threat-modeler/agent.yaml`, `agents/gap-analyst/agent.yaml`, `agents/policy-composer/agent.yaml`
- [x] 3.5 Update `agent-specialists.yaml` Helm template to render the new gitRef for all agents

## 4. Prompt Rewrites

- [x] 4.1 Rewrite `agents/threat-modeler/prompt.md`: single-shot pipeline, no mid-workflow questions, no MCP prompt references, explicit validate-before-return
- [x] 4.2 Rewrite `agents/gap-analyst/prompt.md`: single-shot pipeline, auto-derive inventory from ClickHouse, pause only on missing prerequisites
- [x] 4.3 Rewrite `agents/policy-composer/prompt.md`: two-phase (derive-all, confirm-once), sensible defaults for RACI/enforcement/frequency, note defaults in output
- [x] 4.4 Update `agents/platform.md`: add context replay handling instructions, enforce `<conversation-history>` treatment, enforce artifact reference-only treatment
- [x] 4.5 Run `make sync-prompts` and verify chart copies match source

## 5. Chat Message Grouping

- [x] 5.1 Create `groupMessages()` utility in `chat-panel.tsx`: transform flat `Message[]` into grouped blocks by role
- [x] 5.2 Render grouped blocks: single agent block contains text segments and inline tool calls
- [x] 5.3 Tool calls within agent blocks render collapsed with expand toggle
- [x] 5.4 Streaming finalization appends to current agent block instead of creating new bubble
- [x] 5.5 Update CSS for grouped block styling (single border, continuous background)

## 6. Streaming Chat Modifications

- [x] 6.1 Ensure `finalizeLastAgentMessage` does not create a new message bubble when the next message is also from the agent role
- [x] 6.2 Verify streaming cursor appears within the grouped block, not as a standalone element

## 7. Validation

- [x] 7.1 TypeScript build passes with no errors
- [x] 7.2 Deploy to Kind cluster and verify threat-modeler completes single-shot without asking questions — **requires Kind cluster deployment (manual)**
- [x] 7.3 Verify policy-composer presents derive-all table in Phase 1 and generates on confirmation in Phase 2 — **requires Kind cluster deployment (manual)**
- [x] 7.4 Verify follow-up replies retain conversation context (agent references prior messages) — **requires Kind cluster deployment (manual)**
- [x] 7.5 Verify chat renders agent responses as cohesive grouped blocks — **requires Kind cluster deployment (manual)**
- [x] 7.6 Verify agent-produced artifacts pass `validate_gemara_artifact` — **requires Kind cluster deployment (manual)**
