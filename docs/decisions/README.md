# Architecture Decision Records

## Active

| # | Decision | Status | Date |
|:--|:--|:--|:--|
| 0001 | [PostgreSQL as Primary Persistence Layer](postgres-with-extensions.md) | Accepted | 2026-05-01 |
| 0002 | [Migrate Gateway to Echo Framework](echo-framework-migration.md) | Accepted | 2026-05-01 |
| 0003 | [Modulith Gateway Architecture](backend-architecture.md) | Accepted | 2026-04-18 |
| 0004 | [Audit Dashboard Pivot](audit-dashboard-pivot.md) | Accepted | 2026-04-18 |
| 0005 | [Agent Interaction Model — HITL Chatbot](agent-interaction-model.md) | Accepted | 2026-04-22 |
| 0006 | [Internal Endpoint Isolation — Dual-Port Gateway](internal-endpoint-isolation.md) | Accepted | 2026-04-25 |
| 0007 | [Default Admin & Token Hardening](default-admin-token-hardening.md) | Accepted | 2026-04-25 |
| 0008 | [Query Limit Cap](query-limit-cap.md) | Accepted | 2026-04-25 |
| 0009 | [Gemara-Native Security Development Lifecycle](gemara-native-sdlc.md) | Accepted | 2026-04-25 |
| 0010 | [OTel Collector Is Environment Infrastructure](otel-collector-out-of-chart.md) | Accepted | 2026-04-18 |
| 0011 | [Settings UX — Sidebar Bottom Gear Icon](settings-ux-placement.md) | Accepted | 2026-04-27 |
| 0012 | [Balanced Color Palette](muted-color-palette.md) | Accepted | 2026-04-27 |
| 0013 | [Filter Chip Pattern](filter-chip-pattern.md) | Accepted | 2026-04-26 |
| 0014 | [Evidence Staleness Model](evidence-staleness-model.md) | Accepted | 2026-04-26 |
| 0015 | [Evidence Filter Bar](evidence-filter-bar.md) | Accepted | 2026-04-26 |
| 0016 | [PII in Structured Logs](pii-in-logs.md) | Accepted (revisit at RACI Phase 3) | 2026-04-27 |
| 0021 | [Audit Workspace Inline Context](audit-workspace-inline-context.md) | Accepted | 2026-05-05 |
| 0022 | [Known Issues: Posture Donut and Recommendation Engine](known-issues-posture-recommendations.md) | Resolved | 2026-05-05 |

## Active Workarounds

| # | Decision | Status | Date |
|:--|:--|:--|:--|
| 0017 | [Kagent Declarative Agent Gap Catalog](kagent-gap-catalog.md) | Active — tracks upstream | 2026-04-18 |
| 0018 | [Gemara MCP Session Initialization Failures](gemara-mcp-session-failures.md) | Active workaround | 2026-04-18 |
| 0019 | [ADK Empty Messages Workaround](adk-empty-messages-workaround.md) | Active workaround | 2026-04-19 |

## Implemented

| # | Decision | Status | Date |
|:--|:--|:--|:--|
| 0020 | [Agent Artifact Delivery](agent-artifact-delivery.md) | Phase 1 implemented; Phase 2 deferred | 2026-04-18 |

## Superseded / Resolved / Deferred

Condensed summaries. Full content retained in files for historical reference.

| Decision | Status | Summary |
|:--|:--|:--|
| [ADK A2A Streaming](adk-a2a-streaming.md) | Resolved | Missing `AgentCapabilities(streaming=True)` and `InMemoryQueueManager`. Fixed. |
| [Authorization Model: RACI-Scoped](authorization-model.md) | Superseded | Replaced by simple admin/reviewer RBAC. Full RACI-scoped multi-tenancy deferred. |
| [Session Persistence](session-persistence-storage.md) | Accepted | In-memory session store keyed by user email. Moves to durable storage with auth sessions. |
| [Session Token Storage](session-token-storage.md) | Proposed | Server-side session store (cookie carries session ID, not token). Not implemented. |
| [Agent Trust Model](trust-model-deferred.md) | Rejected for v1 | Graduated agent autonomy rejected — self-enforcement has no separation of duties. |
| [Audit Provenance](audit-provenance-deferred.md) | Deferred | Hash-chained audit logs deferred — in-database chains lack external witness. Revisit with Trillian. |
| [External Authorization Engine](external-authz-engine.md) | Deferred | No external authz engine now. Evaluate at RACI Phase 3. |
| [Transparency Ledger](transparency-ledger.md) | Deferred | Certification verdicts in app DB, not a tamper-evident ledger. Trillian candidate when needed. |
| [OTel-Native Ingestion](otel-native-ingestion.md) | Accepted | Evidence flows through OTel Collector directly to storage. Studio has no OTLP parsing code. |

## Exploratory

Ideas evaluated but not committed to. Retained for context.

| Decision | Summary |
|:--|:--|
| [Procedure Compliance: BPMN and Gemara](procedure-compliance-coverage.md) | BPMN proves compliance through execution; Gemara through evidence. Complementary, not competing. |
| [Impact Graph: Control Failure Blast Radius](impact-graph.md) | SQL join via `mapping_entries` table. Blocked on upstream go-gemara bundle resolution. |
| [Cloud-Native Posture Correction](cloud-native-posture-correction.md) | Rewrite non-goals to reflect actual constraints. Event-driven ingestion, summary-only sovereignty. |
| [Enforcement Log Traceability](enforcement-log-traceability.md) | Ingest EnforcementLogs, link to EvaluationLog findings via justification chain. |
