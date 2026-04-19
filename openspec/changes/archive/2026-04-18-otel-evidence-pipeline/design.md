## Context

Studio's evidence store uses ClickHouse with two tables (`evaluation_logs`, `enforcement_actions`) loaded by a batch Go CLI (`cmd/ingest`). The broader ComplyTime ecosystem is standardizing on OpenTelemetry for evidence transport: `complyctl` uses ProofWatch instrumentation to emit OTLP signals, and `complytime-collector-components` defines semantic conventions (`beacon.evidence` entity) and a truthbeam enrichment processor. Studio should consume evidence via OTel rather than maintaining a parallel batch ingestion path.

The current semconv (`complytime-collector-components/model/`) defines two attribute groups (`registry.policy`, `registry.compliance`) and one entity (`beacon.evidence`). The semconv is under active development and missing several Gemara-specific attributes needed for audit-grade evidence.

## Goals / Non-Goals

**Goals:**

- Align Studio's ClickHouse schema with the `beacon.evidence` OTel entity
- Deploy an OTel Collector in the Helm chart to receive OTLP and export to ClickHouse
- Support two intake paths: Gemara-native (complyctl/ProofWatch) and raw policy engines (truthbeam enrichment)
- Document required semconv additions for upstream contribution to `complytime-collector-components`
- Retain `cmd/ingest` for local development without requiring the full OTel stack
- Update gap-analyst for single-table queries

**Non-Goals:**

- Implementing the truthbeam enrichment processor (owned by `complytime-collector-components`)
- Modifying `complyctl` or ProofWatch (separate project)
- Building a custom ClickHouse exporter (use `otel-collector-contrib` exporter)
- Real-time dashboards or metrics derived from evidence signals
- ClickHouse Cloud or multi-node deployment

## Decisions

### D1: OTel logs as the signal type

Evidence records are discrete events with structured attributes — not aggregated measurements (metrics) or causal chains (traces). Each `AssessmentLog` entry maps to one OTel `LogRecord`. The parent `EvaluationLog` context (target, policy, evaluation ID) maps to OTel Resource attributes shared across all child log records from the same evaluation sweep.

| OTel Concept | Gemara Equivalent |
|:-------------|:------------------|
| Resource | Target + evaluation context (`target.id`, `policy.id`, `evaluation.id`) |
| Scope | Control being evaluated (`control.id`, `control.result`) |
| LogRecord | Individual AssessmentLog entry |

**Alternative considered:** OTel traces with spans per assessment. Rejected — evaluations are independent checks, not causally linked operations. Trace semantics would be misleading.

### D2: Gemara CUE is schema source of truth, Weaver registers the OTel convention

Two schemas exist: Gemara CUE definitions and OTel semantic conventions (Weaver YAML). A third schema is untenable. Gemara CUE owns the data model. The Weaver definition in `complytime-collector-components/model/` is a documented alignment — it registers `policy.*` and `compliance.*` attribute names in OTel's format and enables Weaver validation/testing. It does not replace or compete with CUE.

When Gemara schema changes (new field on AssessmentLog), two things update: (1) the CUE definition, (2) the Weaver attribute mapping. The ClickHouse DDL is derived from the semconv attribute set.

**Alternative considered:** Weaver as the single source generating both OTel attributes and ClickHouse DDL. Rejected — Gemara CUE is becoming a standard; duplicating its definitions in Weaver creates a maintenance burden with no benefit.

### D3: Two intake paths, one schema

The system supports two evidence producers with different levels of Gemara awareness:

**Path A — complyctl / ProofWatch (Gemara-native):**
- Source emits full `policy.*` + `compliance.*` attributes via OTLP
- Collector passes through to ClickHouse (no enrichment needed)
- `compliance.enrichment.status` = `Success` or absent

**Path B — Raw policy engines (OPA, Gatekeeper, admission controllers):**
- Source emits `policy.*` attributes only (rule ID, result, target)
- Truthbeam processor in the collector enriches with `compliance.*` context by joining against Gemara artifacts (Policy, ControlCatalog, MappingDocuments)
- `compliance.enrichment.status` tracks whether enrichment succeeded, was partial, or unmapped

Both paths write to the same `evidence` table. The `compliance.enrichment.status` attribute serves as provenance — it indicates whether compliance context came from the source or was inferred by the collector.

**Path A and Path B are not used together on the same signal.** complyctl already provides Gemara context; running truthbeam on top would be redundant or conflicting.

**Alternative considered:** Single intake path (complyctl only). Rejected — admission controller decisions, CI scan results, and other policy engine outputs are evidence that complyctl cannot produce. Truthbeam bridges the gap.

### D4: Merge to single `evidence` table

Evaluation and remediation are the same event lifecycle. "Target failed eval at 2:25, remediated at 2:26" is a single piece of evidence, not two records in separate tables. The merged table co-locates all evidence attributes with nullable remediation columns for eval-only records.

