## Why

Artifact authoring (ThreatCatalogs, ControlCatalogs, RiskCatalogs, Policies) is converging on local developer tooling — Cursor, Claude Code, and the Gemara MCP server. Engineers already have a software factory for producing GRC artifacts. Building a competing editor inside ComplyTime Studio duplicates that ecosystem with inferior ergonomics.

The gap: no platform synthesizes evidence, maps artifacts to compliance frameworks, tracks audit posture over time, and surfaces gaps with agentic help. ComplyTime Studio pivots from "GRC artifact editor with agents" to "audit dashboard with agentic gap analysis."

## What Changes

- **Remove authoring agents**: `studio-threat-modeler` and `studio-policy-composer` are cut. Artifact authoring belongs to the engineer's local toolchain.
- **Remove workbench editor**: The multi-artifact editor, artifact chaining, and OCI publish-from-editor flows are removed. The workbench becomes a dashboard with a minimal editor for AuditLog review/annotation only.
- **Gap analyst becomes primary agent**: `studio-gap-analyst` is the core agent, upgraded to a BYO ADK agent with deterministic gates (pre-query ClickHouse, validate cadence, completeness checks, structured artifact emission).
- **Chat assistant UX**: Replace the Jobs view with a persistent chat icon (Gemini-in-Gmail pattern) that opens a chat window overlay. The agent is always available, not job-scoped.
- **Policy store**: Import and store policies from OCI registries. Policies are the audit context — what controls and assessment requirements to evaluate against.
- **Evidence ingestion (multi-channel)**: Ingest evidence via OpenTelemetry, REST API, and file upload. ClickHouse is the evidence store.
- **Crosswalk mappings**: Store and manage MappingDocuments that link internal policy criteria to external compliance frameworks (SOC 2, ISO 27001, FedRAMP).
- **Historical AuditLogs**: Store AuditLogs over time. Enable trend analysis and drift detection across audit periods.
- **Dashboard views**: Compliance posture summary, coverage matrix per framework, evidence freshness, gap trends over quarters.

## Capabilities

### New Capabilities
- `audit-dashboard`: Dashboard views for compliance posture, coverage matrices, evidence freshness, and gap trends
- `policy-store`: Import, index, and store policies from OCI registries as audit context
- `crosswalk-mappings`: Store and manage MappingDocuments linking internal criteria to external frameworks
- `auditlog-history`: Historical AuditLog storage with trend analysis and drift detection
- `evidence-file-upload`: File-based evidence upload (CSV, JSON) into ClickHouse
- `evidence-rest-api`: REST API endpoint for programmatic evidence ingestion
- `byo-gap-analyst`: BYO ADK agent replacing the declarative gap analyst with deterministic gates and structured artifact emission
- `chat-assistant`: Persistent chat icon with overlay window replacing the Jobs view

### Modified Capabilities
- `react-workbench`: Pivot from editor-centric to dashboard-centric layout. Remove artifact tabs, toolbar, workspace editor. Add dashboard panels and chat icon.
- `evidence-ingestion`: Expand from ClickHouse-only to multi-channel (OTel + API + file upload)
- `evidence-otel-intake`: Unchanged requirements but elevated priority as a primary ingestion channel
- `registry-import`: Shift from "import artifacts into editor" to "import policies into policy store"
- `streaming-chat`: Chat window becomes a persistent overlay instead of a job-scoped drawer
- `job-lifecycle`: **REMOVED** — replaced by chat-assistant. No more discrete jobs.
- `workspace-editor`: **REMOVED** — replaced by minimal AuditLog viewer/annotator
- `artifact-workspace`: **REMOVED** — replaced by policy-store
- `bundle-publish`: **REMOVED** — publishing happens in engineer's CI/CD, not Studio
- `agent-picker`: **REMOVED** — single agent, no picker needed
- `agent-spec-skills`: **REMOVED** — authoring skills no longer relevant
- `platform-prompt-composition`: Simplified to single agent platform prompt
- `gemara-version-select`: **REMOVED** — validation happens in engineer's toolchain

## Impact

- **Workbench SPA**: Major rewrite — dashboard panels replace editor. Chat overlay replaces jobs view.
- **Gateway**: Simplified — fewer agents to proxy, new REST endpoints for evidence API and policy store.
- **Helm chart**: Remove threat-modeler and policy-composer agent CRDs and model configs. Add BYO gap analyst deployment with sidecars. Remove authoring-related MCP server configurations.
- **ClickHouse**: Elevated from optional to required. Schema additions for policy store and AuditLog history.
- **Agent directory**: Single entry (`studio-gap-analyst`).
- **Skills**: Remove `gemara-authoring` and `risk-reasoning` from this repo. They belong in the gemara-mcp ecosystem.
- **Dependencies**: Add `google-adk` and `mcp` Python SDK for BYO agent container.
- **Existing specs**: ~8 specs removed, ~5 modified, ~8 new.
