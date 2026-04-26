# Architecture Decision Records

| Decision | Status | Date |
|:--|:--|:--|
| [Audit Dashboard Pivot](audit-dashboard-pivot.md) | Accepted — implemented | 2026-04-18 |
| [ADR-001: In-Memory Session Persistence](session-persistence-storage.md) | Accepted | 2026-04-18 |
| [Backend Architecture](backend-architecture.md) | Accepted — A2A proxy embedded, extraction deferred | 2026-04-18 |
| [OTel Collector Out of Chart](otel-collector-out-of-chart.md) | Accepted | 2026-04-18 |
| [Agent Artifact Delivery](agent-artifact-delivery.md) | Accepted (Phase 1); Phase 2 deferred | 2026-04-18 |
| [Kagent Gap Catalog](kagent-gap-catalog.md) | Active — tracks upstream limitations | 2026-04-18 |
| [Gemara MCP Session Failures](gemara-mcp-session-failures.md) | Active workaround | 2026-04-18 |
| [ADK Empty Messages Workaround](adk-empty-messages-workaround.md) | Active workaround | 2026-04-19 |
| [ADK A2A Streaming](adk-a2a-streaming.md) | Resolved | 2026-04-21 |
| [OTel-Native Ingestion](otel-native-ingestion.md) | Accepted | 2026-04-21 |
| [Procedure Compliance: BPMN and Gemara](procedure-compliance-coverage.md) | Exploratory | 2026-04-21 |
| [Impact Graph: Control Failure Blast Radius](impact-graph.md) | Exploratory | 2026-04-21 |
| [Authorization Model: RACI-Scoped Multi-Tenancy](authorization-model.md) | Superseded by simple-authz | 2026-04-21 |
| [Session Token Storage](session-token-storage.md) | Proposed — OAuth token placement | 2026-04-21 |
| [Agent Interaction Model: HITL Chatbot](agent-interaction-model.md) | Accepted | 2026-04-22 |
| [Cloud-Native Posture Correction](cloud-native-posture-correction.md) | Proposed | 2026-04-24 |
| [Enforcement Log Traceability](enforcement-log-traceability.md) | Exploratory | 2026-04-24 |
| [Gemara-Native SDLC](gemara-native-sdlc.md) | Accepted | 2026-04-25 |
| [Internal Endpoint Isolation](internal-endpoint-isolation.md) | Accepted | 2026-04-25 |
| [Default Admin & Token Hardening](default-admin-token-hardening.md) | Accepted | 2026-04-25 |
| [Query Limit Cap](query-limit-cap.md) | Accepted | 2026-04-25 |

> **Related:** [Session Token Storage](session-token-storage.md) discusses OAuth access-token storage; [ADR-001](session-persistence-storage.md) covers server-side **conversation** persistence (`GET/PUT /api/chat/history`).
