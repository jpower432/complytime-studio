## Why

Studio's data layer is a ClickHouse monolith. It stores relational Gemara artifacts (policies, controls, threats), time-series evidence, and full YAML blobs in the same engine. Every traversal (risk→threat→control→evidence) requires the LLM to compose multi-hop JOINs. Adding new evidence formats requires Go code changes and a redeployment. Compliance enrichment lives in an external OTel processor (`truthbeam`), decoupled from the knowledge graph. Evidence quality is only checked at query time, not ingest time.

This is a clean-break redesign of the data architecture into purpose-built layers:

- **PostgreSQL** for the Gemara knowledge graph (relational, FK-enforced, CTE-traversable)
- **OpenSearch** for evidence search and semantic matching (already in customer OpenShift stacks)
- **WASM-based evidence ingestors** for customer-extensible, sandboxed, polyglot format transformation
- **Pluggable bulk store** (OpenSearch default, ClickHouse Operator for scale) for time-series evidence volume

## What Changes

- **Remove ClickHouse** as the sole datastore. Remove `internal/clickhouse/`, `internal/ingest/`, ClickHouse-backed store implementations, Helm chart StatefulSet, schema ConfigMap, and MCP dependency.
- **Add PostgreSQL knowledge mesh** — Gemara artifact graph with FK-enforced referential integrity, JSONB for raw artifact content, materialized `assessment_plan_status` for real-time posture, compliance enrichment via SQL lookups.
- **Add OpenSearch indices** — `compliance-evidence` (FTS, facets, label search, vector embeddings) and `compliance-knowledge` (semantic search across control objectives, assessment requirements, threat descriptions).
- **Add WASM ingestor runtime** — `wazero`-based plugin host in the Gateway. Plugins are `.wasm` OCI artifacts pulled from the same registry used for Gemara bundles. Pure format transformation: raw bytes in, semconv-aligned `EvidenceRow[]` out.
- **Add ingestion pipeline** — Gateway accepts raw evidence + ingestor hint, dispatches to WASM plugin, enriches output against PG knowledge graph, validates provenance against assessment plans, fan-out writes to all three stores.
- **Add pluggable bulk store** — OpenSearch as default (simpler ops), ClickHouse Operator as opt-in for customers needing analytical queries over large evidence volumes.
- **Update agent MCP tools** — Replace `clickhouse-mcp` with `postgres-mcp` for knowledge queries and `opensearch-mcp` (or unified `evidence-mcp`) for evidence search.

## Capabilities

### New Capabilities
- `postgresql-knowledge-mesh`: PostgreSQL schema for the Gemara knowledge graph — policies, controls, threats, risks, assessment plans, mappings, materialized posture status. Replaces all ClickHouse relational tables.
- `opensearch-indices`: OpenSearch index templates for evidence search (FTS + vector) and knowledge search (semantic matching on Gemara artifact text).
- `wasm-ingestor-runtime`: wazero-based WASM plugin host in the Gateway. Loads `.wasm` ingestor modules from OCI registry, executes in capability-restricted sandbox, returns semconv-aligned evidence rows.
- `wasm-plugin-contract`: WASM plugin interface definition — `metadata()` and `transform(bytes)` exports, `log()` host import, `EvidenceRow` output schema, `IngestorMetadata` struct.
- `evidence-ingestion-pipeline`: Fan-out write path from WASM output through compliance enrichment, provenance validation (gate), and writes to PG + OpenSearch + Bulk Store.
- `plugin-registry`: OCI-based plugin discovery and lifecycle — pull, compile, cache, version management for `.wasm` ingestor modules.
- `bulk-store-adapter`: Abstraction layer for the pluggable bulk store (OpenSearch or ClickHouse). Evidence analytical queries route through this adapter.

### Modified Capabilities
- `agent-spec-skills`: Agent skill list and MCP tool references updated for PostgreSQL and OpenSearch backends. `clickhouse-mcp` replaced.

## Impact

- `internal/clickhouse/` — removed
- `internal/ingest/` — removed (replaced by WASM pipeline)
- `internal/store/` — rewritten against PostgreSQL + OpenSearch
- `internal/pg/` — new: PostgreSQL client, migrations, knowledge graph queries
- `internal/opensearch/` — new: OpenSearch client, index management, search queries
- `internal/wasm/` — new: wazero runtime, plugin loading, sandbox management
- `internal/pipeline/` — new: enrichment, provenance gate, fan-out writer
- `charts/complytime-studio/` — ClickHouse resources removed, CloudNativePG Cluster + OpenSearch Operator CRDs added
- `agents/assistant/` — MCP tool references updated, skills updated for new query patterns
- `skills/studio-audit/SKILL.md` — table reference updated for PG + OS
- `skills/posture-check/SKILL.md` — simplified (reads materialized posture from PG)
