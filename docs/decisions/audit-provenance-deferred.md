# Hash-Chained Audit Provenance Deferred

**Status:** Deferred (trigger: regulatory requirement or Trillian integration)
**Date:** 2026-04-29

## Decision

Studio will not add hash-chained provenance (`prev_hash`, `entry_hash`) to the `audit_logs` ClickHouse table. The existing `ReplacingMergeTree` with content-addressed `audit_id` provides deduplication. Tamper-evident audit trails are deferred until a verifiable log infrastructure (e.g., Trillian) justifies the complexity.

## Context

As agents produce more audit artifacts, the question arises whether the `audit_logs` table should provide tamper-evidence — the ability to detect if stored records have been modified after the fact.

A candidate design was evaluated: SHA-256 hash chains where each entry's hash includes the previous entry's hash, forming a linked chain. A `GET /api/audit-logs/verify` endpoint would walk the chain and detect breaks.

## STRIDE Analysis

| Category | Threat | Hash Chain Mitigates? | Rationale |
|:--|:--|:--|:--|
| Tampering | Privileged user modifies audit_log content | Partially | Detects modification if verifier runs. But anyone with ClickHouse write access can rewrite all rows and recompute all hashes. |
| Tampering | Attacker deletes audit_log rows | Yes | Breaks the chain. Detected on next verification run. |
| Repudiation | Agent denies producing artifact | Partially | Proves artifact was stored at a point in time. Does not prove authorship (no cryptographic signature). |

## Why Defer

**1. No external anchor.** The hash chain is self-referential within ClickHouse. Rewriting the entire chain from entry 1 is undetectable without an off-system witness. Real tamper-evidence requires immutable external storage.

**2. ClickHouse eventual consistency.** Async replication means replicas may return incomplete chains during lag windows, causing false verification failures. The verification endpoint must define snapshot semantics — added complexity with no existing requirement driving it.

**3. Single-writer constraint.** Hash chains require serialized ordering. Multiple gateway replicas inserting concurrently create ambiguous chain ordering. Enforcing single-writer is an architectural constraint with operational cost.

**4. Cost exceeds value today.** The `audit_logs` table already uses `ReplacingMergeTree` with a content-addressed `audit_id`. This provides deduplication and idempotent writes. For "what did the agent produce," this is sufficient.

## When to Revisit

If Studio needs a tamper-evident audit trail that withstands privileged attackers, use a verifiable log system:

| Option | Mechanism | Strength |
|:--|:--|:--|
| [Trillian](https://github.com/google/trillian) | Merkle tree-based transparency log | Cryptographic proof of inclusion + consistency. External verifiers can audit without DB access. |
| WORM storage | Append-only S3 with Object Lock | Prevents deletion/modification at storage layer. No cryptographic proof of ordering. |
| Signed checkpoints | Periodic signed hash published to immutable store | Detects chain rewriting between checkpoints. Requires key management. |

**Recommended path:** Trillian. It solves the fundamental problem (external verifiable witness) that in-database hash chains cannot. Scope as a dedicated spec when a regulatory requirement demands it.

## Related

- [Agent Trust Model Deferred](trust-model-deferred.md) — companion decision
- [Agent Interaction Model](agent-interaction-model.md) — HITL chatbot model (all agents confirm with user)
