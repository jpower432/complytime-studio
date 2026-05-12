## 1. State Schema and Dependencies

- [ ] 1.1 Add `langgraph-checkpoint-postgres` to `agents/assistant/requirements.txt`
- [ ] 1.2 Define `AuditState(TypedDict)` in `agents/assistant/state.py` with all typed fields (messages, intent, draft_yaml, evidence_refs, validation_result, validation_attempts, target_inventory)
- [ ] 1.3 Add `POSTGRES_URL` env var to `charts/complytime-studio/templates/byo-assistant.yaml`

## 2. Intent Router

- [ ] 2.1 Create `agents/assistant/router.py` with keyword constants (`POSTURE_KEYWORDS`, `AUDIT_KEYWORDS`) and `classify_intent(message: str) -> str` function
- [ ] 2.2 Implement LLM fallback classifier with constrained output schema in `router.py`
- [ ] 2.3 Create `router_node(state)` function that sets `state.intent` and returns routing decision
- [ ] 2.4 Create `clarify_node(state)` that emits the disambiguation question
- [ ] 2.5 Write unit tests for keyword matching (exact, case-insensitive, substring)

## 3. Validation Gate

- [ ] 3.1 Create `agents/assistant/validation.py` with direct MCP client for `validate_gemara_artifact`
- [ ] 3.2 Implement `verify_evidence_refs(refs: list[str], policy_id: str, window_start: str, window_end: str) -> list[str]` using direct SQL via postgres-mcp
- [ ] 3.3 Implement `validate_draft_node(state: AuditState) -> dict` combining schema validation + evidence ref check
- [ ] 3.4 Implement `route_after_validation(state: AuditState) -> str` with retry budget logic (max 3 → halt)
- [ ] 3.5 Create `halt_node(state)` that emits accumulated errors as a final message
- [ ] 3.6 Write unit tests for validation node (pass, schema fail, ref fail, window fail, retry exhaustion)

## 4. Publish Gate with Interrupt

- [ ] 4.1 Refactor `publish_audit_log` from a `@tool` to a plain async function (no longer LLM-callable)
- [ ] 4.2 Create `publish_draft_node(state: AuditState)` that calls the refactored publish function with `state.draft_yaml` and `state.evidence_refs`
- [ ] 4.3 Remove `publish_audit_log` from `build_tools()` return list
- [ ] 4.4 Verify `interrupt_before=["publish_draft"]` works with `KAgentApp` in integration test

## 5. Graph Topology Rewrite

- [ ] 5.1 Create `agents/assistant/subgraphs/audit.py` with the audit production subgraph (agent → tools → validate_draft → publish_draft, with halt branch)
- [ ] 5.2 Create `agents/assistant/subgraphs/posture.py` with posture check subgraph (agent → tools → end, no publish path)
- [ ] 5.3 Rewrite `agents/assistant/graph.py` top-level graph: `__start__` → `router` → conditional edges to subgraphs
- [ ] 5.4 Configure `PostgresSaver` checkpointer in `graph.py` with `POSTGRES_URL`
- [ ] 5.5 Pass `interrupt_before=["publish_draft"]` to audit subgraph compile
- [ ] 5.6 Remove `validate_gemara_artifact` from LLM-bound tools in the audit subgraph (validation is deterministic node)

## 6. Prompt Updates

- [ ] 6.1 Remove retry/validation instructions from `agents/assistant/prompt.md` (now graph-enforced)
- [ ] 6.2 Remove intent routing instructions from `prompt.md` (now deterministic router)
- [ ] 6.3 Add documentation of the validation gate behavior ("Your draft will be validated automatically. If errors are found, you will receive them and can fix the draft.")
- [ ] 6.4 Run `make sync-prompts` to copy updated prompt to chart

## 7. Integration Testing

- [ ] 7.1 Test: router classifies "run an audit on ampel" as `audit_production` without LLM call
- [ ] 7.2 Test: router classifies "how ready are we for ampel" as `posture_check` without LLM call
- [ ] 7.3 Test: validation gate rejects draft with fabricated evidence_id
- [ ] 7.4 Test: validation gate rejects draft with evidence outside audit window
- [ ] 7.5 Test: retry budget halts after 3 failures
- [ ] 7.6 Test: interrupt fires at publish gate and resumes after human reply
- [ ] 7.7 Test: PostgresSaver persists state across simulated pod restart
- [ ] 7.8 Test: posture check subgraph cannot reach publish_draft node

## 8. Helm and Deployment

- [ ] 8.1 Add `POSTGRES_URL` env to `byo-assistant.yaml` template (reuse existing PostgreSQL secret)
- [ ] 8.2 Verify `helm template` renders correctly with new env
- [ ] 8.3 Update `agents/assistant/agent.yaml` to document the new graph topology
- [ ] 8.4 Add mermaid graph diagram to agent README or design doc
