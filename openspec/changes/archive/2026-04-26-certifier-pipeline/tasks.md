## 1. Schema Changes

- [x] 1.1 Add `certifications` table to `clickhouse-schema-configmap.yaml`
- [x] 1.2 Add `certified Bool DEFAULT false` column to `evidence` table in `clickhouse-schema-configmap.yaml`
- [x] 1.3 Update `EvidenceRow` struct in `internal/ingest/types.go` with `Certified` field

## 2. Certifier Interface & Pipeline

- [x] 2.1 Create `internal/certifier/certifier.go` with `Certifier` interface, `Verdict` type, and `CertResult` struct
- [x] 2.2 Create `internal/certifier/pipeline.go` with `Pipeline` struct and sequential `Run` method
- [x] 2.3 Write unit tests for pipeline runner (all-pass, mixed verdicts, error handling)

## 3. Day-One Certifiers

- [x] 3.1 Implement schema certifier (`internal/certifier/schema.go`) — metadata presence, enum validation, timestamp checks
- [x] 3.2 Implement provenance certifier (`internal/certifier/provenance.go`) — source_registry/attestation_ref presence, known registry check
- [x] 3.3 Implement executor certifier (`internal/certifier/executor.go`) — engine_name presence and registration check
- [x] 3.4 Implement attestation certifier (`internal/certifier/attestation.go`) — bundle resolution from Studio registry, structure/signature/chain verification
- [x] 3.5 Write unit tests for each certifier (pass, fail, skip, error scenarios per spec)

## 4. Certification Handler

- [x] 4.1 Create `internal/events/certification_handler.go` with NATS subscriber, ClickHouse query, pipeline invocation
- [x] 4.2 Implement batch INSERT to `certifications` table after pipeline run
- [x] 4.3 Implement `evidence.certified` UPDATE logic (at least one pass, zero fails)
- [x] 4.4 Register `CertificationHandler` alongside `PostureCheckHandler` in gateway startup
- [x] 4.5 Apply debounce window (existing 30s pattern) to certification handler
- [x] 4.6 Write integration test: evidence ingest → NATS event → certifier pipeline → certifications written

## 5. Remove Manual Upload

- [x] 5.1 Remove `POST /api/evidence/upload` handler from `internal/store/handlers.go`, replace with 410 Gone response
- [x] 5.2 Remove CSV import handler and any associated parsing logic
- [x] 5.3 Remove upload button, file input, and modal from `evidence-view.tsx` entirely
- [x] 5.4 Remove upload-related CSS from `global.css`
- [x] 5.5 Update any tests that reference the upload endpoint or CSV import

## 6. Certification UI

- [x] 6.1 Add certification status column to evidence table with ✓/⚠ icons
- [x] 6.2 Implement per-certifier detail expand panel (query `certifications` table via API)
- [x] 6.3 Add `GET /api/certifications?evidence_id=` endpoint to gateway
- [x] 6.4 Add certification summary bar component (certified/uncertified segments, clickable)
- [x] 6.5 Add `Certification` to the "+ Filter" menu with Certified/Uncertified options
- [x] 6.6 Wire certification bar clicks to filter chip system

## 7. API & Wiring

- [x] 7.1 Add `certified` field to evidence query response in `internal/store/store.go`
- [x] 7.2 Ensure evidence list API returns `certified` column for frontend consumption
- [x] 7.3 Add known-registries and known-engines configuration (environment or config file)
- [x] 7.4 Wire certifier registration in gateway startup (schema, provenance, executor, attestation)
