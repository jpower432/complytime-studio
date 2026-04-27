## Context

Studio ingests evidence via OTel Collector and Gemara artifact uploads (`cmd/ingest`). Evidence arrives in ClickHouse with no trust annotation. The `PostureCheckHandler` fires on NATS `EvidenceEvent` and computes pass-rate deltas for notifications — it does not evaluate evidence legitimacy.

Manual upload paths (`POST /api/evidence/upload`, CSV import) allow unstructured, unverifiable data to enter the system. The existing attestation-verification skill reaches into the client's OCI registry to verify bundles on demand — crossing the trust boundary.

The reference model: ingest blindly, certify asynchronously from within your own trust boundary, never trust client-side claims.

## Goals / Non-Goals

**Goals:**
- Extensible certifier interface — adding a certifier means implementing one function and registering it
- Day-one certifiers: schema, provenance, executor, attestation
- Append-only `certifications` table in ClickHouse for operational queries
- Denormalized `evidence.certified` bool for fast UI reads
- Certification status visible in the evidence UI (indicator, bar, detail, filter)
- Remove manual CSV/form upload paths entirely
- All verification from within Studio's trust boundary — no reaching into client registries

**Non-Goals:**
- Tamper-evident ledger (deferred to Trillian, see `docs/decisions/transparency-ledger.md`)
- Layout validation at ingest time (requires policy context, remains agent-driven via attestation-verification skill)
- Blocking ingestion on certification failure (evidence always stored, certification is annotation)
- Replacing the `PostureCheckHandler` (it coexists as a peer NATS subscriber)
- Certifier UI for managing/configuring certifiers (admin tooling, future work)

## Decisions

### Decision 1: Certifier interface — single function contract

```go
type Verdict string
const (
    VerdictPass  Verdict = "pass"
    VerdictFail  Verdict = "fail"
    VerdictSkip  Verdict = "skip"
    VerdictError Verdict = "error"
)

type CertResult struct {
    Certifier string
    Version   string
    Verdict   Verdict
    Reason    string
}

type Certifier interface {
    Name() string
    Version() string
    Certify(ctx context.Context, row EvidenceRow) CertResult
}
```

**Why a single `Certify` method:** Each certifier checks one thing. No lifecycle hooks, no batch APIs, no configuration DSL. The interface is the smallest useful contract. Adding a certifier is one file with one method.

**Why `skip`:** A certifier may not apply to a given evidence row (e.g., attestation certifier skips rows with no `attestation_ref`). Skip is distinct from pass — "I didn't check" is not "I checked and it's fine."

**Why `error`:** External calls can fail (OCI registry timeout). Error is distinct from fail — "I couldn't check" is retryable, "I checked and it failed" is a finding. Errors don't count against certification.

### Decision 2: Pipeline runner — sequential, fail-open

```go
type Pipeline struct {
    certifiers []Certifier
}

func (p *Pipeline) Run(ctx context.Context, row EvidenceRow) []CertResult
```

Certifiers run sequentially against each evidence row. All certifiers run regardless of prior verdicts — no short-circuiting. The pipeline returns all results.

**Why sequential, not parallel:** Certifiers are fast (local checks or single OCI HEAD). Parallelism adds complexity for negligible latency gain. Sequential is debuggable.

**Why no short-circuit:** The full picture matters. If schema fails AND provenance fails, both findings are useful. Short-circuiting hides secondary issues.

**Why fail-open:** The pipeline never blocks ingestion. Evidence is already in ClickHouse before certifiers run. Certification is annotation, not gating.

### Decision 3: CertificationHandler as NATS subscriber

The `CertificationHandler` subscribes to `studio.evidence.>` (same subject as `PostureCheckHandler`). On each `EvidenceEvent`:

1. Query ClickHouse for the new evidence rows (by `policy_id` + `ingested_at` window)
2. Run the pipeline against each row
3. Batch INSERT results to `certifications` table
4. UPDATE `evidence.certified` for affected rows

**Why query after event, not pass rows in event:** `EvidenceEvent` is lightweight (`policy_id`, `record_count`, `timestamp`). Evidence rows can be large. NATS messages should stay small. The handler queries ClickHouse directly — it already has read access.

**Why coexist with PostureCheckHandler:** Certification answers "is it legit?" Posture answers "did the pass rate change?" Different questions, different outputs, same trigger. No coupling.

### Decision 4: Certifications table — append-only operational metadata

```sql
CREATE TABLE IF NOT EXISTS certifications
(
    evidence_id       String,
    certifier         LowCardinality(String),
    certifier_version LowCardinality(String),
    result            Enum8(
                        'pass' = 1,
                        'fail' = 2,
                        'skip' = 3,
                        'error' = 4
                      ),
    reason            String,
    certified_at      DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(certified_at)
ORDER BY (evidence_id, certifier, certified_at)
```

