## 1. ClickHouse Schema

- [x] 1.1 Write DDL for `evaluation_logs` table (MergeTree, partitioned by `toYYYYMM(collected_at)`, sort key `(target_id, policy_id, control_id, collected_at)`, 24-month TTL)
- [x] 1.2 Write DDL for `enforcement_actions` table (MergeTree, partitioned by `toYYYYMM(started_at)`, sort key `(target_id, policy_id, control_id, started_at)`, 24-month TTL)
- [x] 1.3 Store DDL in a ConfigMap template (`charts/complytime-studio/templates/clickhouse-schema-configmap.yaml`)

## 2. ClickHouse Deployment

- [x] 2.1 Add ClickHouse StatefulSet template with PVC and init container that runs DDL from ConfigMap
- [x] 2.2 Add ClickHouse Service template (port 8123 for HTTP, 9000 for native)
- [x] 2.3 Add Kubernetes Secret template for ClickHouse credentials
- [x] 2.4 Add `clickhouse.enabled` conditional to all ClickHouse templates
- [x] 2.5 Add `clickhouse` section to `values.yaml` (image, storage size, credentials, retention months, enabled flag)

## 3. mcp-clickhouse Deployment

- [x] 3.1 Add mcp-clickhouse Deployment template referencing ClickHouse credentials Secret and `--readonly` mode
- [x] 3.2 Add mcp-clickhouse Service template
- [x] 3.3 Add `clickhouse-mcp` McpServer CRD template (conditional on `clickhouse.enabled`)
- [x] 3.4 Update Gap Analyst Agent CRD to include `type: McpServer` tool entry for `clickhouse-mcp` with tool filter (`run_select_query`, `list_databases`, `list_tables`)

## 4. Evidence Ingestion Loader

- [x] 4.1 Create `cmd/ingest/main.go` entry point accepting file path or stdin
- [x] 4.2 Implement Gemara YAML parsing and CUE validation (`#EvaluationLog`, `#EnforcementLog`)
- [x] 4.3 Implement EvaluationLog flattener (iterate ControlEvaluation → AssessmentLog, produce row structs)
- [x] 4.4 Implement EnforcementLog flattener (iterate ActionResult → AssessmentFinding, produce row structs)
- [x] 4.5 Implement ClickHouse writer using native protocol with idempotent insert logic
- [x] 4.6 Add `Dockerfile.ingest` for the loader binary
- [x] 4.7 Add `ingest-build` and `ingest-image` targets to Makefile

## 5. Gap Analyst Prompt Rewrite

- [x] 5.1 Rewrite `gap_analyst_prompt.md` to evidence-synthesis workflow (load criteria, query ClickHouse, classify, assemble AuditLog)
- [x] 5.2 Add classification table (L5 result × L6 disposition → AuditResult type)
- [x] 5.3 Add example ClickHouse queries for the agent to reference
- [x] 5.4 Document "no eval data" handling (missing requirement rows = Gap)
- [x] 5.5 Update MappingDocument to optional input for cross-framework enrichment

## 6. Orchestrator Routing Update

- [x] 6.1 Update `skills/orchestrator-routing/SKILL.md` Gap Analyst section with new required inputs (`policy_id`, `target_id`) and optional `MappingDocument`
- [x] 6.2 Update routing rules for evidence-backed audit vs document-only gap analysis

## 7. Integration Verification

- [ ] 7.1 Deploy ClickHouse + mcp-clickhouse via `helm upgrade` with `clickhouse.enabled=true`
- [ ] 7.2 Ingest a sample EvaluationLog and verify rows in ClickHouse
- [ ] 7.3 Ingest a sample EnforcementLog and verify rows in ClickHouse
- [ ] 7.4 Verify Gap Analyst can query ClickHouse via mcp-clickhouse
- [ ] 7.5 End-to-end: trigger an audit via orchestrator with a policy and target, verify AuditLog output
