## Context

The Studio assistant graph (`agents/assistant/graph.py`) is a flat 2-node LangGraph loop: `agent` (LLM with bound tools) → `tools` (ToolNode with SQL guard) → back to `agent`. All workflow phases, validation logic, and retry budgets are expressed as prompt instructions in `prompt.md`. The LLM can skip validation, fabricate evidence references, or bypass the intended workflow order because the graph has no structural enforcement.

Gemara is the data contract. Artifacts are validated against CUE schema via `validate_gemara_artifact` (MCP tool on studio-gemara-mcp). Evidence is stored in PostgreSQL with `policy_id` as the join key. The `publish_audit_log` local tool persists drafts to the gateway's internal endpoint.

kagent-langgraph (`KAgentApp`) wraps the compiled graph and provides A2A serving + checkpointing. The workbench communicates via SSE streaming over `/a2a/*`.

## Goals / Non-Goals

**Goals:**
- Eliminate the risk of hallucinated compliance evidence by making validation a graph-enforced prerequisite to publishing
- Make retry budgets deterministic (max 3 attempts encoded as graph state, not prompt instruction)
- Support human-in-the-loop approval as a graph-level interrupt, not a separate workflow
- Preserve structured working memory (evidence refs, draft YAML, validation status) across message window truncation
- Route between Posture Check and Audit Production workflows deterministically

**Non-Goals:**
- Hybrid model orchestration (planner/worker split) — deferred to a later change
- OpenAPI spec generation — separate concern
- Materialized SQL views — separate change
- Cryptographic agent identity (SPIFFE) — infrastructure dependency
- Persistent chat store migration to PostgreSQL — covered by existing `session-persistence-storage` ADR

## Decisions

### D1: Graph topology — subgraph per workflow

**Decision:** Replace the flat 2-node graph with a router node that dispatches to either a `posture_check` subgraph or an `audit_production` subgraph.

**Rationale:** LangGraph `StateGraph` supports subgraphs natively. Separating workflows makes each auditable in LangGraph Studio and prevents cross-contamination (a posture check cannot accidentally trigger publish).

**Alternative rejected:** Single graph with conditional edges for all phases. Grows unwieldy as workflows diverge. Subgraphs compose better.

### D2: Validation gate — deterministic node, not LLM tool call

**Decision:** The `validate_draft` node calls `validate_gemara_artifact` via MCP directly (not via LLM tool binding) and verifies evidence references exist in PostgreSQL via a direct SQL query. This node is a Python function, not an LLM invocation.

**Rationale:** The LLM cannot skip a graph node. By making validation a node on the path to publish, we guarantee it executes. Evidence ref verification prevents fabrication — every `evidence_id` in the draft must resolve to a row in the `evidence` table within the declared audit window.

**Alternative rejected:** Adding validation as a `before_tool` hook on `publish_audit_log`. This still allows the LLM to call publish without triggering the hook if tool routing changes.

### D3: State extension — TypedDict with structured fields

**Decision:** Extend `State` with typed fields beyond `messages`:

```python
class AuditState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]
    intent: str                    # "posture_check" | "audit_production" | ""
    draft_yaml: str                # current draft artifact
    evidence_refs: list[str]       # evidence_ids referenced in draft
    validation_result: dict        # {"valid": bool, "errors": [...]}
    validation_attempts: int       # counter for retry budget
    target_inventory: list[dict]   # discovered targets
```

**Rationale:** LangGraph checkpoints the full State. Structured fields survive message window truncation. The agent node can read `evidence_refs` and `validation_result` without re-querying or relying on message history.

**Alternative rejected:** Storing working memory in tool messages only. Subject to truncation and requires parsing unstructured text.

### D4: Interrupt mechanism — `interrupt_before` on publish

**Decision:** Use LangGraph's `interrupt_before=["publish_draft"]` when compiling the subgraph. This checkpoints state and signals `input-required` via A2A, which the workbench already handles per the job-lifecycle spec.

**Rationale:** The existing job lifecycle supports `input-required` → user reply → resume. `interrupt_before` is the native LangGraph mechanism for this. No new protocol needed.

**Alternative rejected:** External approval queue (separate service). Over-engineers the problem for v1 where approval is same-session.

### D5: Router classification — keyword + fallback LLM

**Decision:** The router node uses a keyword classifier first (fast, deterministic). If no keywords match, falls back to a single LLM call with a constrained output schema (`{"intent": "posture_check" | "audit_production" | "ambiguous"}`). If `ambiguous`, the graph emits a clarifying question via a dedicated node.

**Rationale:** Most routing is unambiguous ("run an audit" vs "how ready are we"). Keywords handle 80%+ of cases with zero latency. The fallback handles edge cases without blocking the fast path.

### D6: Checkpointer — explicit PostgresSaver

**Decision:** Pass `PostgresSaver` to `builder.compile(checkpointer=saver)` alongside the existing `KAgentApp` wrapping. Configure it to use the same PostgreSQL instance as the studio data store (via `POSTGRES_URL` env).

**Rationale:** Makes persistence visible, testable, and independent of kagent internals. If `KAgentApp` also wraps checkpointing, the explicit saver takes precedence (LangGraph uses the checkpointer passed to compile).

**Alternative rejected:** Relying solely on kagent's opaque checkpointer. Not testable in this repo. Behavior undocumented for interrupt scenarios.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| `interrupt_before` may not propagate correctly through `KAgentApp` A2A layer | Test with kagent-langgraph 0.9.2 in dev. If incompatible, use explicit `interrupt()` call inside the publish node instead. |
| Evidence ref verification adds latency (SQL query per draft) | Query is a simple `WHERE evidence_id IN (...)` against indexed column. Expected <50ms even for 100 refs. |
| Router keyword list requires maintenance | Start with high-signal keywords from prompt.md. Add telemetry to log fallback-to-LLM rate. If >20%, expand keyword list. |
| Subgraph complexity increases onboarding cost | Mitigate with LangGraph Studio visualization. Document graph topology in a mermaid diagram in the agent README. |
| PostgresSaver adds a dependency on `langgraph-checkpoint-postgres` | Already compatible with the existing PostgreSQL instance. Pin version in requirements.txt. |

## Open Questions

1. Does `KAgentApp.build()` respect `interrupt_before` set during `compile()`? Needs integration test with kagent-langgraph 0.9.2.
2. Should the evidence ref check verify `collected_at` is within the audit window, or just verify existence? (Recommendation: verify window — prevents stale evidence citation.)
3. Should the router node emit the clarifying question as a regular message or as an `input-required` interrupt? (Recommendation: regular message — reserve interrupts for publish gate.)
