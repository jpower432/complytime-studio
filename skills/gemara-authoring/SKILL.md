---
name: gemara-authoring
description: >-
  Minimal Gemara YAML skeletons and cross-reference rules for ThreatCatalog,
  ControlCatalog, RiskCatalog, and Policy authoring.
---

# Gemara authoring

Use these shapes as starting points. Always run `validate_gemara_artifact` against the correct definition before returning YAML.

## ThreatCatalog skeleton

```yaml
title: Example Threat Catalog
metadata:
  id: example-threats
  type: ThreatCatalog
  gemara-version: "1.0.0"
  description: Minimal skeleton — replace with real content.
  author:
    id: author-1
    name: Example Author
    type: Organization
groups:
  - id: grp-1
    title: Example group
    description: Threats in this group share a common concern.
threats:
  - id: th-1
    title: Example threat
    description: What can go wrong.
    group: grp-1
    capabilities:
      reference-id: cap-cat-ref
      entries:
        - reference-id: cap-1
```

## ControlCatalog skeleton

```yaml
title: Example Control Catalog
metadata:
  type: ControlCatalog
  applicability-groups:
    - id: app-1
      title: Default applicability
groups:
  - id: cg-1
    title: Control group
    description: Controls for this scope.
controls:
  - id: ctl-1
    title: Example control
    objective: Mitigate identified risks.
    group: cg-1
    assessment-requirements:
      - id: ar-1
        text: Verify implementation.
        applicability: app-1
        state: active
    threats:
      - reference-id: th-1
    state: active
```

## RiskCatalog skeleton

```yaml
title: Example Risk Catalog
metadata:
  type: RiskCatalog
groups:
  - id: rc-1
    type: RiskCategory
    title: Operational risk
    description: Loss of availability or integrity.
    appetite: moderate
    max-severity: high
risks:
  - id: rk-1
    title: Example risk
    description: Exposure from linked threats.
    group: rc-1
    severity: medium
    impact: Optional narrative — tie to business consequence types when used.
    threats:
      - reference-id: th-1
```

## Policy skeleton

```yaml
title: Example Policy
metadata:
  type: Policy
  id: example-policy
contacts:
  responsible:
    - id: raci-r1
      name: Policy Owner
      type: Person
  accountable:
    - id: raci-a1
      name: Executive Sponsor
      type: Person
  consulted: []
  informed: []
scope:
  in:
    - dimension: technology
      description: In-scope systems
imports:
  catalogs:
    - reference-id: ctl-cat-ref
adherence:
  assessment-plans:
    - id: plan-1
      title: Quarterly review
      frequency: quarterly
      targets:
        - reference-id: ar-1
  evaluation-methods:
    - id: em-1
      title: Automated check
      type: Technical
```

## Cross-reference constraints

- **Group IDs**: Every `group` field on entries (`threats`, `controls`, `risks`) MUST match an `id` on a row in `groups` for that same document.
- **Imports / extends / mappings**: When a document `imports`, `extends`, or maps to another artifact, include the proper **mapping-reference** (or catalog import reference) so validators can resolve external IDs. Do not invent IDs that are not imported or declared.
- **Threat capabilities**: `threats[].capabilities` requires a **CapabilityCatalog** (or equivalent capability source) reachable via a declared mapping-reference; `reference-id` under `capabilities` must resolve to real capability entries.

## Common pitfalls

- **Missing `gemara-version`** on ThreatCatalog metadata — validation fails early.
- **Empty `groups`** while threats/controls reference group IDs — broken references.
- **Threat without `capabilities` mapping** when your catalog expects capability linkage — incomplete Layer 2 traceability.
- **Control without `assessment-requirements`** — controls must be testable.
- **`RiskCategory` missing `appetite`** — required field for risk grouping semantics.
- **Stale threat IDs on controls** — `controls[].threats` must reference threat IDs that exist in the imported ThreatCatalog (or declared mapping scope).
