## Why

Gemara artifacts form an implicit graph through `#ArtifactMapping`, `#EntryMapping`, and `#MultiEntryMapping` cross-references. Today, only `mapping_entries` and `policy_contacts` are materialized as structured ClickHouse rows — the rest (controls, threats, risks, guidance) live as raw YAML in the `policies` table `content` column. Graph traversals like "which framework objectives are affected if Threat T-3 is reclassified?" require sequential YAML parsing across multiple artifacts. Materializing the remaining Gemara layers into ClickHouse index tables enables SQL-native impact analysis, coverage queries, and gap detection without YAML parsing at query time.

## What Changes

- **New `controls` table**: Parsed L2 ControlCatalog entries — one row per control with objective, group, lifecycle state, and parent catalog reference. Populated at policy import time from imported catalog YAML.
- **New `assessment_requirements` table**: Parsed L2 assessment requirements — one row per requirement with parent control, applicability tags, and lifecycle state. Enables JOIN from evidence directly to requirement text.
- **New `threats` table**: Parsed L2 ThreatCatalog entries — one row per threat with group and description. Populated at import time.
- **New `control_threats` table**: Junction table linking controls to threats via their `threats[]` cross-references. Enables "threat → control → evidence" traversals.
- **New parsers in `internal/gemara/`**: `ParseControls`, `ParseAssessmentRequirements`, `ParseThreats`, `ParseControlThreats` functions following the established `go-gemara` + `goccy/go-yaml` pattern.
- **Parse-at-ingest in gateway**: Extend policy and catalog import handlers to parse and insert structured rows, same pattern as `mapping_entries` and `policy_contacts`.
- **New query patterns in evidence-schema skill**: Impact traversal, coverage completeness, and gap detection queries using the new tables.
- **Retroactive population**: Populate new tables from existing raw YAML on startup, same pattern as `PopulatePolicyContacts`.

## Capabilities

### New Capabilities
- `control-index-tables`: ClickHouse DDL for `controls`, `assessment_requirements`, `control_threats` tables; parse-at-ingest for ControlCatalog content
- `threat-index-tables`: ClickHouse DDL for `threats` table; parse-at-ingest for ThreatCatalog content
- `graph-traversal-queries`: SQL query patterns for multi-hop traversals (threat → control → evidence → framework)

### Modified Capabilities
- `gemara-parsing`: New parse functions (`ParseControls`, `ParseAssessmentRequirements`, `ParseThreats`, `ParseControlThreats`) added to `internal/gemara/`
- `evidence-ingestion`: Import handlers extended to parse and store structured rows for controls and threats alongside existing mapping/contact parsing

## Impact

- **Backend**: `internal/gemara/` — new parser files. `internal/store/` — new store interfaces and insert methods. `internal/store/handlers.go` — extended import handlers. `internal/clickhouse/client.go` — new DDL statements.
- **Schema**: `charts/complytime-studio/templates/clickhouse-schema-configmap.yaml` — new table DDL.
- **Skills**: `skills/evidence-schema/SKILL.md` — new table docs and query patterns.
- **No frontend changes**: Tables are consumed by the assistant via clickhouse-mcp `run_select_query`.
- **No API changes**: Ingest happens within existing import endpoints.
