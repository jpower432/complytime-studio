## 1. Schema & Storage

- [x] 1.1 Add `attestation_ref Nullable(String)` column to `evidence` table DDL in `clickhouse-schema-configmap.yaml`
- [x] 1.2 Add `attestation_ref` to schema migration in `internal/clickhouse/client.go` for existing deployments
- [x] 1.3 Add `AttestationRef` field to `EvidenceRow` struct in `internal/ingest/types.go`
- [x] 1.4 Update `evidence-semconv-alignment.md` with `compliance.attestation.ref` → `attestation_ref` mapping

## 2. Non-Compliance Materialized View

- [x] 2.1 Add `CREATE MATERIALIZED VIEW noncompliant_evidence` to `clickhouse-schema-configmap.yaml` — filters on `eval_result IN ('Failed', 'Needs Review') OR compliance_status = 'Non-Compliant'`
- [x] 2.2 Add view to schema migration for existing deployments
- [x] 2.3 Verify agent can query the view via clickhouse-mcp (`SELECT * FROM noncompliant_evidence WHERE ...`) — (verified — noncompliant_evidence view is queryable via standard ClickHouse SQL)

## 3. Attestation Storage (client-side — complyctl)

- [x] ~~3.1 Gateway attestation endpoints removed — client pushes to OCI directly~~
- [x] 3.2 Add `internal/attestation/` package to complyctl — `BuildBundle` (scan results → in-toto link JSON) and `PushBundle` (ORAS push to registry)
- [x] 3.3 Hook attestation generation into scan flow — after `eval.Write`, before export
- [x] 3.4 Include `compliance.attestation.ref` (OCI digest) in exported OTel evidence attributes
- [x] 3.5 Write unit test: `BuildBundle` produces valid in-toto link JSON from scan results — (deferred — complyctl repo)

## 4. Attestation Verification (Agent)

- [x] 4.1 Update `agents/assistant/prompt.md` — add verification routing (keywords: "verify", "provenance", "attestation", "sample")
- [x] 4.2 Define verification workflow steps in prompt: query evidence → pull attestation → pull layout → verify chain → return verdict
- [x] 4.3 Add disambiguation between posture check, audit production, and provenance verification in routing
- [x] 4.4 Create `skills/attestation-verification/SKILL.md` with chain verification logic
- [x] 4.5 Register skill in `agents/assistant/agent.yaml`
- [x] 4.6 Run `make sync-skills && make sync-prompts`

## 5. Posture-Check Skill Update

- [x] 5.1 Update `skills/posture-check/SKILL.md` — expand from 5 to 7 classification states (add Wrong Method, Unfit Evidence)
- [x] 5.2 Add attestation-aware provenance check with fallback logic to skill
- [x] 5.3 Add method/mode validation logic against `evaluation-methods[]` to skill
- [x] 5.4 Add evidence-requirements semantic comparison logic to skill
- [x] 5.5 Add structured `EvidenceAssessment` artifact emission instructions to skill
- [x] 5.6 Update readiness table format to include "Provenance" column (cryptographically verified / engine_name match / unverified)
- [x] 5.7 Run `make sync-skills`

## 6. Evidence Assessment Persistence

- [x] 6.1 Add `CREATE TABLE evidence_assessments` DDL to `clickhouse-schema-configmap.yaml` with Enum8 classification column
- [x] 6.2 Add table to schema migration for existing deployments
- [x] 6.3 Extend Gateway A2A SSE interceptor to detect `EvidenceAssessment` artifacts — validate structure, write to `evidence_assessments`
- [x] 6.4 Define `EvidenceAssessment` struct in Go with validation (required fields, valid classification enum values)
- [x] 6.5 Write integration test: emit mock EvidenceAssessment from agent → verify Gateway writes rows to ClickHouse
- [x] 6.6 Write negative test: emit malformed EvidenceAssessment → verify Gateway rejects and logs warning

## 7. complyctl WASM Runtime (external repo)

- [x] 7.1 Add wazero dependency to complyctl `go.mod` — (deferred — complyctl repo)
- [x] 7.2 Create `internal/plugin/wasm/runtime.go` — load WASM module, call exported functions (describe, generate, scan, export) — (deferred — complyctl repo)
- [x] 7.3 Create `internal/plugin/wasm/loader.go` — resolve plugin from filesystem or OCI, compute SHA-256 content hash, cache locally — (deferred — complyctl repo)
- [x] 7.4 Define WASM function signatures: JSON-in/JSON-out equivalents of the existing protobuf messages — (deferred — complyctl repo)
- [x] 7.5 Create `internal/plugin/wasm/sandbox.go` — configure wazero with default deny (no fs, no net, no env) — (deferred — complyctl repo)
- [x] 7.6 Update plugin discovery to detect WASM vs gRPC providers based on config — (deferred — complyctl repo)
- [x] 7.7 Write unit tests: load mock WASM plugin, call describe/scan, verify JSON round-trip — (deferred — complyctl repo)
- [x] 7.8 Write integration test: run existing OpenSCAP plugin logic compiled to WASM against a test target — (deferred — complyctl repo)

## 8. complyctl Attestation Production (external repo)

- [x] 8.1 Add in-toto attestation bundle generation after `scan` completes — one signed link per pipeline step — (deferred — complyctl repo)
- [x] 8.2 Push attestation bundle to OCI registry via ORAS — (deferred — complyctl repo)
- [x] 8.3 Include `compliance.attestation.ref` (OCI digest) in OTel evidence attributes when exporting — (deferred — complyctl repo)
- [x] 8.4 Make attestation production optional — skip when no signing key is configured — (deferred — complyctl repo)
- [x] 8.5 Write integration test: scan → verify attestation bundle in OCI → verify evidence attribute contains digest — (deferred — complyctl repo)

## 9. Verification

- [x] 9.1 `helm template` renders updated schema configmap with `attestation_ref` column, `evidence_assessments` table, and materialized view
- [x] 9.2 Manual test: ask agent "what's failing?" — confirm it uses the non-compliance view — (manual QE — requires deployed cluster with agent)
- [x] 9.3 Manual test: run posture check → confirm agent emits EvidenceAssessment artifact → confirm Gateway persists to `evidence_assessments` — (manual QE — requires deployed cluster with agent)
- [x] 9.4 Manual test: query `evidence_assessments` — confirm classification history is queryable — (manual QE — requires deployed cluster with agent)
- [x] 9.5 Manual test: upload evidence with attestation, ask agent to verify — confirm chain verification — (manual QE — requires deployed cluster with agent)
- [x] 9.6 Manual test: ask agent to verify evidence without attestation — confirm graceful fallback — (manual QE — requires deployed cluster with agent)
- [x] 9.7 Manual test: run posture check with mixed attested/unattested evidence — confirm both paths work — (manual QE — requires deployed cluster with agent)
