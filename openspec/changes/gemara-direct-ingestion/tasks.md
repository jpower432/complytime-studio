# Tasks: Direct Gemara-Based Evidence Ingestion

- [ ] Extract `detectType` from `cmd/ingest/main.go` → `internal/ingest/detect.go`
- [ ] Extract `derivePolicyID` from `cmd/ingest/main.go` → `internal/ingest/policy.go`
- [ ] Extract `bytesFetcher` from `cmd/ingest/main.go` → `internal/ingest/fetcher.go`
- [ ] Add `importGemaraEvidenceHandler` to `internal/store/handlers.go`
- [ ] Handle `application/x-yaml`, `text/yaml`, and `multipart/form-data` content types
- [ ] Add `IngestWriter *ingest.Writer` to `Stores` struct
- [ ] Initialize `ingest.Writer` in `cmd/gateway/main.go` (reuse ClickHouse config)
- [ ] Register `POST /api/evidence/import` route
- [ ] Wire NATS event publish in handler (reuse existing `EvidencePublisher`)
- [ ] Refactor `cmd/ingest/main.go` to call `internal/ingest` functions (thin wrapper)
- [ ] Add deprecation notice to `cmd/ingest` README/help output
- [ ] Write ADR superseding `otel-native-ingestion.md`: primary path is now API push
- [ ] Tests: valid EvalLog, valid EnfLog, invalid YAML, wrong type, empty evaluations, multipart, content-type rejection, NATS publish verification
