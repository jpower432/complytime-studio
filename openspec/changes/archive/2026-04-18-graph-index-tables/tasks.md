## 1. Gemara Parsers

- [x] 1.1 Define `ControlRow`, `AssessmentRequirementRow`, `ControlThreatRow` structs in `internal/gemara/controls.go`
- [x] 1.2 Implement `ParseControlCatalog(content, catalogID, policyID)` returning `([]ControlRow, []AssessmentRequirementRow, []ControlThreatRow, error)`
- [x] 1.3 Define `ThreatRow` struct in `internal/gemara/threats.go`
- [x] 1.4 Implement `ParseThreatCatalog(content, catalogID, policyID)` returning `([]ThreatRow, error)`
- [x] 1.5 Write unit tests for `ParseControlCatalog` in `internal/gemara/controls_test.go` — valid catalog, empty controls, invalid YAML
- [x] 1.6 Write unit tests for `ParseThreatCatalog` in `internal/gemara/threats_test.go` — valid catalog, empty threats, invalid YAML

## 2. ClickHouse DDL

- [x] 2.1 Add `controls` CREATE TABLE statement to `EnsureSchema` in `internal/clickhouse/client.go`
- [x] 2.2 Add `assessment_requirements` CREATE TABLE statement to `EnsureSchema`
- [x] 2.3 Add `control_threats` CREATE TABLE statement to `EnsureSchema`
- [x] 2.4 Add `threats` CREATE TABLE statement to `EnsureSchema`

## 3. Store Interfaces and Implementations

- [x] 3.1 Define `ControlStore` interface in `internal/store/store.go` with `InsertControls`, `InsertAssessmentRequirements`, `InsertControlThreats`, `CountControls`
- [x] 3.2 Define `ThreatStore` interface in `internal/store/store.go` with `InsertThreats`, `CountThreats`
- [x] 3.3 Implement `ControlStore` methods in ClickHouse store (batch insert for controls, assessment_requirements, control_threats)
- [x] 3.4 Implement `ThreatStore` methods in ClickHouse store (batch insert for threats)
- [x] 3.5 Add `ControlStore` and `ThreatStore` to `Stores` struct in `internal/store/handlers.go`

## 4. Import Handlers

- [x] 4.1 Add `POST /api/catalogs/import` handler that accepts catalog YAML, detects type (ControlCatalog vs ThreatCatalog), stores raw content, and parses structured rows
- [x] 4.2 Extend `importPolicyHandler` to parse embedded catalog references from policy imports (if catalog content is included)
- [x] 4.3 Wire new handler in `Register` function

## 5. Backfill on Startup

- [x] 5.1 Implement `PopulateControls` in `internal/store/populate.go` — iterate stored catalogs, skip if `CountControls > 0`, parse and insert
- [x] 5.2 Implement `PopulateThreats` in `internal/store/populate.go` — iterate stored catalogs, skip if `CountThreats > 0`, parse and insert
- [x] 5.3 Call `PopulateControls` and `PopulateThreats` from gateway startup sequence alongside existing `PopulatePolicyContacts` and `PopulateMappingEntries`

## 6. Skill and Schema Documentation

- [x] 6.1 Add `controls`, `assessment_requirements`, `control_threats`, `threats` table schemas to `skills/evidence-schema/SKILL.md`
- [x] 6.2 Add threat impact traversal query pattern to skill file
- [x] 6.3 Add coverage completeness query pattern to skill file
- [x] 6.4 Add requirement text enrichment query pattern to skill file
- [x] 6.5 Add framework-to-threat traversal query pattern to skill file

## 7. Helm Chart

- [x] 7.1 Add new table DDL to `charts/complytime-studio/templates/clickhouse-schema-configmap.yaml` (if schema ConfigMap is used for init)
