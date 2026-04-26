## Why

Evidence in Studio has no provenance chain. The `evidence` table records *what* a scanner reported (`engine_name`, `eval_result`) but not *how* the assessment was performed or *whether* the pipeline was authorized. When an auditor samples evidence, they cannot verify that the right policy was fetched, the right scanner ran, or the right actor triggered it. The posture-check skill flags "Wrong Source" based on `engine_name` alone — a string comparison, not a cryptographic guarantee.

Beyond pass/fail, evidence can be wrong in ways that require deeper analysis: untrusted actor, wrong assessment method, evidence that doesn't match the plan's requirements. These classifications require the agent to reason over Policy assessment plans and evidence together — not just SQL filters.

Separately, complyctl's plugin system uses standalone gRPC executables (`complyctl-provider-*`). Adding new scanner formats requires building and distributing a new binary. WASM plugins would make this sandboxed, polyglot, and distributable via OCI.

## What Changes

- **Client-side attestation storage**: complyctl pushes in-toto attestation bundles to OCI after scan, includes the digest as `compliance.attestation.ref` in evidence OTel attributes. Studio receives the digest passively — no attestation upload endpoint needed.
- **Attestation verification**: The assistant agent gains a verification workflow — pull the attestation bundle from OCI via oras-mcp, pull the layout from the Policy reference, verify the chain, return a verdict.
- **Layout reference in Policy**: The Gemara `AssessmentPlan` gains an optional `layout` field referencing an in-toto layout stored in OCI. The layout defines expected steps, authorized actors, and material/product chaining.
- **Evidence assessment persistence**: The agent emits structured `EvidenceAssessment` artifacts classifying each evidence row against its assessment plan (7 states: Healthy, Failing, Wrong Source, Wrong Method, Unfit Evidence, Stale, Blind). The Gateway intercepts these from the A2A stream and persists them to an `evidence_assessments` table — same pattern as AuditLog auto-persist. Agent never writes to ClickHouse directly.
- **Non-compliance materialized view**: A ClickHouse materialized view surfaces evidence with failing results as a fast-path for the agent.
- **complyctl WASM plugins**: complyctl migrates from gRPC provider binaries to WASM modules loaded via wazero. The host resolves credentials and injects them as variables. Plugins are distributed via OCI.

## Capabilities

### New Capabilities
- `attestation-storage`: OCI-based storage and retrieval of in-toto attestation bundles linked to evidence rows via `attestation_ref`
- `attestation-verification`: Agent workflow to pull attestation bundle + layout from OCI, verify the chain, and return a provenance verdict
- `layout-reference`: Optional `layout` field on Policy assessment plans referencing an in-toto layout artifact in OCI
- `evidence-assessment-persist`: Gateway intercepts structured EvidenceAssessment artifacts from the A2A stream and writes classifications to `evidence_assessments` table with provenance (model, prompt version, timestamp)
- `noncompliance-view`: ClickHouse materialized view surfacing failing/non-compliant evidence for agent queries
- `complyctl-wasm-plugins`: WASM plugin runtime in complyctl replacing gRPC provider binaries, with credential injection and OCI distribution

### Modified Capabilities
- `semconv-alignment`: Add `attestation_ref` column to the evidence table and semconv mapping
- `posture-check-skill`: Expand from 5 to 7 classification states (add Wrong Method, Unfit Evidence). Use attestation verification when `attestation_ref` is present. Emit structured EvidenceAssessment artifact for Gateway persistence.

## Impact

- **ClickHouse schema**: New `attestation_ref Nullable(String)` column on `evidence` table. New `evidence_assessments` table for persisted classifications. New materialized view for non-compliant evidence.
- **Gateway**: Extend A2A SSE interceptor to detect and persist `EvidenceAssessment` artifacts (same pattern as AuditLog auto-persist)
- **OCI Registry**: Stores attestation bundles and WASM plugin modules as new artifact types
- **Policy schema (upstream)**: Proposed optional `layout` field on `#AssessmentPlan` — requires Gemara schema discussion
- **Assistant agent**: New verification workflow, expanded posture-check skill (7 states), structured EvidenceAssessment output
- **complyctl (external repo)**: WASM runtime (wazero), plugin loader (filesystem + OCI), credential injection. Produces attestation bundles and pushes to OCI.
