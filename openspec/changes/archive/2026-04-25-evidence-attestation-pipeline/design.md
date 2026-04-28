## Context

Studio ingests evidence via OTel Collector or REST API, stores it in ClickHouse, and uses the assistant agent to produce AuditLogs. Evidence arrives already semconv-aligned — complyctl's plugins handle scanner-specific logic and produce structured EvaluationLogs. Studio is the consumption and analysis side.

The posture-check skill validates evidence cadence and source identity using `engine_name` — a plain string with no cryptographic backing. There is no way to verify that evidence was produced by an authorized pipeline.

complyctl currently uses hashicorp/go-plugin with gRPC provider binaries. Adding a new scanner requires building and distributing a standalone executable. WASM plugins (loaded via wazero) would make this sandboxed, polyglot, and distributable via OCI — following established patterns for host/plugin separation, credential injection, content-addressed identity, and envelope wrapping.

## Goals / Non-Goals

**Goals:**
- Store in-toto attestation bundles in OCI, linked to evidence rows by digest
- Enable the agent to verify evidence provenance on demand (auditor sampling)
- Surface non-compliant evidence via ClickHouse materialized view
- Define the WASM plugin contract for complyctl (replacing gRPC providers)
- Keep ClickHouse as the sole query engine — no new databases

**Non-Goals:**
- Evidence transformation in Studio — complyctl normalizes scanner output, Studio receives semconv-aligned rows
- Graph database — ClickHouse JOINs suffice for current entity count
- Dashboards — the agent replaces dashboards with natural language queries
- Background monitoring — the agent answers when asked; if "I forget to check" becomes real, it's a cron job
- Distributed orchestration (NATS, message queues)

## Decisions

### Decision 1: Client pushes attestation bundles to OCI, Studio only reads

complyctl pushes in-toto attestation bundles to the OCI registry after scan, then includes the OCI digest as `compliance.attestation.ref` in the evidence OTel attributes. The evidence row arrives in ClickHouse with `attestation_ref` already populated. Studio never touches bundles during ingestion — the agent reads them from OCI via oras-mcp when verification is requested.

**Why client-side push, not Gateway intermediary:** complyctl already has OCI push capability (oras-go). It knows the attestation digest at scan time. Having the client include it as an OTel attribute means one write, not two. The Gateway doesn't need to correlate separate attestation and evidence uploads.

**Why no Studio attestation endpoints:** Attestation bundles are produced by the scan client, not by Studio. Adding upload/download endpoints creates a coordination problem (race between evidence row and attestation upload) and makes the Gateway responsible for stitching — exactly the kind of transform logic we moved out of Studio.

**Why OCI over ClickHouse blob storage:** Attestation bundles are signed artifacts, not queryable data. They belong with policies — immutable, versioned, content-addressed.

### Decision 2: Layout lives alongside policy in OCI, referenced by digest

The in-toto layout is stored as an additional layer in the Policy's OCI manifest or as a separately tagged artifact. The Policy's `assessment-plans[].layout` field references it by digest.

**Why not embed in Policy YAML:** Layouts contain signing key identities and step definitions that change independently of policy content. Decoupling avoids forcing a policy version bump for layout-only changes.

**Why digest reference:** Immutable binding. No TOCTOU issues.

### Decision 3: complyctl WASM plugins via wazero

complyctl migrates from hashicorp/go-plugin (gRPC provider binaries) to WASM modules loaded via wazero. The plugin interface mirrors the existing gRPC contract: `Describe`, `Generate`, `Scan`, `Export`.

```
Current:  complyctl → gRPC → complyctl-provider-openscap (binary)
Proposed: complyctl → wazero → openscap-plugin.wasm (sandboxed)
```

**Why wazero:** Pure Go, no CGO, no external dependencies. Runs in the same process as complyctl. Plugins are sandboxed by default — no filesystem, no network unless explicitly granted.

**Why mirror the existing gRPC contract:** The Describe/Generate/Scan/Export interface already works. Plugins already implement it. Minimizing interface changes means existing plugin logic ports directly — only the transport changes (gRPC → WASM function calls).

**Credential injection:** complyctl already resolves credentials and passes them as `target_variables` in `GenerateRequest` and `ScanRequest`. Same pattern continues — host resolves, plugin receives. Plugins never see credential resolution logic.

**Distribution:** WASM modules distributed via OCI. complyctl pulls plugins the same way it pulls policies. Content-addressed identity via sha256 of the WASM blob.

