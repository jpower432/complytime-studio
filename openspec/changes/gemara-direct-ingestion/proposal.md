# Proposal: Direct Gemara-Based Evidence Ingestion

## User Story

As a compliance engineer, I need to push EvaluationLog and EnforcementLog artifacts directly to Studio's API so that evidence flows through the certification pipeline without requiring an OTel Collector deployment.

## Problem

Evidence ingestion has two paths with different trade-offs:

| Path | Owner | Issue |
|:--|:--|:--|
| OTel Collector → ClickHouse exporter | Operator infrastructure | Requires deploying, configuring, and maintaining a collector per environment. Semconv attribute mapping adds a translation layer between Gemara types and ClickHouse columns. |
| `cmd/ingest` binary | Studio | CLI-only. No API. Not integrated with the gateway's certification pipeline event trigger. |
| `POST /api/evidence` (JSON) | Gateway | Accepts flat JSON records, not Gemara artifacts. Caller must flatten before posting. |

None of these accept a Gemara artifact and produce certified evidence rows in a single API call.

## Solution

Add `POST /api/evidence/import` to the gateway. Accepts Gemara EvaluationLog or EnforcementLog YAML. The handler:

1. Detects artifact type from `metadata.type`
2. Loads and validates via `go-gemara` SDK
3. Flattens to evidence rows (existing `internal/ingest` logic)
4. Inserts into ClickHouse
5. Publishes NATS event → certification pipeline runs asynchronously

The OTel Collector path remains valid for operators who want it — Studio keeps the semconv alignment doc as a reference contract — but Studio no longer treats it as the primary ingestion path.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| `POST /api/evidence/import` gateway handler | Removing OTel semconv alignment doc (kept as reference) |
| Move `internal/ingest` flatten logic into gateway handler | Modifying OTel Collector config |
| Gemara SDK validation before insert | New artifact types beyond EvaluationLog / EnforcementLog |
| NATS event publish → certification pipeline | Changing certification pipeline logic |
| Supersede `cmd/ingest` binary (deprecate) | Removing `cmd/ingest` immediately (deprecate first) |
| Update ADR: primary path is now API push | |
