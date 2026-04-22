# Demo Script: SOC 2 Gap Analysis for Branch Protection

Live demo using real AMPEL branch protection policy from [complytime-policies](https://github.com/complytime/complytime-policies). Seeds 45 evidence records across 3 ComplyTime repositories scanned on 3 dates, with a SOC 2 Trust Services Criteria mapping.

## Setup

```bash
GATEWAY_URL=http://localhost:8080 STUDIO_API_TOKEN=dev-seed-token ./demo/seed.sh
```

## Evidence Summary

| Repository | BP-1 (PR Reviews) | BP-2 (Min Approvals) | BP-3 (Force Push) | BP-4 (Admin Bypass) | BP-5 (Code Owner) |
|:--|:--|:--|:--|:--|:--|
| complytime/complyctl | Passed | Passed | Passed | Passed | Passed |
| complytime/complytime-studio | Passed | Passed | Passed | **Failed** | **Failed** |
| complytime/complytime-policies | Passed | **Failed** | Passed | Passed | **Not Run** |

## Demo Flow

### Step 1: Orient — "What policy are we enforcing?"

> Show me the AMPEL branch protection policy and its controls.

**Expected:** Assistant loads the policy from ClickHouse, lists BP-1 through BP-5 with titles and assessment requirements. Establishes context for the audience.

### Step 2: Inventory — "What repos are being scanned?"

> What evidence do we have for the ampel-branch-protection policy? Show me all targets.

**Expected:** Assistant queries evidence, discovers 3 repositories (complyctl, complytime-studio, complytime-policies), shows 45 total records across 3 scan dates (April 7, 14, 16). Mentions all targets are `github-repository` type in production.

### Step 3: Gap analysis — "Where do we stand on SOC 2?"

> Run a SOC 2 gap analysis for policy ampel-branch-protection, audit period April 1-18 2026.

**Expected:** Assistant loads the SOC 2 mapping document, joins evidence results with CC8.1 and CC6.1 mappings. Should surface:
- **CC8.1 (Change Management):** Not fully covered — complytime-studio fails BP-4 (admin bypass) and BP-5 (code owner review); complytime-policies fails BP-2 (minimum approvals) and BP-5 is not run
- **CC6.1 (Logical Access):** At risk — BP-4 failure on complytime-studio weakens access controls
- **complyctl** is clean across all controls

### Step 4: Drill down — "What exactly is failing?"

> Show me the branch protection failures on complytime-studio. What's the risk?

**Expected:** Assistant returns BP-4.01 (admin bypass enabled) and BP-5.01 (code owner review not enabled) failures, consistent across all 3 scan dates. Should explain that admin bypass means protection rules can be overridden, and missing code owner review means changes to owned paths aren't reviewed by domain experts. Both are persistent — not a one-time issue.

### Step 5: Produce artifact — "Generate the audit log"

> Generate the audit log for this analysis.

**Expected:** Assistant produces a validated Gemara `#AuditLog` YAML artifact covering:
- 3 targets, 5 criteria each
- Findings for BP-4 (studio), BP-2 (policies), BP-5 (studio + policies)
- Gap for BP-5 on policies (Not Run = no evidence)
- SOC 2 CC8.1 and CC6.1 coverage assessment in recommendations
- Classification: complyctl = Strength, studio BP-4/BP-5 = Finding, policies BP-2 = Finding, policies BP-5 = Gap

## Troubleshooting

**"No policy found"** — Seed script didn't import. Re-run `seed.sh` and check for 201 responses.

**"No mapping documents"** — SOC 2 mapping import failed. Verify `mapping-soc2.json` was imported via the `/api/mappings/import` endpoint.

**"Streaming not supported"** — Assistant pod needs `InMemoryQueueManager` and `AgentCapabilities(streaming=True)`. Check `agents/assistant/main.py`.
