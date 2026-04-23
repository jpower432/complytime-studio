## 1. Schema & DDL

- [ ] 1.1 Add `risks` CREATE TABLE statement to `EnsureSchema` in `internal/clickhouse/client.go`
- [ ] 1.2 Add `risk_threats` CREATE TABLE statement to `EnsureSchema` in `internal/clickhouse/client.go`

## 2. Types & Store Interface

- [ ] 2.1 Define `RiskRow` and `RiskThreatRow` structs in `internal/gemara/risks.go`
- [ ] 2.2 Define `RiskStore` interface in `internal/store/store.go` with `InsertRisks`, `InsertRiskThreats`, `CountRisks`
- [ ] 2.3 Implement `InsertRisks`, `InsertRiskThreats`, `CountRisks` on `Store` in `internal/store/store.go`
- [ ] 2.4 Add compile-time check `var _ RiskStore = (*Store)(nil)`

## 3. Parser

- [ ] 3.1 Implement `ParseRiskCatalog` in `internal/gemara/risks.go`
- [ ] 3.2 Write unit tests for `ParseRiskCatalog` in `internal/gemara/risks_test.go` covering: multi-risk catalog, risk with no threats, severity string extraction

## 4. Import Handler

- [ ] 4.1 Add `"RiskCatalog"` case to `detectCatalogType` in `internal/store/handlers.go`
- [ ] 4.2 Add `"RiskCatalog"` case to `parseCatalogStructuredRows` calling `ParseRiskCatalog` and inserting via `RiskStore`
- [ ] 4.3 Add `RiskStore` to `Stores` struct and wire in `Register` and `importCatalogHandler`

## 5. Startup Backfill

- [ ] 5.1 Implement `PopulateRisks` in `internal/store/populate.go` (iterate stored RiskCatalogs, skip populated, parse, insert)
- [ ] 5.2 Call `PopulateRisks` from `cmd/gateway/main.go` alongside existing populate functions

## 6. Agent Skill

- [ ] 6.1 Add `risks` and `risk_threats` table DDL documentation to `skills/evidence-schema/SKILL.md`
- [ ] 6.2 Add risk severity traversal query pattern to `skills/evidence-schema/SKILL.md`
- [ ] 6.3 Add risk exposure summary query pattern to `skills/evidence-schema/SKILL.md`
- [ ] 6.4 Add unmitigated risk identification query pattern to `skills/evidence-schema/SKILL.md`

## 7. Verification

- [ ] 7.1 Run `go vet -tags dev ./...` and confirm no errors
- [ ] 7.2 Run `go test -tags dev -race ./internal/gemara/... ./internal/store/...` and confirm pass
- [ ] 7.3 Deploy to cluster, import a sample RiskCatalog, verify `risks` and `risk_threats` tables populated
