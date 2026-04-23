## 1. Remove ClickHouse

- [ ] 1.1 Remove `internal/clickhouse/` package (client, connection logic)
- [ ] 1.2 Remove `internal/ingest/` package (writer.go, types.go)
- [ ] 1.3 Remove ClickHouse-backed store implementations from `internal/store/`
- [ ] 1.4 Remove `charts/complytime-studio/templates/clickhouse.yaml` (StatefulSet)
- [ ] 1.5 Remove `charts/complytime-studio/templates/clickhouse-schema-configmap.yaml`
- [ ] 1.6 Remove ClickHouse values from `charts/complytime-studio/values.yaml`
- [ ] 1.7 Remove `clickhouse-mcp` references from agent.yaml and Helm templates

## 2. PostgreSQL Knowledge Mesh

- [ ] 2.1 Add CloudNativePG Cluster CRD to Helm chart (or PostgreSQL StatefulSet for dev)
- [ ] 2.2 Create `internal/pg/` package with connection pool (`pgxpool`)
- [ ] 2.3 Create initial migration: Gemara knowledge graph tables with FK constraints (policies, control_catalogs, controls, assessment_requirements, threat_catalogs, threats, risk_catalogs, risks, mapping_documents, mapping_entries, catalogs)
- [ ] 2.4 Create migration: `rule_mappings` table (rule_id → control_id, requirement_id, plan_id)
- [ ] 2.5 Create migration: `assessment_plan_status` materialized table
- [ ] 2.6 Create migration: `ingestors` plugin registry table
- [ ] 2.7 Create migration: `gate_results` audit trail table
- [ ] 2.8 Implement migration runner in Gateway startup (idempotent, tracked in schema_migrations)
- [ ] 2.9 Rewrite `internal/store/` interfaces against PostgreSQL (PolicyStore, EvidenceStore, AuditLogStore, MappingStore)
- [ ] 2.10 Add JSONB content column to artifact tables, implement path query helpers

## 3. OpenSearch Indices

- [ ] 3.1 Add OpenSearch Operator CRD to Helm chart (or OpenSearch StatefulSet for dev)
- [ ] 3.2 Create `internal/opensearch/` package with client and index management
- [ ] 3.3 Create index template: `compliance-evidence` (FTS + keyword facets + date range + labels)
- [ ] 3.4 Create index template: `compliance-knowledge` (control/threat/guideline text + dense vector)
- [ ] 3.5 Implement index template application at Gateway startup
- [ ] 3.6 Implement evidence search API (`GET /api/evidence/search`) backed by OpenSearch

## 4. WASM Ingestor Runtime

- [ ] 4.1 Add `wazero` dependency to Gateway
- [ ] 4.2 Create `internal/wasm/` package with runtime initialization and module cache
- [ ] 4.3 Define WASM plugin interface: `metadata()` and `transform()` exports, `log()` host import
- [ ] 4.4 Define `EvidenceRow` and `EvidenceBatch` serialization format for WASM boundary (JSON or FlatBuffers)
- [ ] 4.5 Implement plugin loading: pull `.wasm` from OCI registry via ORAS, compile, validate exports, cache
- [ ] 4.6 Implement sandbox configuration: memory limits, execution timeout, WASI capability restriction
- [ ] 4.7 Implement `transform` invocation: pass input bytes to plugin, deserialize output to `EvidenceBatch`

## 5. Evidence Ingestion Pipeline

- [ ] 5.1 Create `internal/pipeline/` package with pipeline orchestrator
- [ ] 5.2 Implement `POST /api/evidence/ingest` endpoint (raw bytes + X-Ingestor + X-Policy-Id)
- [ ] 5.3 Implement compliance enrichment step: `rule_id` → PG `rule_mappings` lookup
- [ ] 5.4 Implement provenance gate: engine_name vs assessment plan executor, collected_at vs frequency window
- [ ] 5.5 Implement fan-out writer: PG (evidence_summary + assessment_plan_status) → OpenSearch (index) → Bulk Store (append)
- [ ] 5.6 Implement provenance tagging: stamp ingestor_name/version from WASM metadata on every row

## 6. Plugin Registry

- [ ] 6.1 Implement `GET /api/ingestors` endpoint listing available plugins from `ingestors` table
- [ ] 6.2 Implement `POST /api/ingestors/register` endpoint for explicit plugin registration by OCI reference
- [ ] 6.3 Implement lazy-pull: resolve plugin on first ingest request if not in registry

## 7. Bulk Store Adapter

- [ ] 7.1 Define `BulkStore` interface in `internal/store/` (Append, Query, Close)
- [ ] 7.2 Implement `OpenSearchBulkStore` (default) using bulk API against `compliance-evidence` index
- [ ] 7.3 Implement `ClickHouseBulkStore` (opt-in) with existing evidence table schema
- [ ] 7.4 Add `bulkStore.backend` Helm value (default: "opensearch") to select implementation
- [ ] 7.5 Wire bulk store adapter into MCP evidence query tool

## 8. Agent Updates

- [ ] 8.1 Replace `studio-clickhouse-mcp` with `studio-postgres-mcp` and `studio-evidence-mcp` in agent.yaml
- [ ] 8.2 Update `skills/studio-audit/SKILL.md` table reference for PG + OpenSearch
- [ ] 8.3 Update `skills/posture-check/SKILL.md` to read from `assessment_plan_status` table
- [ ] 8.4 Update `agents/assistant/prompt.md` schema discovery section for new backends
- [ ] 8.5 Run `make sync-skills && make sync-prompts`

## 9. Helm Chart

- [ ] 9.1 Add PostgreSQL deployment (CloudNativePG or StatefulSet) with PVC
- [ ] 9.2 Add OpenSearch deployment (Operator or StatefulSet) with PVC
- [ ] 9.3 Add MCP server CRDs for `studio-postgres-mcp` and `studio-evidence-mcp`
- [ ] 9.4 Update Gateway environment variables (PG connection, OpenSearch URL, bulk store backend)
- [ ] 9.5 Update `docker-compose.yaml` for local dev with PG + OpenSearch

## 10. Verification

- [ ] 10.1 Build and start stack with `docker compose up`
- [ ] 10.2 Register a test ingestor plugin (mock .wasm)
- [ ] 10.3 Ingest evidence via `POST /api/evidence/ingest` with the test plugin
- [ ] 10.4 Verify evidence appears in PostgreSQL (evidence_summary, assessment_plan_status)
- [ ] 10.5 Verify evidence is searchable in OpenSearch (label search, FTS)
- [ ] 10.6 Verify agent posture check reads from `assessment_plan_status`
- [ ] 10.7 Verify agent audit production queries work against new backends
