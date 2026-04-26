## Why

Studio's UX is built around a pull model — the user searches for evidence, triggers audits, and navigates between disconnected views. The primary persona is the **audit liaison**: someone who prepares evidence, coordinates with system owners, and presents to auditors. They need to know what arrived, what's missing, and what it means — not hunt for it.

The current nav is confusing: "Review" doesn't convey its purpose (agent draft inbox), "Audit History" is buried as a top-level view when it's a per-policy drill-down, and posture cards don't communicate that each card represents an inventory item with its own targets, controls, and evidence lifecycle. Evidence arrives from pipelines but nothing reacts to it.

## What Changes

- Rename "Review" → **"Inbox"** — the agent's draft queue plus event notifications
- Restructure Audit History as a **breadcrumb drill-down** under Posture (Posture > Policy > History) instead of a standalone nav item
- Add **inventory context** to posture cards: target count, control count, last evidence timestamp, owner information from Policy RACI contacts
- Introduce **event-driven evidence reactions**: new evidence triggers a lightweight agent posture check, results surface as inbox notifications
- Add a **NATS event bus** between the ingest pipeline and the gateway so evidence arrivals are observable in real-time
- Surface **risk severity overlay** (Phase 2) on posture cards and requirement matrix rows using the existing `risks`/`risk_threats` graph data

## Capabilities

### New Capabilities
- `inbox-view`: Unified inbox replacing "Review" — shows agent drafts, posture change notifications, and evidence arrival events with badge count
- `posture-drilldown`: Breadcrumb navigation pattern replacing standalone Audit History — Posture > [Policy] shows requirements, evidence timeline, and audit history as tabs within a single policy context
- `inventory-cards`: Enhanced posture cards showing target inventory, control scope, evidence freshness, and RACI owner from Policy contacts
- `evidence-event-bus`: NATS-based event bus emitting evidence arrival events from the ingest pipeline, consumed by the gateway to trigger agent posture checks and inbox notifications
- `risk-severity-overlay`: Risk severity indicators on posture cards and requirement matrix rows, driven by `risks`/`risk_threats` graph queries

### Modified Capabilities
- `react-workbench`: Navigation restructured — sidebar loses Audit History, gains Inbox with badge; main view routing adds breadcrumb drill-down pattern
- `streaming-chat`: Chat becomes the primary work surface — pre-loaded context from the active policy/view when user opens chat
- `posture-check-skill`: Agent auto-triggers on evidence events instead of only on-demand

## Impact

- **Frontend**: Sidebar, app router, posture-view, audit-history-view, draft-review-view all refactored. New inbox component. Breadcrumb component added.
- **Backend**: New NATS dependency (`nats.go`). Gateway subscribes to evidence events. New `/api/inbox` endpoint for notifications. `cmd/ingest` publishes to NATS on evidence insert.
- **Helm**: NATS server added to chart (or external NATS reference). Gateway env vars for NATS connection. Ingest env vars for NATS publish.
- **Agent**: Posture-check skill gains an event-triggered mode. Inbox notifications stored in ClickHouse or in-memory.
- **Dependencies**: `github.com/nats-io/nats.go` (CNCF graduated project, Go-native, lightweight).
