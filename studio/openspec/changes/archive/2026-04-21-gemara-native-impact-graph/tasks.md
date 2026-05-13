## 1. Schema

- [x] 1.1 Add `CREATE TABLE mapping_entries` statement to `internal/clickhouse/client.go` schema init
- [x] 1.2 Add `mapping_entries` to the ClickHouse schema ConfigMap in `charts/complytime-studio/templates/clickhouse-schema-configmap.yaml`

## 2. Store Layer

- [x] 2.1 Add `MappingEntry` struct to `internal/store/store.go`
- [x] 2.2 Add `InsertMappingEntries(ctx, []MappingEntry)` method to `Store`
- [x] 2.3 Add `CountMappingEntries(ctx, mappingID)` method for retroactive population check

## 3. YAML Parsing

- [x] 3.1 Create `internal/store/mapping_parser.go` with `ParseMappingYAML(content string) ([]MappingEntry, error)` function
- [x] 3.2 Write tests for parsing: valid mapping, missing optional fields, invalid YAML, empty mappings array

## 4. Import Handler

- [x] 4.1 Update `importMappingHandler` in `internal/store/handlers.go` to parse YAML and call `InsertMappingEntries` after blob insert
- [x] 4.2 Log warning on parse failure; do not fail the HTTP response

## 5. Retroactive Population

- [x] 5.1 Add a `PopulateMappingEntries` function that reads all `mapping_documents`, checks for existing entries, and backfills
- [x] 5.2 Call `PopulateMappingEntries` from gateway startup after schema init

## 6. Skill Update

- [x] 6.1 Add `mapping_entries` table schema to `skills/evidence-schema/SKILL.md`
- [x] 6.2 Add impact query pattern (evidence × mapping_entries JOIN) to the skill
- [x] 6.3 Add aggregation query pattern (GROUP BY framework, reference) to the skill

## 7. Verification

- [~] 7.1 Run `demo/seed.sh` and verify `mapping_entries` is populated (7 rows for the SOC 2 mapping) — *Skipped: requires running cluster, not automatable in CI*
- [~] 7.2 Run the impact query manually via ClickHouse to confirm join produces expected results — *Skipped: requires running cluster, not automatable in CI*
- [~] 7.3 Ask the assistant "which SOC 2 objectives are affected by failures in ampel-branch-protection?" and verify structured response — *Skipped: requires running cluster, not automatable in CI*
