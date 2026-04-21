## 1. ClickHouse schema and requirements

- [x] 1.1 Make ClickHouse a required dependency — set `clickhouse.enabled: true` as default in `values.yaml`, add gateway readiness check for ClickHouse connectivity
- [x] 1.2 Add `policies` table to ClickHouse schema (policy_id, title, version, oci_reference, content, imported_at, imported_by)
- [x] 1.3 Add `mapping_documents` table (mapping_id, policy_id, framework, content, imported_at)
- [x] 1.4 Add `audit_logs` table (audit_id, policy_id, audit_start, audit_end, framework, created_at, created_by, content, summary)

## 2. Gateway — policy store and evidence API

- [x] 2.1 Add `POST /api/policies/import` endpoint — pull OCI artifact, validate via gemara-mcp, store in ClickHouse `policies` table
- [x] 2.2 Add `GET /api/policies` endpoint — list stored policies with metadata
- [x] 2.3 Add `GET /api/policies/:id` endpoint — return full policy YAML and linked mapping documents
- [x] 2.4 Add `POST /api/mappings/import` endpoint — pull OCI artifact, validate, store in `mapping_documents`, link to policy
- [x] 2.5 Add `POST /api/evidence` endpoint — accept JSON array of evidence records, validate, insert into ClickHouse
- [x] 2.6 Add `POST /api/evidence/upload` endpoint — accept CSV/JSON file, parse, validate, insert into ClickHouse with partial failure reporting
- [x] 2.7 Add `GET /api/evidence` endpoint — query with filters (policy_id, target_id, control_id, start, end, limit, offset)
- [x] 2.8 Add `POST /api/audit-logs` endpoint — store AuditLog YAML with pre-computed summary JSON
- [x] 2.9 Add `GET /api/audit-logs` endpoint — query by policy_id, time range, with summary counts

## 3. BYO gap analyst agent

- [x] 3.1 Create `agents/gap-analyst/Dockerfile` — Python container with google-adk, mcp SDK, pyyaml, bundled skills
- [x] 3.2 Create `agents/gap-analyst/main.py` — `LlmAgent` + `A2aAgentExecutor` with custom event converter, two `McpToolset` instances (gemara-mcp with `use_mcp_resources=True`, clickhouse-mcp)
- [x] 3.3 Implement `before_agent_callback` — parse policy reference and audit timeline from user message, validate inputs exist, pre-query ClickHouse for target inventory and evidence summary, load MCP resources, inject structured context
- [x] 3.4 Implement `after_agent_callback` — extract YAML from output, validate via gemara-mcp `validate_gemara_artifact` (#AuditLog), check completeness (every criteria has AuditResult), `save_artifact` on success, return errors for retry on failure (max 3)
- [x] 3.5 Implement custom `EventConverter` subclass — detect `artifact_delta` on events, load artifact from service, emit `TaskArtifactUpdateEvent` with `application/yaml` MIME type and filename metadata
- [x] 3.6 Implement `before_tool_callback` for ClickHouse SQL sanitization — restrict to SELECT-only, reject DDL/DML
- [x] 3.7 Add Helm template `byo-gap-analyst.yaml` — Deployment with agent container + gemara-mcp sidecar + clickhouse-mcp sidecar, Service on port 8080, model config via env vars

## 4. Workbench — dashboard layout

- [x] 4.1 Replace workbench SPA layout — remove artifact tabs, toolbar, workspace editor. Add sidebar navigation with four views (Posture, Policies, Evidence, Audit History)
- [x] 4.2 Implement Posture view — cards per policy with pass/fail/gap/observation counts from most recent AuditLog, evidence freshness indicator, trend sparklines
- [x] 4.3 Implement Policies view — table of stored policies (title, version, import date, mapping count), import button, policy detail with read-only YAML viewer and linked mappings
- [x] 4.4 Implement Evidence view — filterable table (policy, target, control, time range), file upload drop zone with progress and import summary
- [x] 4.5 Implement Audit History view — timeline of AuditLogs per policy, summary cards, quarter-over-quarter comparison, drill-down into individual AuditResults

## 5. Chat assistant overlay

- [x] 5.1 Add persistent chat icon component — bottom-right corner, all views, toggles overlay visibility
- [x] 5.2 Implement chat overlay window — message input, conversation history, streaming text rendering, conversation persistence in browser storage
- [x] 5.3 Implement dashboard context injection — detect current view and active selections (policy_id, time range, framework), inject as metadata in A2A messages
- [x] 5.4 Implement structured artifact handling — detect `TaskArtifactUpdateEvent` with `application/yaml`, render artifact card with YAML preview and "Save to Audit History" button that calls `POST /api/audit-logs`
- [x] 5.5 Wire chat to BYO gap analyst A2A endpoint — use existing `streamMessage`/`streamReply` functions, remove agent picker and job creation flow

## 6. Helm chart cleanup

- [x] 6.1 Remove `studio-threat-modeler` and `studio-policy-composer` from `agent-specialists.yaml` and `model-config.yaml`
- [x] 6.2 Remove authoring-related entries from `agentDirectory` in `values.yaml` — keep only gap-analyst
- [x] 6.3 Update `agentDirectory` to point to BYO gap analyst deployment instead of kagent-managed pod
- [x] 6.4 Remove `skills/gemara-authoring` and `skills/risk-reasoning` from repo (move to gemara-mcp ecosystem)
- [x] 6.5 Remove unused agent prompts from `charts/complytime-studio/agents/threat-modeler/` and `charts/complytime-studio/agents/policy-composer/`
- [x] 6.6 Simplify `platform.md` — remove multi-agent references, `{{.AgentName}}` template, cross-agent coordination

## 7. Documentation and gap catalog

- [x] 7.1 Update `README.md` — new product description (audit dashboard, not GRC editor), updated architecture diagram, revised setup instructions
- [x] 7.2 Update `AGENTS.md` — single agent, BYO pattern, no kagent CRD instructions for authoring agents
- [x] 7.3 Create `docs/decisions/audit-dashboard-pivot.md` — decision record capturing the pivot rationale, what was cut, and why
- [x] 7.4 Create `docs/decisions/kagent-gap-catalog.md` — document all 9 kagent limitations discovered, with code evidence and upstream issue templates
