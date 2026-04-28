## Why

The agent is positioned as the primary analyst interface — the [audit-dashboard-pivot](../../docs/decisions/audit-dashboard-pivot.md) describes a "persistent chat assistant" and the evidence-attestation-pipeline design lists "Dashboards" as a non-goal. This forces all analytical workflows through conversation, which is slow for routine questions ("what's our SOC 2 posture?") and produces non-exportable, non-bookmarkable results.

The [cloud-native posture correction](../../docs/decisions/cloud-native-posture-correction.md) repositions the agent as a power tool that augments structured dashboard views rather than replacing them.

## What Changes

- **Dashboard-first posture**: PostureView, requirement matrix, and evidence browser are the primary analyst interface. They show live ClickHouse aggregates, not agent-produced summaries.
- **Agent as contextual assistant**: The chat overlay remains but is repositioned as a synthesis and deep-analysis tool. Analysts use it for questions the grid can't answer: "Why did our posture drop this week?", "Draft the executive summary for the Q1 audit", "Which controls are covered by both SOC 2 and NIST 800-53?"
- **View-to-agent handoff**: Structured views provide "Ask the agent" affordances that pre-populate the chat with context (current policy, selected control, filtered evidence set). The agent receives structured context, not a cold-start conversation.
- **Agent output to view**: Agent-produced artifacts (AuditLogs, EvidenceAssessments) update the dashboard views in real time via existing SSE stream + auto-persistence. The agent's work is reflected in the grid, not trapped in the chat.
- **Canned queries**: Common analytical questions available as one-click buttons in the chat overlay: "Run posture check", "Generate AuditLog", "Summarize gaps". Reduces the learning curve for analysts unfamiliar with the agent.

## Capabilities

### New Capabilities
- `view-to-agent-context`: Dashboard views inject structured context (policy, control, evidence filters) into agent conversations via the existing `buildInjectedContext` mechanism
- `canned-queries`: Pre-defined one-click prompts for common analytical workflows

### Modified Capabilities
- `posture-view`: Shows live ClickHouse aggregates from `policy_posture` view instead of agent-produced summary cards
- `chat-assistant`: Repositioned as contextual overlay with pre-populated context from the active view

### Removed Capabilities
- `agent-as-dashboard`: The pattern of using the agent as the primary data presentation layer is retired

## Impact

- **Workbench**: PostureView refactored to query `policy_posture` and `unified_compliance_state` views directly via REST. Chat overlay gains context injection from active view and canned query buttons.
- **Gateway**: New REST endpoint for posture aggregates (`GET /api/posture`) returning `policy_posture` view data as JSON. Existing A2A proxy unchanged.
- **Agent**: No changes to the agent itself — prompt, skills, and tools remain the same. The change is in how the workbench presents and invokes the agent.
- **ClickHouse**: No schema changes — relies on existing `policy_posture` and `unified_compliance_state` views.

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS

Dashboard views and the agent operate on the same data independently. Neither depends on the other for core functionality. Context injection is additive, not required.

### II. Composability First

**Assessment**: PASS

Each view is a standalone route. The agent is a standalone service. Context handoff is a workbench concern, not an architectural coupling.

### III. Observable Quality

**Assessment**: PASS

Dashboard data traces to ClickHouse views with full provenance. Agent artifacts are persisted and reflected in views with `assessed_by` and `assessed_at` metadata.

### IV. Testability

**Assessment**: PASS

Posture REST endpoint testable with seeded data. View-to-agent context injection testable via workbench integration tests. Canned queries testable as pre-formed A2A requests.
