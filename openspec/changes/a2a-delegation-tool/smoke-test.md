# Smoke Test: A2A Delegation Tool

Run after deploying the updated assistant and workbench to verify delegation and routing changes.

## Prerequisites

- Studio deployed with at least one BYO agent registered in `agentDirectory`
- BYO agent accessible at its declared URL via AgentGateway
- Assistant pod has `GATEWAY_URL` set (for directory resolution)

## Test 1: Workbench always routes to studio-assistant

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Open workbench, observe chat panel | Header shows "Studio Assistant" |
| 2 | If multiple agents registered, observe picker | Picker is visible but informational |
| 3 | Change picker selection | NO session wipe. Messages remain. No new A2A task created. |
| 4 | Send a message | Network tab shows POST to `/a2a/studio-assistant` (not the selected agent) |

**Pass criteria:** Routing always goes to `studio-assistant` regardless of picker state.

## Test 2: Delegation fires when needs_delegation is set

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Configure a policy that triggers delegation (set `needs_delegation` via policy metadata or LLM flag) | Policy recognized |
| 2 | Start audit workflow for that policy | Graph routes through `delegate` node before LLM |
| 3 | Observe assistant's response | Response incorporates BYO agent's domain data |

**Pass criteria:** Worker data appears in the conversation. Assistant references it in analysis.

## Test 3: Delegation skipped when not needed

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Start audit for a standard policy (no delegation trigger) | Graph skips delegate node |
| 2 | Check logs or LangGraph Studio trace | No A2A call to BYO agent |

**Pass criteria:** Standard audits work unchanged. No unnecessary delegation.

## Test 4: BYO agent unavailable — graceful degradation

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Scale BYO agent to 0 replicas (or block network) | Agent unreachable |
| 2 | Trigger audit with delegation | Delegate node fires, fails |
| 3 | Observe assistant response | Reports "Agent unavailable" to user. Does not crash or hang. |

**Pass criteria:** Error in `worker_data`. LLM reports the issue. Audit continues without worker data (degraded but functional).

## Test 5: BYO agent returns successfully

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Ensure BYO agent is running and healthy | Agent responds to A2A |
| 2 | Trigger delegation | Delegate node calls BYO agent |
| 3 | Verify `worker_data` in State | Contains BYO agent's response keyed by agent ID |
| 4 | Verify LLM references the data | Next LLM turn mentions domain-specific content from worker |

**Pass criteria:** End-to-end delegation round-trip works. Data flows from BYO agent into assistant's analysis.

## Test 6: X-Agent-ID header sent on delegation

| Step | Action | Expected |
|:--|:--|:--|
| 1 | Enable request logging on AgentGateway or BYO agent | Headers visible |
| 2 | Trigger delegation | A2A call fires |
| 3 | Check request headers | `X-Agent-ID: studio-assistant` present |

**Pass criteria:** Identity header propagated. CEL policies can match.

## Quick Validation (CI-friendly)

```bash
cd agents/assistant
python -m pytest test_delegate.py test_router.py test_validation.py test_tools.py -v
```

All 59 tests must pass.
