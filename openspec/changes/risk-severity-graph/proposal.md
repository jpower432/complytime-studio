## Why

Evidence queries cannot answer "what is the risk severity for this failed control?" without manual lookup. The Gemara layer model defines the full relationship chain — Policy→Risk, Policy→Control, Control mitigates Threat, Risk links to Threat — but the ClickHouse schema has no `risks` or `risk_threats` tables. The agent must be able to derive risk severity through joins, not denormalized columns.

## What Changes

- Add `risks` table (from `RiskCatalog.Risks[]`) with `severity` column
- Add `risk_threats` junction table (from `Risk.Threats[]`) linking risks to threats
- Add parser for `RiskCatalog` YAML → structured rows
- Add `PopulateRisks` startup backfill (same pattern as `PopulateControls`/`PopulateThreats`)
- Add `/api/catalogs/import` support for `RiskCatalog` type detection and structured row extraction
- Update `evidence-schema` skill with new table DDL and risk severity traversal query pattern
- **BREAKING**: Drop denormalized `frameworks` column from `evidence` table (migration v2, already in progress)

## Capabilities

### New Capabilities
- `risk-index-tables`: ClickHouse `risks` and `risk_threats` tables, parser, backfill, and import handler support
- `risk-severity-query`: Agent query pattern for deriving risk severity from evidence through the control→threat→risk join path

### Modified Capabilities
- `graph-traversal-queries`: Add risk severity traversal pattern to the evidence-schema skill

## Impact

- `internal/clickhouse/client.go` — DDL for `risks` and `risk_threats` tables
- `internal/gemara/` — New `ParseRiskCatalog` function
- `internal/store/populate.go` — `PopulateRisks` backfill
- `internal/store/handlers.go` — `RiskCatalog` detection in `importCatalogHandler`
- `skills/evidence-schema/SKILL.md` — New table docs and query patterns
- `charts/complytime-studio/samples/` — May need a sample `RiskCatalog` for seed job
