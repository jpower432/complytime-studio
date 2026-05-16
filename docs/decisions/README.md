# Architecture Decision Records

## Active

| # | Decision | Status | Date |
|:--|:--|:--|:--|
| 0001 | [PostgreSQL as Primary Persistence Layer](postgres-with-extensions.md) | Accepted | 2026-05-01 |
| 0002 | [Migrate Gateway to Echo Framework](echo-framework-migration.md) | Accepted | 2026-05-01 |
| 0003 | [Modulith Gateway Architecture](backend-architecture.md) | Superseded by #0025 | 2026-04-18 |
| 0004 | [Audit Dashboard Pivot](audit-dashboard-pivot.md) | Accepted | 2026-04-18 |
| 0005 | [Agent Interaction Model — HITL Chatbot](agent-interaction-model.md) | Accepted | 2026-04-22 |
| 0006 | [Internal Endpoint Isolation — Dual-Port Gateway](internal-endpoint-isolation.md) | Superseded | 2026-04-25 |
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
| 0022 | [Studio SPA Extraction](studio-spa-extraction.md) | Accepted | 2026-05-12 |
| 0023 | [complytime-mcp Server](complytime-mcp-server.md) | Accepted | 2026-05-12 |
| 0024 | [Agent MCP Surface — complytime-mcp vs postgres-mcp](agent-mcp-surface.md) | Accepted | 2026-05-12 |
| 0025 | [Data Platform + Workbench Split](data-platform-workbench-split.md) | Accepted | 2026-05-13 |
| 0026 | [ConnectRPC Internal API for complytime-mcp](connectrpc-internal-api.md) | Superseded | 2026-05-13 |
| 0027 | [JWT Bearer Authentication for Headless API Access](jwt-bearer-headless-auth.md) | Accepted | 2026-05-13 |
| 0028 | [Async Evidence Ingest: Accept-the-Loss Durability](async-ingest-durability.md) | Accepted | 2026-05-13 |
| 0029 | [Notifications Belong Outside the Gateway](notifications-outside-gateway.md) | Accepted | 2026-05-14 |
| 0030 | [Recommendation Engine Deferred to Workbench](recommendation-engine-deferred.md) | Deferred | 2026-05-14 |
| 0031 | [Three-Protocol Serving Layer](serving-layer-protocols.md) | Accepted | 2026-05-15 |
| 0032 | [Architecture Extraction — Core + Studio](architecture-extraction.md) | Accepted | 2026-05-15 |
| 0033 | [Evidence Quality Boundary](evidence-quality-boundary.md) | Accepted | 2026-05-15 |
| 0034 | [Unified Ingest Pipeline](unified-ingest-pipeline.md) | Accepted | 2026-05-16 |
| 0035 | [Kind + Helm as Sole Deployment Path](kind-only-deployment.md) | Accepted | 2026-05-16 |

## Active Workarounds

| # | Decision | Status | Date |
|:--|:--|:--|:--|
| 0018 | [Gemara MCP Session Initialization Failures](gemara-mcp-session-failures.md) | Active workaround | 2026-04-18 |

## Historical Workarounds (resolved by architecture change)

| # | Decision | Status | Date |
|:--|:--|:--|:--|
| 0017 | [Kagent Declarative Agent Gap Catalog](kagent-gap-catalog.md) | Obsolete — agents now run in workbench, not via kagent | 2026-04-18 |
| 0019 | [ADK Empty Messages Workaround](adk-empty-messages-workaround.md) | Obsolete — replaced by LangGraph | 2026-04-19 |

## Implemented

| # | Decision | Status | Date |
|:--|:--|:--|:--|
| 0020 | [Agent Artifact Delivery](agent-artifact-delivery.md) | Phase 1 implemented; Phase 2 deferred | 2026-04-18 |

## Superseded / Resolved / Deferred

Condensed summaries. Full content retained in files for historical reference.

| Decision | Status | Summary |
|:--|:--|:--|
| [Three-Component Architecture](three-component-architecture.md) | Superseded by #0025 | Original monorepo three-component split. Replaced by data-platform + workbench separation across repos. |
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
