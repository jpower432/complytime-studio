## Why

Studio stores mapping documents as YAML blobs. When a control fails, there is no SQL join path to answer "which certifications or ATOs are affected?" The assistant parses YAML ad-hoc, which is fragile and non-deterministic. Studio claims to be Gemara-native but treats Gemara artifacts as opaque strings.

## What Changes

- Parse mapping document YAML at import time into a structured `mapping_entries` ClickHouse table.
- Enable SQL joins between `evidence` and `mapping_entries` for impact/blast radius queries.
- Add an impact query pattern to the assistant's evidence-schema skill.
- Preserve raw `mapping_documents.content` blob for display; structured rows are the query path.

## Capabilities

### New Capabilities

- `mapping-entry-storage`: Parse and store individual mapping entries (control → framework reference) in a structured ClickHouse table at import time.
- `impact-query`: SQL join pattern across `evidence × mapping_entries` to resolve which framework objectives are affected by control failures.

### Modified Capabilities

- `evidence-ingestion`: The mapping import path gains a structured write step alongside the existing blob storage.

## Impact

- **Schema**: New `mapping_entries` table in ClickHouse. Migration needed.
- **Gateway**: `POST /api/mappings/import` handler gains YAML parsing + batch insert to `mapping_entries`.
- **Store**: New `MappingEntry` type and `InsertMappingEntries` / query methods in `internal/store/`.
- **Skill**: `skills/evidence-schema/SKILL.md` updated with impact query patterns.
- **No changes to**: evidence ingestion pipeline, complyctl, OTel collector, or frontend.
- **Upstream dependency (deferred)**: `effective_policy` MCP tool blocked on [go-gemara#64](https://github.com/gemaraproj/go-gemara/issues/64). Not in scope for this change.
