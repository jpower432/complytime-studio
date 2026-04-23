## Context

Studio is a prototype. ClickHouse was chosen for speed-to-demo, not architectural fitness. It stores relational Gemara artifacts, time-series evidence, and full YAML documents in a single columnar analytics engine. The `risk-severity-graph` change exists solely because the agent needs a four-hop traversal that ClickHouse makes painful. Evidence ingestion is hardcoded to JSON/CSV formats. Compliance enrichment lives in an external OTel processor. Evidence quality is discovered at audit time, not ingest time.

Customer environments run OpenShift with PostgreSQL (always on the allowed list) and OpenSearch (deployed for logging). ClickHouse is neither.

## Goals / Non-Goals

**Goals:**
- Replace ClickHouse with purpose-built stores: PostgreSQL (knowledge), OpenSearch (search + bulk default), optional ClickHouse Operator (analytical scale)
- Customer-extensible evidence ingestion via WASM plugins — no redeployment for new scanner formats
- Compliance enrichment and provenance validation at ingest time, not query time
- Materialized posture status in PostgreSQL — agent reads a table instead of computing in-prompt
- Evidence search by label, full-text, and vector similarity via OpenSearch

**Non-Goals:**
- Migration tooling from ClickHouse — clean break, prototype data is disposable
- Multi-tenancy — single-tenant deployment model unchanged
- WASM plugin authoring SDK or CLI — out of scope, plugins are compiled with standard WASI toolchains
- Real-time streaming ingestion (Kafka, NATS) — batch/request-driven ingestion is sufficient for now
- Phase 2 autonomous monitoring agent — separate proposal

## Decisions

### D1: PostgreSQL as knowledge mesh, not just a relational store

**Choice:** Model the full Gemara artifact graph in PostgreSQL with proper foreign keys, JSONB for raw artifact content, and materialized views for posture status.

**Why:** The Gemara model is a directed graph: Guidance→Controls→Threats→Risks, joined by MappingDocuments, governed by Policies. PostgreSQL handles this natively with FK constraints (referential integrity), recursive CTEs (multi-hop traversal in one query), and JSONB (query into raw artifact YAML without agent-side parsing). The `risk-severity-graph` change becomes a single CTE instead of four new junction tables.

**Alternative:** Keep ClickHouse, add PG alongside. Rejected — dual-write complexity for a prototype with no production data to preserve.

### D2: wazero for WASM runtime (pure Go, no CGo)

**Choice:** Use `wazero` as the WASM runtime embedded in the Gateway.

**Why:** wazero is a pure Go WebAssembly runtime — zero CGo dependencies, compiles cleanly with the existing Go toolchain, supports WASI preview 1. The Gateway is already Go. Embedding the runtime avoids a new sidecar or service. WASI P1 is sufficient for the `transform(bytes) → EvidenceRow[]` contract — the plugin receives a memory buffer and returns serialized output. No filesystem, network, or clock access needed.

**Alternative:** wasmtime (Rust/C, requires CGo). Rejected — adds CGo cross-compilation complexity to the Gateway build. Performance difference is negligible for the transform workload (sub-second, small payloads).

**Alternative:** Dedicated WASM sidecar pod. Rejected for v2 — adds latency (network hop) and deployment complexity. Revisit if plugin execution becomes CPU-intensive.

### D3: Plugins are pure format transformers, not compliance-aware

**Choice:** WASM plugins only transform raw scanner output into semconv-aligned `EvidenceRow[]`. They do NOT perform compliance mapping (`policy_id`, `control_id`, `requirement_id`). Compliance enrichment is the host's responsibility, using the PostgreSQL knowledge graph.

**Why:** Separates concerns cleanly. Plugin authors only need to understand their scanner's output format — not Gemara. Compliance mapping logic is centralized in one place (the enrichment pipeline), not scattered across N plugins. A rule-to-control mapping change updates one table in PG, not every plugin.

**Alternative:** Plugins emit fully enriched rows. Rejected — couples plugin authoring to Gemara schema knowledge, creates N copies of mapping logic, makes mapping updates require plugin rebuilds.

### D4: OCI registry for plugin distribution

**Choice:** WASM ingestor modules are OCI artifacts stored in the same registry used for Gemara bundles. The Gateway discovers and pulls them via ORAS (existing infrastructure).

**Why:** Reuses existing OCI registry infrastructure (ORAS MCP, in-cluster Zot for dev, customer registries for prod). No new distribution mechanism. Plugin versioning follows OCI tag conventions. Customers publish plugins to their own registry.

