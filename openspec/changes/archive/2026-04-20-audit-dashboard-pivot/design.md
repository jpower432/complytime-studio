## Context

ComplyTime Studio currently has three agents (threat-modeler, policy-composer, gap-analyst), a multi-artifact workbench editor, OCI publish/import, and a Jobs-based UX. Artifact authoring is converging on local developer tooling (Cursor, Claude Code) + gemara-mcp. Studio's authoring features duplicate that ecosystem.

Stakeholder feedback: Studio should focus on audit — storing policies, synthesizing evidence, crosswalk mappings, creating AuditLogs, and tracking compliance posture over time. The gap analyst becomes the sole agent.

**Stakeholders**: Compliance analysts (primary users), Security engineers (evidence producers), Governance leads (policy importers).

**Current architecture**:
- Preact SPA (workbench) — editor-centric, multi-artifact workspace
- Go gateway — A2A proxy, OAuth, registry proxy, evidence proxy
- kagent agents — 3 declarative agents with MCP tools
- ClickHouse — optional evidence store
- OCI registry — artifact storage

## Goals / Non-Goals

**Goals:**
- Pivot workbench from editor to audit dashboard
- Make ClickHouse the required primary data store (policies, evidence, AuditLogs, mappings)
- Support three evidence ingestion channels (OTel, REST API, file upload)
- Replace kagent gap-analyst with BYO ADK agent with deterministic gates
- Replace Jobs view with persistent chat assistant (Gemini-in-Gmail pattern)
- Store and query AuditLogs historically for trend analysis
- Support crosswalk MappingDocuments for multi-framework coverage analysis

**Non-Goals:**
- Artifact authoring (threat catalogs, control catalogs, risk catalogs, policies) — engineer's local toolchain
- OCI publishing from Studio — happens in engineer's CI/CD
- Multi-agent orchestration — single agent
- Production auth/RBAC/multi-tenancy — future work
- Real-time alerting — future work (dashboard is query-based)

## Decisions

### D1: ClickHouse as unified store

**Decision**: ClickHouse stores policies, MappingDocuments, AuditLogs, and evidence in separate tables. It becomes a required dependency, not optional.

**Rationale**: ClickHouse already handles evidence. Policies and AuditLogs are YAML documents with metadata — stored as structured rows with a `content` column holding the raw YAML. ClickHouse's columnar storage and time-series orientation fit audit history queries (posture over quarters, trend lines). One store simplifies the architecture vs. splitting across ClickHouse + PostgreSQL + object storage.

**Alternative**: PostgreSQL for policies/mappings, ClickHouse for evidence only. Rejected — adds an entire database dependency for what amounts to a document store with metadata. ClickHouse handles both workloads.

### D2: Policy import via OCI registry pull

**Decision**: Users import policies by providing an OCI reference (e.g., `ghcr.io/org/policy-bundle:v1.0`). The gateway pulls the artifact, validates it via gemara-mcp, and stores it in ClickHouse.

**Rationale**: Engineers publish to OCI from their CI/CD. Studio pulls from that same registry. The import flow already exists in the gateway (`internal/registry`). The change: artifacts go to ClickHouse instead of the browser workspace.

**Alternative**: Git sync (watch a repo for policy changes). More complex, requires webhooks or polling, and couples Studio to a specific git provider.

### D3: Three evidence ingestion channels

**Decision**:
- **OTel**: Existing `evidence-otel-intake` spec. OTLP receiver on the gateway writes to ClickHouse. Primary channel for automated evidence from CI/CD and runtime tools.
- **REST API**: New `POST /api/evidence` endpoint. Accepts JSON payloads matching the evidence schema. For programmatic integrations that don't use OTel.
- **File upload**: New `POST /api/evidence/upload` endpoint. Accepts CSV or JSON files. For manual bulk imports (e.g., spreadsheet exports from legacy tools).

**Rationale**: Different evidence producers have different capabilities. OTel covers modern tooling. REST API covers custom integrations. File upload covers the long tail.

### D4: BYO ADK gap analyst with deterministic gates

**Decision**: Replace the kagent declarative `studio-gap-analyst` with a standalone Python container using Google ADK directly. Same architecture from the `byo-author-agent` design (D2-D5), applied to the gap analyst flow:

- `before_agent_callback`: Parse policy, pre-query ClickHouse for target inventory and evidence summary, load MCP resources
- `after_agent_callback`: Validate AuditLog via gemara-mcp, completeness check (every criteria has a result), `save_artifact`
- Custom event converter for `TaskArtifactUpdateEvent`
- gemara-mcp and clickhouse-mcp as sidecars (stdio)