### Decision 4: Verification is agent-driven, on-demand

The assistant agent verifies attestation chains when an auditor asks:

1. Query ClickHouse for the evidence row → get `attestation_ref`
2. Pull attestation bundle from OCI via oras-mcp
3. Pull layout from Policy's OCI reference via oras-mcp
4. Compare attestation steps against layout expectations
5. Return verdict (verified / broken chain / missing attestation)

**Why not a Gateway endpoint:** Verification is an audit activity, not ingestion middleware. The agent already queries ClickHouse and calls oras-mcp.

**Why not a dedicated verification service:** One question, one place. No microservice decomposition for a single-user operation.

### Decision 5: Materialized view for non-compliance flagging

A ClickHouse materialized view continuously surfaces evidence with `eval_result = 'Failed'` or `compliance_status = 'Non-Compliant'`. The agent queries this view for posture questions.

```sql
CREATE MATERIALIZED VIEW noncompliant_evidence
ENGINE = ReplacingMergeTree(ingested_at)
ORDER BY (policy_id, control_id, target_id, collected_at)
AS SELECT *
FROM evidence
WHERE eval_result IN ('Failed', 'Needs Review')
   OR compliance_status = 'Non-Compliant'
```

**Why a materialized view, not a query filter:** ClickHouse MV is incrementally maintained — new rows are filtered on insert. No periodic recomputation. The agent gets fast reads over a pre-filtered dataset.

### Decision 6: Posture-check falls back gracefully

When `attestation_ref` is present, provenance check uses cryptographic verification. When absent, it falls back to `engine_name` string comparison (current behavior). Zero-disruption adoption.

### Decision 7: Agent proposes, Gateway writes

The agent performs posture checks and emits structured `EvidenceAssessment` artifacts containing per-evidence classifications. The Gateway intercepts these from the A2A SSE stream and persists them to an `evidence_assessments` table in ClickHouse. This mirrors the existing AuditLog auto-persist pattern.

```sql
CREATE TABLE IF NOT EXISTS evidence_assessments
(
    evidence_id   String,
    policy_id     String,
    plan_id       String,
    classification Enum8(
        'Healthy' = 1,
        'Failing' = 2,
        'Wrong Source' = 3,
        'Wrong Method' = 4,
        'Unfit Evidence' = 5,
        'Stale' = 6,
        'Blind' = 7
    ),
    reason        String,
    assessed_at   DateTime64(3),
    assessed_by   String
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(assessed_at)
ORDER BY (policy_id, plan_id, evidence_id, assessed_at)
```

**Why agent proposes, Gateway writes:**
- Agent stays read-only — no ClickHouse write credentials, no INSERT access
- Classifications persist as queryable, historical data
- Gateway validates structure before writing — rejects malformed or hallucinated data
- Leverages existing AuditLog auto-persist interceptor — no new architectural pattern
- `assessed_by` (model + prompt version) and `assessed_at` provide provenance on the agent's opinion

**Why not agent-writes-directly:** The agent is an LLM. Granting write access to a shared data store creates a blast radius. The Gateway is trusted infrastructure with validation logic.

**Why not fire-and-forget:** Classifications are valuable for trend analysis, regression detection, and audit sampling. Ephemeral responses lose this signal.

### Decision 8: Posture-check expanded to 7 classification states

The posture-check skill moves from 5 states (Healthy, Failing, Wrong Source, Stale, Blind) to 7 states, adding:
- **Wrong Method** — evidence exists but was collected using a method/mode that doesn't match the assessment plan's `evaluation-methods`
- **Unfit Evidence** — evidence exists, correct source and method, but content doesn't satisfy the plan's `evidence-requirements` (semantic mismatch)

Priority (worst wins): Blind > Wrong Source > Wrong Method > Unfit Evidence > Stale > Failing > Healthy.

**Why these states:** "Wrong Source" covers *who* ran it. "Wrong Method" covers *how* it was run. "Unfit Evidence" covers *what* was collected. Together they give a complete picture of evidence quality beyond pass/fail.

### Decision 9: OCI registry is the source of truth for policies

Policies are authored as Gemara OCI bundles and pushed to Studio's registry. Studio watches or pulls from its own registry to populate ClickHouse (content, metadata, catalog decomposition). complyctl pulls from the same registry.

```
Author → push OCI bundle → Studio Registry
                              ↓                    ↓
                     Studio (ClickHouse)      complyctl (scan)
```

