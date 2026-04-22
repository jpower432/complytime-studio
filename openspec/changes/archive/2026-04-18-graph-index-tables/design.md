## Context

ClickHouse stores six tables today: `evidence`, `policies`, `mapping_documents`, `mapping_entries`, `policy_contacts`, and `audit_logs`. The `mapping_entries` and `policy_contacts` tables are "index tables" — structured rows parsed from raw Gemara YAML at import time by `internal/gemara/` parsers, then inserted by `internal/store/handlers.go`. This pattern enables SQL-native JOINs without YAML parsing at query time.

Gemara L2 artifacts (ControlCatalog, ThreatCatalog) and their cross-references are **not** yet materialized. The assistant currently reads raw YAML content from the `policies` table and parses it in-context. This is slow, token-expensive, and prevents multi-hop SQL traversals like "threat → control → evidence → framework."

The `internal/gemara/` package already provides `ParsePolicyContacts`, `ParseMappingEntries`, and `ParseAuditLog` using `go-gemara` types and `goccy/go-yaml`. New parsers follow the identical pattern.

## Goals / Non-Goals

**Goals:**
- Materialize L2 ControlCatalog entries (controls, assessment requirements, control-threat cross-references) into ClickHouse index tables
- Materialize L2 ThreatCatalog entries into a ClickHouse index table
- Follow the established parse-at-ingest and backfill-on-startup patterns
- Enable SQL-native graph traversals: threat → control → evidence, control → requirement → evidence, framework → mapping → control → threat
- Update the `evidence-schema` skill with new table schemas and query patterns

**Non-Goals:**
- L1 GuidanceCatalog materialization (deferred — lower query priority)
- L3 RiskCatalog materialization (deferred — depends on L2 tables being stable first)
- New REST API endpoints (tables are query-only via clickhouse-mcp `run_select_query`)
- Frontend changes (assistant consumes tables directly)
- Schema migrations for existing tables

## Decisions

### Decision 1: Four new tables, not one denormalized table

**Choice:** Create `controls`, `assessment_requirements`, `threats`, and `control_threats` as separate tables.

**Rationale:** Matches the Gemara artifact structure (controls contain assessment-requirements, controls reference threats). Separate tables enable targeted JOINs and avoid N*M row explosion. The `control_threats` junction table captures the many-to-many relationship between controls and threats.

**Alternative considered:** Single denormalized `catalog_entries` table with a `type` discriminator column. Rejected because the column sets are different (controls have `objective`, `group`, `state`; threats have `description`, `group`, `capabilities`), and the junction relationship between controls and threats requires a separate table regardless.

### Decision 2: Catalog ID as partition key, not policy ID

**Choice:** Use `catalog_id` (from `metadata.id`) as the primary identifier, with `policy_id` tracked optionally via the policy that imported the catalog.

**Rationale:** A ControlCatalog or ThreatCatalog can be referenced by multiple policies. The catalog is the source of truth for its entries. `policy_id` is stored for provenance (which policy import brought it in) but the catalog's own `metadata.id` is the primary key.

**Alternative considered:** Using `policy_id` as the partition key. Rejected because the same catalog (e.g., a CNSC control catalog) can be imported by multiple policies, creating duplicate rows partitioned by different policy IDs.

### Decision 3: Reuse `go-gemara` types directly

**Choice:** New parsers in `internal/gemara/` unmarshal into `go-gemara` types (`gemara.ControlCatalog`, `gemara.ThreatCatalog`) then extract flat rows for insertion.

**Rationale:** Matches the pattern established by `ParseAuditLog` which unmarshals into `gemara.AuditLog`. Keeps parsing logic minimal — the heavy lifting is done by `go-gemara`'s struct definitions and `goccy/go-yaml`.

### Decision 4: Parse catalogs at policy-import time

**Choice:** Extend `importPolicyHandler` (and add a new `/api/catalogs/import` handler if needed) to parse ControlCatalog and ThreatCatalog YAML at ingest time, same as contacts and mappings.

**Rationale:** The existing pattern in `importPolicyHandler` already parses contacts on import. Catalogs referenced by a policy can be imported alongside the policy or independently. Either way, parsing happens once at ingest, not at query time.

### Decision 5: Backfill via Populate functions on startup

**Choice:** Add `PopulateControls`, `PopulateThreats` functions in `internal/store/populate.go` following the `PopulatePolicyContacts` pattern — iterate existing raw content, skip if already populated, parse and insert.

**Rationale:** Ensures existing data is retroactively indexed on the next deployment. The idempotent "count > 0 → skip" guard prevents duplicate work.

## Risks / Trade-offs

- **[Storage increase]** → New tables add rows proportional to catalog size. Typical ControlCatalogs have 10-50 controls with 2-5 requirements each. Storage impact is negligible compared to the `evidence` table. Mitigated by using `ReplacingMergeTree` for deduplication.
- **[Import latency]** → Parsing catalogs adds milliseconds to import handlers. The parsing itself is in-memory YAML unmarshalling, not network-bound. Acceptable trade-off for query-time elimination of YAML parsing.
- **[Schema drift]** → If `go-gemara` types change, parsers must be updated. Mitigated by pinning `go-gemara` version and updating parsers when bumping the dependency.
- **[Incomplete catalog import]** → If a catalog YAML fails parsing, structured rows are skipped (warn-and-continue pattern). The raw YAML is always stored, so no data is lost. The assistant falls back to reading raw content.

## Future Exploration: Graph Query Layer

If traversal patterns outgrow fixed-hop SQL JOINs (recursive queries, variable-depth path-finding, cycle detection), evaluate OSI-licensed graph query engines that can overlay ClickHouse. No viable candidate exists today — Apache AGE targets PostgreSQL only, and ClickHouse-native options (e.g., PuppyGraph) are closed-source. Revisit when an OSI-licensed alternative matures.