**Rationale**: The gap analyst is the most data-intensive agent. Deterministic pre-queries reduce token waste. Post-validation ensures every AuditLog is complete and schema-valid. Structured emission removes client-side text extraction.

### D5: Chat assistant overlay (Gemini-in-Gmail pattern)

**Decision**: Replace the Jobs view and job-scoped ChatDrawer with a persistent chat icon in the bottom-right corner of the dashboard. Clicking opens a chat overlay window. The agent is always available, not tied to discrete job creation.

**Rationale**: The Jobs abstraction adds friction (create job → pick agent → pick artifacts → wait). A persistent assistant matches the mental model: "I'm looking at my dashboard, I have a question, I ask." Context comes from what the user is viewing (current policy, current audit period, current framework).

**Alternative**: Keep Jobs as a background queue for scheduled analysis. Deferred — scheduled runs are a non-goal for this pivot. Can be added later as a server-side feature without the Jobs UI.

### D6: Dashboard layout

**Decision**: Four primary views accessible via sidebar navigation:

| View | Content |
|:--|:--|
| **Posture** | Compliance posture summary — pass/fail/gap counts per policy, trend sparklines, evidence freshness indicators |
| **Policies** | Imported policies with metadata, linked MappingDocuments, import history |
| **Evidence** | Evidence explorer — filter by policy, target, control, time range. Upload button for file ingestion. |
| **Audit History** | AuditLog timeline — browse by audit period, compare across quarters, drill into individual results |

**Rationale**: Maps to the compliance analyst workflow: "What's my current posture?" → "Which policies am I audited against?" → "What evidence do I have?" → "What happened in past audits?"

### D7: AuditLog history schema

**Decision**: AuditLogs stored in ClickHouse with:
- `audit_id` (UUID)
- `policy_id` (reference to stored policy)
- `audit_start` / `audit_end` (DateTime)
- `framework` (optional — from MappingDocument)
- `created_at` (DateTime — when the audit was run)
- `created_by` (agent or user ID)
- `content` (String — full YAML)
- `summary` (JSON — pre-computed counts: strengths, findings, gaps, observations)

**Rationale**: The `summary` column enables fast dashboard queries without parsing YAML on every page load. Full YAML in `content` preserves the complete artifact for export and drill-down.

### D8: Migration plan

**Phase 1 — Dashboard shell + evidence ingestion**
1. Add dashboard views (empty states) to workbench
2. Add REST API and file upload endpoints to gateway
3. Make ClickHouse required, add policy and AuditLog tables
4. Add chat assistant overlay (wired to existing gap-analyst)

**Phase 2 — BYO gap analyst**
5. Build BYO ADK agent container with sidecars
6. Swap declarative agent for BYO in Helm chart
7. Remove threat-modeler and policy-composer from Helm chart

**Phase 3 — Remove authoring features**
8. Remove workspace editor, artifact tabs, toolbar, gemara-version picker
9. Remove agent-picker, job lifecycle, bundle-publish
10. Clean up unused skills, prompts, and Helm templates

Phases can overlap. Phase 1 works with existing agents. Phase 3 is pure deletion.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| Removing authoring features alienates users who started there | Clear communication: authoring moved to Cursor/CC + gemara-mcp. Studio is where you see if it's working. |
| ClickHouse as document store is unconventional | Policies and AuditLogs are append-mostly, queried by time range. ClickHouse handles this well. Monitor query patterns. |
| Single agent means single point of failure | BYO agent with sidecars is self-contained. Health checks + restart policy. Deterministic gates catch most errors before the LLM runs. |
| Chat assistant context may be unclear | Inject current dashboard context (active policy, selected time range, selected framework) into chat automatically. |
| Migration is large and touches every component | Phased approach — dashboard shell first, BYO agent second, deletion third. Each phase is independently deployable. |
| Evidence ingestion at scale may stress ClickHouse | Existing retention policy (24 months). Add TTL-based partition dropping. Monitor disk usage. |

## Open Questions

1. **Scheduled gap analysis** — Should the BYO agent support cron-triggered runs in addition to interactive chat? Deferred but architecturally possible (A2A message sent by a CronJob).
2. **MappingDocument authoring** — Where do crosswalk mappings come from? Imported from registry like policies, or authored in Studio? Likely imported, but may need a simple editor.
3. **Evidence schema evolution** — The current ClickHouse schema covers EvaluationLogs and EnforcementLogs. Will new evidence types emerge? Design for extensibility via a `evidence_type` discriminator column.