**Why registry-first, not REST-first:** The REST `POST /api/policies/import` creates a split — ClickHouse has the policy but the registry doesn't. complyctl can't find it. Two sources of truth for the same artifact.

**Why not "import writes to both":** Import becomes a publisher, which violates the registry's role as the immutable, content-addressed store. The registry should be the origin; ClickHouse is the derived, queryable projection.

**Migration:** The seed script and `POST /api/policies/import` remain for bootstrapping but are deprecated for production use. The canonical path is OCI push.

### Decision 10: Accept evidence for unknown policies, warn

When evidence arrives with a `policy_id` that doesn't exist in the `policies` table, Studio accepts and ingests it but logs a warning. Evidence is never rejected due to import ordering — the policy may be imported later.

**Why accept:** Evidence and policy imports may arrive in any order. Rejecting evidence because a policy hasn't been imported yet causes silent data loss. The scan client shouldn't need to coordinate timing with Studio.

**Why warn:** Orphaned evidence (policy never imported) is invisible to the agent — it joins `policies` + `evidence`. A warning in gateway logs surfaces the gap. Future: the agent can query for evidence where `policy_id NOT IN (SELECT policy_id FROM policies)` to surface orphans on demand.

### Decision 11: AuditLog results carry optional assessment-override

AuditLog `results[]` entries may include an `assessment-override` field carrying the 7-state classification (Healthy, Failing, Wrong Source, Wrong Method, Unfit Evidence, Stale, Blind). When present, the Gateway interceptor extracts it during AuditLog persistence and writes a corresponding row to `evidence_assessments` with `assessed_by = 'audit:<audit_id>'`.

```yaml
results:
  - id: result-01
    title: "Branch Protection"
    type: Finding
    assessment-override: Wrong Source
    description: "Evidence collected by qualys, policy requires nessus"
```

When `assessment-override` is absent, the interceptor falls back to a coarse mapping:

| AuditLog result type | Default classification |
|:-----|:-----------|
| Strength | Healthy |
| Finding | Failing |
| Gap | Blind |
| Observation | *(no override written)* |

`evidence_assessments` gains a `source_audit_id` column linking overrides to the originating AuditLog. The `FINAL` deduplication naturally prefers the latest row — human overrides written after agent assessments take precedence.

**Why optional field, not separate API:** The AuditLog review is already the auditor's workflow. Adding a field to the existing artifact avoids a second write path. The interceptor already parses AuditLog YAML — extracting one more field is incremental.

**Why not force the coarse mapping:** The 7-state model (Wrong Source, Wrong Method, Unfit Evidence) captures *why* something failed. The AuditLog's Finding type doesn't. Coarse mapping loses signal. The override field preserves it when the auditor (or agent) provides it.

### Decision 12: Auto-trigger posture-check on evidence ingestion

When new evidence is ingested (REST or OTel), the Gateway asynchronously triggers a posture-check for each affected `policy_id`. The trigger is fire-and-forget — evidence ingestion succeeds regardless of whether posture-check completes.

```
Evidence arrives (POST /api/evidence)
     │
     ├── INSERT into ClickHouse (synchronous, existing)
     │
     └── for each distinct policy_id:
           enqueue posture-check(policy_id)  (async)
               │
               ▼
         Agent runs posture-check
               │
               ▼
         EvidenceAssessment artifact emitted
               │
               ▼
         Gateway interceptor persists to evidence_assessments
```

**Why auto-trigger, not cron:** Posture should reflect the latest evidence. A daily cron means assessments are up to 24 hours stale. Auto-trigger means assessments are current within seconds of evidence arrival.

**Why fire-and-forget:** Evidence ingestion must not block on agent availability. If the agent is busy or down, evidence is stored; assessments catch up when the agent is available. The `unassessed` count in `policy_posture` surfaces the gap.

**Deduplication:** Multiple evidence rows for the same policy in a single batch trigger one posture-check, not N. The Gateway deduplicates by policy_id within a configurable window (default: 30s).

**Cold start:** On first deployment with existing evidence but no assessments, a one-time sweep triggers posture-check for all policies with evidence. Subsequent triggers are incremental.

### Decision 13: Unified compliance views

Two ClickHouse VIEWs (not materialized) provide pre-composed query surfaces:

**`unified_compliance_state`** — row-level JOIN of evidence + latest assessment:

```sql
CREATE VIEW unified_compliance_state AS
SELECT
    e.evidence_id, e.policy_id, e.control_id, e.target_id,
    e.target_name, e.eval_result, e.collected_at, e.attestation_ref,
    a.classification, a.reason, a.assessed_by, a.assessed_at,
    a.source_audit_id
FROM evidence e
LEFT JOIN (SELECT * FROM evidence_assessments FINAL) a
    ON e.evidence_id = a.evidence_id
```

**`policy_posture`** — aggregated per-policy, per-target summary:

```sql
CREATE VIEW policy_posture AS
SELECT
    e.policy_id, p.title AS policy_title, e.target_id,
    countIf(a.classification = 'Healthy') AS healthy,
    countIf(a.classification = 'Failing') AS failing,
    countIf(a.classification = 'Wrong Source') AS wrong_source,
    countIf(a.classification = 'Wrong Method') AS wrong_method,
    countIf(a.classification = 'Unfit Evidence') AS unfit,
    countIf(a.classification = 'Stale') AS stale,
    countIf(a.classification = 'Blind') AS blind,
    countIf(a.classification IS NULL) AS unassessed,
    max(e.collected_at) AS latest_evidence,
    max(a.assessed_at) AS latest_assessment
FROM evidence e
LEFT JOIN (SELECT * FROM evidence_assessments FINAL) a
    ON e.evidence_id = a.evidence_id
LEFT JOIN policies p ON e.policy_id = p.policy_id
GROUP BY e.policy_id, p.title, e.target_id
```

**Why VIEWs, not materialized:** ClickHouse MVs only trigger on INSERT to the source table, not on JOINed tables. A MV over `evidence JOIN evidence_assessments` would miss assessment updates. Query-time VIEWs are always current.

**Consumers:** Agent posture queries (`policy_posture` for fast summaries, `unified_compliance_state` for drill-down), future REST endpoints, future workbench dashboard.

### Exploratory: Policy version binding on assessments

Assessments are point-in-time against a specific policy version. "Compliant with Policy A v1.0 at time T" is not invalidated by Policy A v2.0 existing. Currently `evidence_assessments` has `policy_id` but no version — the version context is implicit (whatever was imported at `assessed_at` time).

**Candidate change:** Add `policy_version` to `evidence_assessments`. Keeps `policy_id` stable across versions while binding each assessment to the exact version it was evaluated against. Enables historical queries like "posture against v1.0 vs v2.0."

**Related principle:** Catalog changes warrant a new policy version, not mutation of an existing policy. The OCI registry enforces this naturally — new tag, old digest persists. Policy import should trigger posture re-evaluation for new evidence, not retroactive invalidation of historical assessments.

**Status:** Exploratory. Not blocking current work. Revisit when multi-version policy support is implemented.

## Risks / Trade-offs

| Risk | Mitigation |
|:-----|:-----------|
| No attestation producers exist yet — attestation_ref will be NULL for all current evidence | Graceful fallback (Decision 6). Storage contract is defined here; any tool that produces in-toto bundles can use it. |
| Layout schema is a proposed Gemara extension — upstream may reject | Layout reference is optional. If rejected, layouts are standalone OCI artifacts. Verification still works via convention. |
| complyctl WASM migration is a large change to an external repo | Interface mirrors existing gRPC contract exactly. Plugin logic ports directly — only transport changes. Can run gRPC and WASM plugins side-by-side during migration. |
| wazero may not support all WASI features plugins need | wazero supports wasi_snapshot_preview1 (filesystem, env, clock). complyctl plugins that need HTTP can use host-provided functions. |
| Agent hallucination during verification | Verification logic is deterministic tool output, not LLM reasoning. The tool returns structured pass/fail; the agent presents it. |
| Classifications can become stale | Auto-trigger on evidence ingest (Decision 12). `policy_posture.unassessed` surfaces rows the agent hasn't reached yet. |
| Gateway interceptor adds complexity | Mirrors existing AuditLog interceptor. Same code path, different artifact type. Incremental, not novel. |
| Auto-trigger floods agent with posture-checks | Deduplication window (30s) per policy_id. Agent processes one check per policy per batch, not per evidence row. |
| `policy_posture` VIEW performance at scale | Query-time JOINs on indexed columns. ClickHouse handles this at 10M+ rows. If latency grows, promote to periodic refresh MV. |
| AuditLog `assessment-override` field not in Gemara schema | Field is optional and ignored by `validate_gemara_artifact` (extra fields are allowed). Propose upstream when pattern stabilizes. |
