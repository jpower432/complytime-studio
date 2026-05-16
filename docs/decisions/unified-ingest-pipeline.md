# ADR #0034 — Unified Ingest Pipeline

**Status:** Accepted
**Date:** 2026-05-16
**Supersedes:** Partially supersedes #0028 (Async Evidence Ingest) scope

## Context

The gateway has five content entry points that accept Gemara artifacts:

| # | Path | Types | Mode |
|---|------|-------|------|
| 1 | `PopulateCatalogsFromRegistry` (startup) | ControlCatalog, ThreatCatalog | Sync, auto |
| 2 | `POST /api/import` (OCI ref body) | Policy, Mapping, all Catalogs | Sync |
| 3 | `POST /api/import` (raw YAML body) | Policy, Mapping, all Catalogs | Sync |
| 4 | `POST /api/evidence/ingest` | EvaluationLog, EnforcementLog | Sync |
| 5 | `POST /api/evidence/ingest/async` | EvaluationLog, EnforcementLog | Async (NATS) |

Problems:

1. **Parallel paths:** Catalogs/policies import synchronously in the request handler. Evidence has both sync and async paths. Type detection and storage logic is duplicated across `handlers_import.go` and `ingest_handler.go`.
2. **Hardcoded seed:** `PopulateCatalogsFromRegistry` has a hardcoded repo list and only handles two catalog types. It duplicates what `/api/import` already does.
3. **Inconsistent async:** Evidence uses NATS for async processing; policies and catalogs do not. The async infrastructure exists but only covers half the artifact types.
4. **Deployment complexity:** Docker Compose requires separate `registry-seed` and `gateway-seed` containers feeding different code paths to populate the same database.

## Decision

Collapse all content ingestion into a single async pipeline:

```
POST /api/ingest  →  accept raw content  →  assign job ID  →  NATS ingest.artifact  →  worker
```

The worker:
1. Detects artifact type via `metadata.type`
2. Routes to the appropriate storage function (evidence flatten, policy insert, catalog parse, mapping parse)
3. Updates job status tracker
4. Publishes downstream events (e.g., `studio.evidence.<policy_id>` for certifier)

### Single Endpoint Contract

```
POST /api/ingest
Content-Type: application/x-yaml | application/json

→ 202 Accepted {"job_id": "<uuid>", "status": "pending"}
```

```
GET /api/ingest/jobs/{job_id}

→ 200 {"job_id": "...", "status": "completed|failed|pending", "result": {...}}
```

### OCI Reference Support

OCI bundle import remains a separate concern — it resolves an OCI reference into individual artifact YAMLs, then feeds each one into the same ingest pipeline. Implemented as a thin wrapper:

```
POST /api/import
{"reference": "registry:5000/org/bundle:tag"}

→ Pulls bundle → publishes each artifact to NATS ingest.artifact → returns job IDs
```

This keeps OCI resolution (network I/O, authentication) at the API boundary while reusing the unified worker for all parsing and storage.

### NATS Subject Consolidation

| Current | New |
|---|---|
| `studio.ingest.raw` (evidence only) | `core.ingest` (all artifact types) |
| `studio.evidence.<policy_id>` (post-insert) | `core.evidence.<policy_id>` (unchanged role) |
| `studio.draft-audit-log.<policy_id>` | `core.draft.<policy_id>` (unchanged role) |

The `core.ingest` subject carries all artifact types. The worker determines the type and routes internally.

### Startup Seed

`PopulateCatalogsFromRegistry` is removed. Seeding is handled by a single seed job (container or init script) that posts artifacts to `POST /api/ingest` like any other client. The gateway has no hardcoded knowledge of seed repos.

## Consequences

**Positive:**
- One ingestion path for all Gemara content — policies, catalogs, evidence, mappings
- Async by default; consistent job tracking for all artifact types
- Seed is just another API client — no special startup logic
- Docker Compose collapses from two seed containers to one
- Worker is the single place to add new artifact type support

**Negative:**
- All ingestion is async — callers must poll for completion (already solved: job status endpoint + notification events)
- OCI import adds a thin layer on top rather than being eliminated entirely (justified: OCI resolution is network-bound and benefits from being at the API edge)

## Affected Code

| File | Change |
|------|--------|
| `internal/store/handlers_import.go` | Remove `rawBodyImport`, `importPolicyFromArtifactBody`, `importCatalogFromArtifactBody`. Keep OCI import as wrapper. |
| `internal/store/ingest_handler.go` | Remove `IngestGemaraHandler` (sync). Rename `IngestAsyncHandler` to `IngestHandler`. Expand to accept all artifact types. |
| `internal/store/ingest_worker.go` | Expand `IngestWorker` to handle Policy, Catalog, Mapping in addition to EvaluationLog/EnforcementLog. |
| `internal/store/populate.go` | Remove `PopulateCatalogsFromRegistry` and `defaultSeedCatalogs`. |
| `internal/store/handlers.go` | Remove `/evidence/ingest` sync route. Promote `/evidence/ingest/async` to `/ingest`. |
| `internal/events/nats.go` | Rename subjects to `core.*` namespace. Remove `SubjectEvidence` alias. |
| `deploy/compose/docker-compose.yaml` | Single `seed` container replaces `registry-seed` + `gateway-seed`. |