**Why not ReplacingMergeTree:** Certifiers may re-run (version bump, retry after error). Each run appends a new row. Historical verdicts remain. The latest verdict is `ORDER BY certified_at DESC LIMIT 1` per certifier per evidence.

**Why `certifier_version`:** When certifier logic changes, the version increments. Enables queries like "show all evidence certified under schema-certifier v1 vs v2" for re-evaluation campaigns.

**Why this is not a ledger:** Same database, same credentials, mutable. It's queryable operational metadata. The path to a real ledger is `docs/decisions/transparency-ledger.md`.

### Decision 5: Denormalized `evidence.certified` column

```sql
ALTER TABLE evidence ADD COLUMN certified Bool DEFAULT false
```

Computed after pipeline run: `true` = at least one `pass` verdict AND zero `fail` verdicts across all certifiers (latest run per certifier). `skip` and `error` do not count as either pass or fail.

**Why denormalized:** The evidence table is queried on every page load. Joining `certifications` on every read is unnecessary cost. The bool is updated atomically after the pipeline completes.

**Why default false:** New evidence starts uncertified. Pre-existing evidence (before this change) is also `false` — no grandfather clause. If certifiers haven't run, the evidence is uncertified.

### Decision 6: Attestation certifier verifies from Studio's registry

The attestation certifier pulls the in-toto bundle from Studio's own OCI registry — not from `source_registry` on the evidence row. `source_registry` becomes informational metadata ("the client says it came from here"), not an address Studio acts on.

**Why not use `source_registry`:** The client controls that registry. Verifying from client-controlled infrastructure means the trust boundary is crossed. Studio must form its own opinion from data it controls.

**Prerequisite:** complyctl pushes attestation bundles to Studio's registry (or Studio mirrors them). The bundle's OCI digest (`attestation_ref`) must resolve in Studio's registry.

**Fallback:** If `attestation_ref` doesn't resolve in Studio's registry, the attestation certifier returns `fail` with reason "bundle not found in Studio registry." This is intentional — if Studio doesn't have the bundle, it can't verify it.

### Decision 7: Remove manual upload, not deprecate

`POST /api/evidence/upload` and the CSV import handler are removed entirely. The UI upload button and form are removed from `evidence-view.tsx`.

**Why remove, not deprecate:** Deprecation implies a migration period. There are no production consumers of manual upload — it was a bootstrapping convenience. Keeping it creates an unverifiable backdoor that circumvents the certifier pipeline. Clean removal.

**Evidence ingestion paths after this change:**

| Path | Structured | Certifiable |
|---|---|---|
| OTel Collector → ClickHouse | Yes (semconv) | Yes |
| `cmd/ingest` (Gemara artifacts) | Yes (schema) | Yes |
| ~~`POST /api/evidence/upload`~~ | Removed | — |
| ~~CSV import~~ | Removed | — |

### Decision 8: Three-concern separation

```
NATS: studio.evidence.{policy_id}
    │
    ├── CertificationHandler → certifications table, evidence.certified
    │   (eager, ingest-time, "is it legit?")
    │
    └── PostureCheckHandler  → notifications table
        (eager, ingest-time, "did the pass rate change?")

Agent (lazy, on-demand):
    └── posture-check skill  → evidence_assessments table
        ("does it satisfy the policy?")
```

Three distinct concerns, three distinct tables, one event trigger for the eager path. No coupling between certification and posture. Assessment remains agent-driven and independent.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| Removing manual upload breaks unknown consumers | No production consumers identified. `cmd/ingest` covers structured upload. Add back if demand surfaces. |
| Attestation certifier requires Studio-side OCI registry with mirrored bundles | Initially returns `skip` when no `attestation_ref` or `fail` when bundle not found. Functional without bundles — just fewer certifiers produce `pass`. |
| Certifiers add latency to the post-ingest path | Certifiers are async (NATS). Evidence is already stored. UI shows uncertified until pipeline completes. No user-facing latency. |
| All existing evidence starts as `certified = false` | Intentional. Run certifiers over historical data as a one-time backfill job if needed. No auto-grandfather. |
| `error` verdicts accumulate if an external dependency is persistently down | Monitor `error` count. Future: retry queue for error verdicts. Current: manual re-trigger via backfill. |
| ClickHouse `ALTER TABLE ADD COLUMN` on large evidence table | ClickHouse handles this as a metadata-only operation for default-valued columns. No rewrite. |