```sql
CREATE TABLE IF NOT EXISTS evidence (
    evidence_id           String,
    target_id             String,
    target_name           Nullable(String),
    target_type           Nullable(String),
    target_env            Nullable(String),
    engine_name           Nullable(String),
    engine_version        Nullable(String),
    rule_id               String,
    rule_name             Nullable(String),
    rule_uri              Nullable(String),
    eval_result           Enum8('Not Run'=0,'Passed'=1,'Failed'=2,
                                'Needs Review'=3,'Not Applicable'=4,'Unknown'=5),
    eval_message          Nullable(String),
    policy_id             Nullable(String),
    control_id            Nullable(String),
    control_catalog_id    Nullable(String),
    control_category      Nullable(String),
    control_applicability Array(String),
    requirement_id        Nullable(String),
    plan_id               Nullable(String),
    confidence            Nullable(Enum8('Undetermined'=0,'Low'=1,'Medium'=2,'High'=3)),
    steps_executed        Nullable(UInt16),
    compliance_status     Enum8('Compliant'=0,'Non-Compliant'=1,'Exempt'=2,
                                'Not Applicable'=3,'Unknown'=4),
    risk_level            Nullable(Enum8('Critical'=0,'High'=1,'Medium'=2,
                                        'Low'=3,'Informational'=4)),
    frameworks            Array(String),
    requirements          Array(String),
    remediation_action    Nullable(Enum8('Block'=0,'Allow'=1,'Remediate'=2,
                                        'Waive'=3,'Notify'=4,'Unknown'=5)),
    remediation_status    Nullable(Enum8('Success'=0,'Fail'=1,'Skipped'=2,'Unknown'=3)),
    remediation_desc      Nullable(String),
    exception_id          Nullable(String),
    exception_active      Nullable(Bool),
    enrichment_status     Enum8('Success'=0,'Unmapped'=1,'Partial'=2,
                                'Unknown'=3,'Skipped'=4),
    collected_at          DateTime64(3),
    ingested_at           DateTime64(3) DEFAULT now64(3),
    row_key               String MATERIALIZED concat(evidence_id,'/',control_id,'/',requirement_id)
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(collected_at)
ORDER BY (target_id, policy_id, control_id, collected_at, row_key)
TTL toDateTime(collected_at) + INTERVAL 24 MONTH;
```

Every column maps to a semconv attribute. The schema IS the convention in DDL form.

**Alternative considered:** Keep two tables with collector routing by attribute presence. Rejected — forces the agent to correlate across tables and adds routing logic to the collector pipeline.

**Alternative considered:** One raw table with materialized views projecting into eval/enforcement views. Rejected — adds ClickHouse complexity for backward compatibility that isn't needed (the gap-analyst prompt is being rewritten anyway).

### D5: Collector deployment is a topology decision, not an architectural one

The OTel Collector can be deployed as:

| Pattern | Topology | When to use |
|:--------|:---------|:------------|
| Gateway | Centralized Deployment in-cluster, exposed via Service/Ingress | Multiple producers, shared enrichment config |
| Agent | Sidecar or DaemonSet alongside the evidence source | Low-latency, source-local buffering |
| Direct | Local collector on developer machine with ClickHouse exporter | Development, no cluster needed |

complytime's only responsibility is emitting OTLP to a configured endpoint. The Helm chart deploys the gateway pattern by default. Documentation covers all three patterns.

### D6: `cmd/ingest` retained for local testing

Deploying the full OTel stack (collector, ClickHouse, optionally truthbeam) for local development is heavy. `cmd/ingest` is updated to write to the merged `evidence` table and serves as a fast path: parse Gemara YAML → flatten → insert. No collector, no OTLP, no enrichment. For development and testing only.

**Alternative considered:** Remove `cmd/ingest` entirely, require OTel stack for all testing. Rejected — raises the barrier to local development significantly.

### D7: Semconv gaps require upstream contribution

The current `beacon.evidence` entity in `complytime-collector-components/model/` is missing Gemara-specific attributes. These need to be proposed upstream:

| Proposed Attribute | Type | Group | Rationale |
|:-------------------|:-----|:------|:----------|
| `compliance.policy.id` | string | `registry.compliance` | Links evidence to the Gemara L3 Policy being assessed |
| `compliance.assessment.requirement.id` | string | `registry.compliance` | Assessment granularity — one result per requirement, not per control |
| `compliance.assessment.plan.id` | string | `registry.compliance` | Ties assessment to a specific plan within the policy |
| `compliance.assessment.confidence` | enum | `registry.compliance` | Confidence level of the assessment result (High/Medium/Low/Undetermined) |
| `compliance.assessment.steps` | int | `registry.compliance` | Number of evaluation steps executed |

These extend the existing `compliance.*` namespace — they don't create a new `gemara.*` namespace. The convention stays generic enough for non-Gemara tools while supporting the full Gemara assessment model.

## Risks / Trade-offs

| Risk | Mitigation |
|:-----|:-----------|
| Breaking schema change (two tables → one) | No production data exists yet. Dev clusters re-ingest from scratch. Document migration path for future production use. |
| Semconv not yet accepted upstream | Studio can use the proposed attributes immediately. Upstream acceptance is a naming/convention concern, not a blocker. |
| ClickHouse exporter schema mismatch | The `otel-collector-contrib` ClickHouse exporter writes to its own default schema. Need to confirm it supports custom table mappings or use a processor to reshape before export. |
| Truthbeam processor not yet production-ready | Path A (complyctl/ProofWatch) works without truthbeam. Path B is additive — can be enabled when truthbeam matures. |
| `cmd/ingest` diverges from OTel schema over time | `cmd/ingest` writes to the same `evidence` table. Schema drift is caught by column mismatch at insert time. |
| Agent constructs incorrect SQL against wider table | Schema is still one table with a clear sort key. Gap-analyst prompt includes example queries. Wider table is actually simpler (no joins). |

## Open Questions

- **ClickHouse exporter configuration:** Does the `otel-collector-contrib` ClickHouse exporter support custom table schemas, or does it require the default `otel_logs` table? If not, a custom exporter or pre-export processor may be needed.
- **Truthbeam artifact access:** How does the truthbeam processor access Gemara artifacts (Policy, ControlCatalog) for enrichment? Filesystem mount, OCI pull, or API?
- **Backfill strategy:** When migrating from two tables to one, is re-ingestion from original Gemara YAML sufficient, or do we need a SQL-level migration script?
- **Collector authentication:** How does the in-cluster collector authenticate OTLP producers? mTLS, bearer tokens, or network-level trust?
