# 0028 — Async Evidence Ingest: Accept-the-Loss Durability

**Status:** Accepted
**Date:** 2026-05-13

## Context

The gateway now supports async evidence ingest (`POST /api/evidence/ingest/async`). Raw Gemara YAML is published to NATS for background processing. The HTTP handler returns `202 Accepted` with a `job_id` for polling.

Three durability tiers were evaluated:

| Tier | Mechanism | Guarantees | Complexity |
|:--|:--|:--|:--|
| 1 | NATS core pub/sub + in-memory tracker | At-most-once; state lost on restart | Minimal |
| 2 | NATS JetStream | At-least-once; replay on consumer restart | Moderate — requires stream/consumer config |
| 3 | Postgres WAL-backed outbox | Exactly-once via transactional outbox pattern | High — polling, deduplication, new tables |

## Decision

**Tier 1: accept-the-loss.** NATS core pub/sub with in-memory job tracking.

- If the gateway restarts mid-flight, pending jobs and their YAML payloads are lost.
- Callers can re-submit. The sync endpoint (`POST /api/evidence/ingest`) remains available as a fallback with guaranteed persistence.
- Job status is tracked in a Go `sync.Map`-style structure, not persisted to Postgres.

## Rationale

- The sync ingest path already exists and is fully durable. Async is an optimization for non-blocking submission, not a replacement.
- JetStream adds operational complexity (stream provisioning, consumer management, retention policies) disproportionate to the current user base.
- The Postgres outbox pattern requires new migration, polling loop, and deduplication logic.
- This decision is explicitly reversible: upgrading to JetStream requires only swapping `Publish` for `PublishMsg` with a stream subject, and replacing the in-memory tracker with a Postgres table.

## Consequences

- **Positive:** Zero new infrastructure. No new migrations. Minimal code surface.
- **Negative:** Job state is ephemeral. Monitoring must account for lost jobs after restarts.
- **Migration path:** When durability becomes a requirement, evaluate JetStream first (lower lift than outbox). Open a new ADR at that point.
