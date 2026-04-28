## Context

The ClickHouse schema materializes Gemara Layer 2 (Controls, Threats) but not Layer 3 (Risks). The agent can traverse `evidence → controls → control_threats → threats` today. Adding `risks` and `risk_threats` closes the loop so the agent can derive **risk severity** from evidence through joins alone, without denormalized columns.

The Gemara `RiskCatalog` type defines:
- `Risk`: id, title, severity (Low/Medium/High/Critical), group, impact, threats[]
- `Risk.Threats[]`: `MultiEntryMapping` linking a risk to threat entries
- `RiskCategory`: group-level `max-severity` tolerance boundary

The `Policy.Risks` block references risks via `EntryMapping(reference-id, entry-id)` for both mitigated and accepted risks.

## Goals / Non-Goals

**Goals:**
- Materialize `RiskCatalog` into `risks` and `risk_threats` ClickHouse tables
- Enable the agent to derive risk severity from evidence via: `evidence → control_threats → risk_threats → risks`
- Follow the same parse/backfill/import pattern as controls and threats
- Document the risk severity traversal query in the evidence-schema skill

**Non-Goals:**
- Policy-level risk disposition tracking (mitigated vs accepted) — that's policy-composer workflow, not schema
- Risk scoring or aggregation views — agent computes these at query time
- RiskCatalog authoring — handled by policy-composer agent, not this change
- UI visualization of risk data — separate change

## Decisions

### 1. Two new tables: `risks` and `risk_threats`

| Table | Columns | Order Key |
|:--|:--|:--|
| `risks` | catalog_id, risk_id, title, description, severity, group_id, impact, policy_id, imported_at | (catalog_id, risk_id) |
| `risk_threats` | catalog_id, risk_id, threat_reference_id, threat_entry_id, imported_at | (catalog_id, risk_id, threat_reference_id, threat_entry_id) |

`severity` stored as `LowCardinality(String)` — matches how `state` is stored on controls. The 4 values (Low, Medium, High, Critical) are low-cardinality.

**Alternative**: Store severity as `Enum8`. Rejected — enums are rigid across schema migrations and the string approach is consistent with existing tables.

### 2. `risk_threats` mirrors `control_threats` structure

Both are junction tables with the same `(catalog_id, entity_id, threat_reference_id, threat_entry_id)` shape. The shared `threat_entry_id` column is the join key that connects risks to controls through their common threats.

### 3. `ParseRiskCatalog` in `internal/gemara/`

New function following the established pattern of `ParseControlCatalog` and `ParseThreatCatalog`. Returns `[]RiskRow` and `[]RiskThreatRow`.

### 4. Import via existing `importCatalogHandler`

`detectCatalogType` already switches on `metadata.type`. Adding `"RiskCatalog"` to the switch and calling `parseCatalogStructuredRows` for the new type keeps the import path unified.

### 5. Startup backfill via `PopulateRisks`

Same pattern as `PopulateControls` — iterate stored catalogs, skip already-populated ones, parse and insert. Called from `main.go` alongside the other populate functions.

## Risks / Trade-offs

- **[Risk]** RiskCatalog may not exist for all policies → **Mitigation**: Backfill silently returns when no RiskCatalog rows exist. Agent handles missing data gracefully.
- **[Risk]** `threat_entry_id` join between `control_threats` and `risk_threats` assumes consistent threat IDs across catalogs → **Mitigation**: This is enforced by Gemara's `mapping-references` contract. Both tables reference the same threat catalog via `threat_reference_id`.
- **[Trade-off]** `impact` stored as String, not structured → Acceptable for agent consumption. The field is prose by definition in the Gemara schema.