**Alternative:** ConfigMap-based plugin storage. Rejected — size limits (1MB), no versioning, poor developer experience.

### D5: OpenSearch as default bulk store, ClickHouse as opt-in

**Choice:** OpenSearch serves as both search index AND default bulk evidence store. ClickHouse is offered via Operator for customers who need sub-second analytical aggregation over large evidence volumes.

**Why:** OpenSearch is already deployed in OpenShift environments for logging. Adding a compliance index is marginal effort for the customer. Using it for bulk storage eliminates one component from the default deployment. ClickHouse Operator is available for customers at scale — they opt in, they operate it.

**Alternative:** ClickHouse as default, OpenSearch as opt-in search layer. Rejected — ClickHouse is not on most customer "allowed" lists and adds operational burden.

### D6: Fan-out write with inline provenance gate

**Choice:** The ingestion pipeline writes to all three stores in a single request path: PG (knowledge summary + gate results), OpenSearch (searchable evidence), Bulk Store (raw volume). Provenance validation happens inline before writes — each row is tagged with `gate_status` (accepted/flagged/rejected).

**Why:** Ingest-time validation is the core insight from the posture-check exploration. Evidence quality problems should surface when evidence arrives, not when an auditor asks. The gate checks `engine_name` against the assessment plan's `executor.id`, validates `collected_at` against frequency windows, and flags mismatches. The posture-check agent skill becomes a simple `SELECT * FROM assessment_plan_status` instead of a multi-query computation.

**Alternative:** Async enrichment via message queue. Rejected for v2 — adds infrastructure (queue), eventual consistency complexity, harder to debug. Revisit if ingest throughput requires decoupling.

### D7: Enrichment replaces truthbeam OTel processor

**Choice:** Compliance enrichment (rule→control→requirement mapping) moves from the external OTel `truthbeam` processor into the Gateway's ingestion pipeline, backed by PostgreSQL lookups.

**Why:** Enrichment needs the knowledge graph. The knowledge graph is in PostgreSQL. Doing enrichment in the application layer (Gateway) gives direct access to the graph, structured error handling, and audit-grade provenance tracking. The OTel Collector becomes optional infrastructure — useful for streaming telemetry but not required for evidence ingestion.

**Alternative:** Keep truthbeam, feed it from PG instead of static config. Rejected — adds latency (PG→truthbeam round trip), keeps enrichment logic outside the application boundary.

### D8: Materialized assessment_plan_status in PostgreSQL

**Choice:** Maintain a materialized table `assessment_plan_status` in PostgreSQL, updated on every evidence write. Columns: `policy_id`, `plan_id`, `target_id`, `last_evidence_at`, `source_match`, `cadence_status`, `latest_result`, `classification` (Healthy/Failing/Wrong Source/Stale/Blind).

**Why:** The posture-check skill currently computes this by parsing Policy YAML and querying evidence per-plan. Materializing it means the agent (or the UI dashboard) reads one table. Updates happen at ingest time as part of the fan-out write — zero additional cost.

**Alternative:** Keep as agent-computed. Rejected — unreliable (LLM YAML parsing), slow (N queries per check), not available to the UI without agent involvement.

## Risks / Trade-offs

**[Risk] PostgreSQL write throughput under heavy evidence ingest** → Evidence summary upserts are lightweight (one row per policy+plan+target combination). Raw volume goes to OpenSearch/ClickHouse. PG handles the knowledge graph and materialized views, not bulk evidence. Acceptable for expected scale.

**[Risk] wazero WASI P1 limitations** → WASI P1 lacks component model (typed interfaces). The plugin contract uses serialized bytes across the WASM boundary (JSON or FlatBuffers). Functional but less ergonomic than component model. Acceptable — migrate to WASI P2 when wazero supports it.

**[Risk] OpenSearch as bulk store may underperform ClickHouse for analytical queries** → Mitigated by offering ClickHouse Operator as opt-in. Default path optimizes for deployment simplicity over query performance. Customers who need `GROUP BY` over billions of rows self-select into the ClickHouse path.

**[Risk] Fan-out write failure partial consistency** → If PG write succeeds but OpenSearch write fails, stores diverge. Mitigated by: (1) PG is source of truth for posture, (2) OpenSearch is eventually consistent by design, (3) bulk store append is idempotent. Retry with dead-letter for failed writes.

**[Trade-off] Three stores vs one** → More operational surface. Mitigated by: PG and OpenSearch are already in customer environments. ClickHouse is opt-in. The default deployment adds PG (universally supported) and an OpenSearch index (marginal on existing deployment).
