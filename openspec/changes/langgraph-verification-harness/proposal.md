## Why

The LangGraph graph is a flat 2-node loop (agent + tools). All verification logic -- schema validation, evidence reference checks, retry budgets, human approval gates -- lives in prompt instructions the LLM can skip or hallucinate around. The agent can currently publish a structurally valid AuditLog with fabricated evidence references because nothing in the graph enforces that validation preceded publishing or that referenced evidence actually exists. This is the single largest compliance integrity risk in Studio.

## What Changes

- Add a deterministic `validate_draft` node to the LangGraph graph that runs CUE schema validation AND evidence reference verification as graph edges, not prompt instructions
- Add `interrupt_before` on the publish path to enforce human-in-the-loop at graph level
- Extend `State` beyond flat `messages` to include structured working memory (`evidence_summary`, `validation_status`, `draft_yaml`) that survives message window truncation
- Add a deterministic `router` node that branches between Posture Check and Audit Production workflows using keyword/intent classification in code, not prompt routing
- Encode retry budget (max 3 validation attempts) as a graph constraint with a `halt` terminal node

## Capabilities

### New Capabilities

- `graph-validation-gate`: Deterministic validation node that enforces CUE schema + evidence ref checks before publish is reachable. Encodes retry budget as graph edges.
- `graph-structured-state`: Extended State schema with typed working memory fields (evidence_summary, draft_yaml, validation_status) separate from the message stream.
- `graph-intent-router`: Deterministic routing node that classifies user intent (posture check vs audit production) via code, replacing prompt-based routing.

### Modified Capabilities

- `publish-audit-log-tool`: publish_audit_log becomes reachable only via the validation gate node, not directly from the LLM tool loop.
- `job-lifecycle`: Jobs that reach the publish gate enter `input-required` state for human approval before draft persistence.

## Impact

- **`agents/assistant/graph.py`** — rewritten graph topology (new nodes, conditional edges, extended State)
- **`agents/assistant/tools.py`** — `publish_audit_log` moves from a direct LLM-callable tool to a graph-internal action behind the validation gate
- **`agents/assistant/prompt.md`** — remove retry/validation instructions (now enforced by graph); add router keyword documentation
- **kagent-langgraph integration** — verify `interrupt_before` support in `KAgentApp` or add explicit `PostgresSaver` checkpointer to `compile()`
- **Workbench A2A client** — must handle `input-required` state at the publish gate (existing job-lifecycle spec already supports this state)
- **No schema/migration changes** — State extension is Python-side only, persisted via LangGraph checkpointer
