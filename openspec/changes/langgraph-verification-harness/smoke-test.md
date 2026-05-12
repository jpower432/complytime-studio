# Smoke Test: LangGraph Verification Harness

Run after deploying the updated assistant image to verify the graph topology works end-to-end.

## Prerequisites

- Studio deployed with `postgres.enabled=true`
- `POSTGRES_URL` secret exists with valid connection string
- At least one policy imported with evidence in the audit window
- Gemara MCP server reachable

## Test 1: Router classifies posture check (no LLM routing)

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Open workbench, start new session |Session connects to `studio-assistant` |
| 2 | Type: "Check posture for ampel" | Agent responds with readiness table (posture check workflow) |
| 3 | Verify NO AuditLog is produced | Posture workflow has no publish path |

**Pass criteria:** Agent executes posture check without asking "do you want posture or audit?"

## Test 2: Router classifies audit production

| Step | Action | Expected |
|:--|:--|:--|
| 1 | New session | Clean state |
| 2 | Type: "Run an audit for ampel-branch-protection, last 30 days" | Agent starts evidence assembly (Phase 1) |
| 3 | Wait for evidence summary | Factual table appears — no classifications yet |

**Pass criteria:** Agent enters audit workflow without routing ambiguity.

## Test 3: Router handles ambiguous input

| Step | Action | Expected |
|:--|:--|:--|
| 1 | New session | Clean state |
| 2 | Type: "Tell me about ampel" | Agent asks: "Do you want a posture check or a full audit?" |
| 3 | Reply: "audit" | Agent proceeds to audit workflow |

**Pass criteria:** Clarify node fires. No hang, no guess.

## Test 4: Validation gate catches schema errors

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Trigger audit workflow (test 2) | Agent drafts AuditLog |
| 2 | Observe validation gate execution | Graph logs show `validate_draft_node` executing |
| 3 | If draft has errors, agent receives them and retries | Up to 3 attempts visible in conversation |

**Pass criteria:** Validation errors appear as agent messages (not silent failures). Agent self-corrects.

## Test 5: Human approval interrupt fires

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Complete audit workflow through validation | Draft passes validation |
| 2 | Job status transitions to `input-required` | Workbench shows approval prompt |
| 3 | Reply "approve" | Draft is published, confirmation message appears |

**Pass criteria:** Graph pauses at publish gate. Does NOT auto-publish without human reply.

## Test 6: Halt after 3 failures

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Simulate repeated validation failure | (May require temporarily breaking gemara-mcp or evidence data) |
| 2 | After 3 attempts | Agent emits halt message with all accumulated errors |
| 3 | Job transitions to `failed` | Workbench shows error state |

**Pass criteria:** Graph does not loop forever. Deterministic halt after 3 attempts.

## Test 7: State survives pod restart (checkpointer)

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Start audit, reach evidence assembly phase | Agent has queried evidence |
| 2 | Restart assistant pod (`kubectl delete pod studio-assistant-*`) | Pod recreates |
| 3 | Send follow-up message in same session | Agent resumes with prior context intact |

**Pass criteria:** Conversation state persisted by PostgresSaver. No "I don't have context" after restart.

## Test 8: Posture check cannot reach publish

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Run posture check (test 1) | Agent executes posture workflow |
| 2 | In same session, ask "publish this as an audit log" | Agent refuses or explains this is a posture check, not an audit |

**Pass criteria:** Posture subgraph has no publish_draft node. Cannot accidentally produce AuditLogs.

## Quick Validation (CI-friendly)

```bash
cd agents/assistant
python -m pytest test_router.py test_validation.py test_tools.py -v
```

All 56 tests must pass. This validates routing logic and validation gate routing without requiring a running cluster.
