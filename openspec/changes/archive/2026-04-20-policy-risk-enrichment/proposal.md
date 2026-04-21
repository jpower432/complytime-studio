## Why

The policy-composer already produces a RiskCatalog, but the risk entries lack depth. Severity is assigned without structured reasoning, and the qualitative signals in the Gemara schema (threat density, vector breadth, capability exposure, tolerance cap violations) are not leveraged. The result: every risk gets a severity label but no traceable justification, and the downstream Policy treatment decisions (mitigate vs accept) lack grounded rationale.

Adding risk reasoning as an optional enrichment step within the policy-composer workflow produces RiskCatalogs with justified severity assignments and Policies with evidence-backed treatment decisions -- without introducing a new agent or changing the existing job model.

## What Changes

- New `risk-reasoning` skill encapsulating domain knowledge: appetite vs tolerance semantics, prioritization signals (threat density, vector breadth, capability exposure, tolerance cap checks), residual risk pattern, and the catalog-vs-policy data boundary.
- Updated `policy-composer/prompt.md` Phase 1 workflow to optionally invoke risk reasoning: when the user requests enriched risk analysis, the agent traverses the threat graph, justifies severity assignments, flags tolerance violations, and produces impact narratives before proceeding to Policy authoring.
- Updated `policy-composer/agent.yaml` to reference the new skill.
- Existing "fast path" (derive risks without deep analysis) remains the default for users who skip enrichment.

## Capabilities

### New Capabilities
- `risk-reasoning`: Domain knowledge and workflow for qualitative risk analysis within policy composition — threat graph traversal, severity justification, tolerance cap checks, and residual risk identification.

### Modified Capabilities
- `job-lifecycle`: The policy-composer job flow adds an optional enrichment prompt ("Do you want risk analysis included?") before Phase 1 begins. No breaking change — declining enrichment preserves the current behavior.

## Impact

- `agents/policy-composer/prompt.md` — Phase 1 gains conditional enrichment steps
- `agents/policy-composer/agent.yaml` — new skill reference
- `skills/risk-reasoning/SKILL.md` — new file
- No new agent CRDs, no new MCP tools, no Helm template changes
- Existing policy-composer jobs without risk enrichment are unaffected
