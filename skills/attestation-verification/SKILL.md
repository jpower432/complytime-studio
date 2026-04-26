---
name: attestation-verification
description: On-demand provenance verification of evidence attestation chains against Policy-defined in-toto layouts
---

# Attestation Verification

Verify that evidence was produced by an authorized pipeline by checking in-toto attestation chains stored in OCI against layouts defined in the Policy's assessment plans.

## Inputs

| Input | Source | Required |
|:--|:--|:--|
| evidence_id | User query or ClickHouse lookup | Yes |
| attestation_ref | `evidence.attestation_ref` column | Yes (halt if NULL) |
| source_registry | `evidence.source_registry` column | No (use default registry when NULL) |
| layout digest | Policy `assessment-plans[].layout` field | No (report if absent) |

## Verification Steps

### 1. Retrieve attestation bundle

Query ClickHouse for the evidence row's `attestation_ref` and `source_registry`. When `source_registry` is set, use it as the OCI registry host for oras-mcp. When `source_registry` is NULL, use the default registry from environment or policy context.

The bundle contains one or more signed link files (JSON), each representing a pipeline step:

```json
{
  "step": "fetch-policy",
  "signer": "key-id-1",
  "materials": [{"uri": "oci://registry/policy:v1", "hash": "sha256:aaa"}],
  "products": [{"uri": "policy.yaml", "hash": "sha256:bbb"}],
  "timestamp": "2026-04-20T10:00:00Z"
}
```

### 2. Retrieve layout

Parse the Policy YAML to find `adherence.assessment-plans[]` matching the evidence's `plan_id`. Check for the `layout` field containing an OCI digest.

If `layout` is absent, report: "No layout defined for assessment plan <plan_id>. Attestation bundle exists but cannot be verified against expected steps."

Use oras-mcp to pull the layout from OCI.

### 3. Compare attestation against layout

The layout defines:
- **Expected steps** — ordered list of step names that must appear in the attestation
- **Authorized keys** — key identifiers allowed to sign each step
- **Material/product chaining** — step N's materials must match step N-1's products

For each expected step in the layout:

| Check | Pass | Fail |
|:--|:--|:--|
| Step exists in bundle | Link found for step name | "Missing step: <name>" |
| Signer authorized | Link signer matches layout's authorized keys | "Unauthorized signer: <key> for step <name>" |
| Materials chain | Step's materials match prior step's products | "Chain break: <step_a>.products ≠ <step_b>.materials" |

### 4. Return verdict

| Verdict | Condition |
|:--|:--|
| **CHAIN VERIFIED** | All expected steps present, all signers authorized, all material/product hashes chain |
| **BROKEN CHAIN** | One or more checks failed — report the first failure with specific details |
| **NO LAYOUT** | Attestation bundle exists but no layout reference in the assessment plan |
| **NO ATTESTATION** | `attestation_ref` is NULL — cannot verify |
| **REGISTRY UNAVAILABLE** | OCI registry unreachable — report registry host and halt without failing the conversation |
| **REGISTRY MISMATCH** | `source_registry` resolved but credentials missing or unauthorized — report the registry and do not claim verification success |

## Verdict Display

```
Evidence: <evidence_id>
Attestation: <attestation_ref>
Registry: <source_registry or "default">
Layout: <layout_digest or "none">

Verdict: CHAIN VERIFIED

| Step          | Signer  | Materials Match | Timestamp           |
|:--------------|:--------|:----------------|:--------------------|
| fetch-policy  | key-id-1| ✓               | 2026-04-20T10:00:00Z|
| evaluate      | key-id-2| ✓               | 2026-04-20T10:05:00Z|
```

## ClickHouse Tables

Query via `run_select_query`:

```
evidence: ..., attestation_ref, source_registry, ...
```

- `attestation_ref` — `Nullable(String)` containing an OCI digest (e.g., `sha256:abc123`).
- `source_registry` — `Nullable(String)` containing the OCI registry hostname where the attestation bundle resides. NULL means use the default registry.

