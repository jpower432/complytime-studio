---
name: risk-reasoning
description: >-
  Qualitative risk reasoning for policy composition — appetite vs tolerance,
  graph-derived prioritization signals, catalog/policy boundary, residual risk
  patterns, and severity justification heuristics.
---

# Risk Reasoning

Use this skill when enriching RiskCatalog entries from a ThreatCatalog: justify
severity with **countable graph signals** (no invented numeric scores), respect
**risk appetite vs tolerance**, and keep **catalog vs policy** responsibilities
clear.

## Appetite vs tolerance (ISO 31000 framing)

| Term | Meaning in this workflow |
|:-----|:-------------------------|
| **Risk appetite** | Strategic posture — what kinds of loss the organization chooses to pursue or accept at a category level (directional, qualitative). |
| **Risk tolerance** | Concrete boundary — expressed here as a **cap** on acceptable loss per category via `RiskCategory.max-severity` (or equivalent). A risk whose `severity` exceeds the group's `max-severity` **violates tolerance**. |

Appetite answers “what risk posture fits our strategy?” Tolerance answers “did
we cross a defined line for this category?”

## Prioritization signals (all countable from the threat graph)

Derive these from `ThreatCatalog` structure only — **counts and booleans**, not
weighted scores:

| Signal | How to count |
|:-------|:-------------|
| **Threat density** | `count(risk.threats[].entries[])` (or equivalent threat links for that risk). |
| **Vector breadth** | Count of attack **vectors** linked to those threats (sum or distinct count per your catalog shape — be explicit in rationale). |
| **Capability exposure** | Count of **capabilities** linked to those same threats. |
| **Tolerance violation** | Binary: compare risk `severity` to the owning `RiskCategory` `max-severity` ordering (e.g., Critical > High > Medium > Low). |
| **Impact type** | Classify narrative impact as **financial**, **operational**, and/or **reputational** based on threat/control context (qualitative tags, not scores). |

## Catalog vs policy data boundary

| Artifact | Boundary |
|:---------|:---------|
| **Catalogs** (e.g., ThreatCatalog, RiskCatalog as *inherent* risk register) | **Context-free** — describe scenarios, threats, and risks without binding to a specific deployed inventory. |
| **Policy** | **Context-bound** — imports catalogs, selects controls, and binds mitigations/acceptances to **this** organization's scope and decisions (`Policy.imports`, `Policy.risks.*`, scope, RACI). |

Keep inherent risk in catalogs; keep “what we do about it here” in Policy.

## Severity justification heuristics (signal → band)

Use these **example mappings** as qualitative guides — always cite the actual
counts and threat/vector IDs from the user's ThreatCatalog:

| Band | Example signal pattern |
|:-----|:-----------------------|
| **Critical** | **3+** linked threats **and** **2+** vectors per threat on average **or** **wide** capability surface across those threats **OR** any **tolerance violation** (severity above category `max-severity`). |
| **High** | **2+** threats with **multiple** vectors each **or** **moderate** capability exposure (several capabilities touched across the set). |
| **Medium** | **1–2** threats, **limited** vector breadth, **narrow** capability surface. |
| **Low** | **Single** threat, **single** primary vector, **minimal** capability exposure, **no** tolerance violation. |

Document the rationale next to the assignment: e.g., “High: 2 threats (t-01,
t-04), 5 vectors total, 4 capabilities referenced; within tolerance cap
High.”

## Residual risk flow pattern

1. **Inherent Risk Catalog** — document risks, linked threats, and severities
   (pre-control).
2. **Policy** — import controls that mitigate those risks; record coverage under
   `Policy.risks.mitigated` (or equivalent mitigation linkage).
3. **Residual Risk Catalog** — where controls do not fully address linked
   threats, document **remaining exposure** as a **new** RiskCatalog (residual
   entries should reference inherent risk IDs).
4. **Policy update** — reference residual risks as **accepted** where
   appropriate (`Policy.risks.accepted` with explicit justification), or drive
   further control work.
5. **EvaluationLog** — over time, verify controls addressing linked threats are
   **passing**; failing checks reopen residual exposure.
