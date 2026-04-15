---
name: policy-risk-linkage
description: Risk-to-control linkage logic for Policy composition — join risks and controls via shared threat references.
---

# Risk-to-Control Linkage

Determine which controls mitigate which risks by joining on shared threat references.

## Join Logic

1. For each `RiskCatalog.risks[]` entry, collect the threat IDs from `risks[].threats[].entries[].reference-id`.
2. For each `ControlCatalog.controls[]` entry, collect the threat IDs from `controls[].threats[].entries[].reference-id`.
3. A control mitigates a risk when they share at least one threat reference ID.

## Classification

| Risk State | Condition | Policy Field |
|:-----------|:----------|:-------------|
| Mitigated | At least one control shares a threat reference with the risk | `Policy.risks.mitigated` |
| Accepted | No controls share threat references AND user provides justification | `Policy.risks.accepted` with `justification` text |
| Unmitigated | No controls share threat references AND user has not yet decided | Present to user for decision |

## Presentation Format

Present the linkage analysis as a table:

```
| Risk    | Mitigated By          | Status     |
|:--------|:----------------------|:-----------|
| r-001   | ctrl-ac-01, ctrl-ac-02| Mitigated  |
| r-005   | (no controls)         | Accept?    |
```

For each unmitigated risk, ask the user: accept (with justification) or flag for future controls.
