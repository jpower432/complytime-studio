## 1. ClickHouse Schema Merge

- [x] 1.1 Write DDL for single `evidence` table (ReplacingMergeTree, partitioned by `toYYYYMM(collected_at)`, sort key `(target_id, policy_id, control_id, collected_at, row_key)`, 24-month TTL)
- [x] 1.2 Replace `evaluation_logs` and `enforcement_actions` DDL in `clickhouse-schema-configmap.yaml` with `evidence` table DDL
- [x] 1.3 Verify `helm template` renders the new schema with `clickhouse.enabled=true`

## 2. Semconv Alignment Documentation

- [x] 2.1 Create attribute mapping table: every `beacon.evidence` semconv attribute → ClickHouse `evidence` column name and type
- [x] 2.2 Document the five proposed semconv additions (`compliance.policy.id`, `compliance.assessment.requirement.id`, `compliance.assessment.plan.id`, `compliance.assessment.confidence`, `compliance.assessment.steps`) with types, groups, and rationale
- [ ] 2.3 Open issue or PR on `complytime/complytime-collector-components` proposing the new attributes in `model/attributes.yaml` **(external — not completed during apply, create issue)**

## 3. OTel Collector Helm Templates

- [x] 3.1 Add `otel.enabled` flag to `values.yaml` with collector image, resource limits, and ClickHouse exporter config
- [x] 3.2 Create collector Deployment template (conditional on `otel.enabled`)
- [x] 3.3 Create collector Service template exposing OTLP gRPC (4317) and HTTP (4318) ports
- [x] 3.4 Create collector ConfigMap with pipeline configuration: OTLP receivers → batch processor → ClickHouse exporter
- [ ] 3.5 Validate ClickHouse exporter supports custom table schema; if not, document workaround (processor reshaping or custom exporter) **(external research — not completed during apply, create issue)**

## 4. Update `cmd/ingest` for Merged Table

- [x] 4.1 Update EvaluationLog flattener to produce rows matching the `evidence` table schema (nullable remediation columns)
- [x] 4.2 Update EnforcementLog flattener to produce rows with co-located eval + remediation columns
- [x] 4.3 Set `enrichment_status` to `Success` on all ingested rows
- [x] 4.4 Populate new columns (`engine_name`, `target_type`, `risk_level`, `frameworks`, `requirements`) from Gemara YAML where data is available, NULL otherwise
- [x] 4.5 Remove old table-specific insert logic (`evaluation_logs`, `enforcement_actions`)

## 5. Gap-Analyst Prompt Update

- [x] 5.1 Replace two-table example queries with single-table `evidence` queries
- [x] 5.2 Update column references in the prompt to match new schema names
- [x] 5.3 Remove cross-table correlation instructions (no longer needed)

## 6. Deployment Documentation

- [x] 6.1 Add ClickHouse + OTel section to README covering the evidence pipeline
- [x] 6.2 Document gateway topology (default Helm deployment)
- [x] 6.3 Document agent topology (sidecar pattern for co-located producers)
- [x] 6.4 Document direct topology (local collector for development)
- [x] 6.5 Document `cmd/ingest` as the local testing path (no OTel stack required)

## 7. Integration Verification

- [ ] 7.1 Deploy ClickHouse + OTel Collector via `helm upgrade` with `otel.enabled=true` **(manual — requires deployed cluster)**
- [ ] 7.2 Send a sample OTLP log record with `beacon.evidence` attributes to the collector; verify row in `evidence` table **(manual — requires deployed cluster)**
- [ ] 7.3 Ingest a sample EvaluationLog via `cmd/ingest`; verify row in `evidence` table with NULL remediation columns **(manual — requires deployed cluster)**
- [ ] 7.4 Ingest a sample EnforcementLog via `cmd/ingest`; verify row in `evidence` table with co-located eval + remediation **(manual — requires deployed cluster)**
- [ ] 7.5 Verify gap-analyst can query the `evidence` table via `mcp-clickhouse` with single-table queries **(manual — requires deployed cluster)**
