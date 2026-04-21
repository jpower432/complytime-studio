# Agent Data Flows

Data flow diagrams for ComplyTime Studio, traced from the workbench UI through the gateway to the studio assistant and its backend services.

## End-to-End Request Flow

```
Browser (Workbench SPA)
    │
    │  POST /api/a2a/studio-assistant
    │  Authorization: Bearer {token} (when OAuth enabled)
    │
    ▼
Gateway (Go)
    │  Auth middleware → decode cookie → inject Bearer header
    │  A2A proxy → forward to agent pod
    │
    ▼
Studio Assistant (Python ADK, :8080)
    │  LlmAgent processes prompt with skills context
    │  Calls MCP tools as needed:
    │  ├── clickhouse-mcp → run_select_query (evidence, policies, mappings)
    │  └── gemara-mcp → validate_gemara_artifact, load resources
    │
    │  SSE event stream (TaskStatusUpdateEvent, TaskArtifactUpdateEvent)
    │
    ▼
Gateway (passthrough)
    │
    ▼
Browser
    ChatDrawer receives SSE events
    extractArtifacts() scans for YAML code blocks
    Artifacts proposed to WorkspaceView editor
```

## Authentication Flow

```
Browser → Gateway → Agent Pod → MCP Server

  cookie    decode     inject      allowedHeaders
            session    Bearer      propagates
                       header      Authorization
```

When OAuth is disabled, no token propagation occurs. MCP servers use static credentials from Secrets.

## Job Lifecycle

```
1. User opens Chat Drawer, types prompt
2. Workbench POST /api/a2a/studio-assistant (JSON-RPC: message/send)
3. Gateway proxies to studio-assistant pod
4. Assistant queries ClickHouse via clickhouse-mcp (evidence, policies)
5. Assistant produces AuditLog YAML, validates via gemara-mcp
6. SSE events stream back through gateway to browser
7. ChatDrawer extracts YAML artifacts from response text
8. Artifacts appear as proposals in WorkspaceView editor
9. User reviews, edits, validates, publishes
```

## Artifact Extraction

Agent responses contain Gemara YAML embedded in markdown. The workbench extracts artifacts client-side.

```
Agent response text
    │
    ▼
extractArtifacts(text)
    ├── Scan for ```yaml fenced blocks
    ├── Fallback: detect inline YAML (metadata:, threats:, etc.)
    │
    ▼
isGemaraArtifact(yaml)?
    ├── YES → detectDefinition(yaml) → inferArtifactName(yaml)
    │         → proposeArtifact to editor
    └── NO  → keep in prose text
```

Recognized artifact keys: `threats`, `controls`, `capabilities`, `guidances`, `policy`, `results`, `risks`, `mappings`, `metadata`.

## Post-Authoring Actions

```
YAML in Editor
    ├── Validate    → POST /api/validate (gemara-mcp)
    ├── Download    → browser Blob API
    ├── Copy        → clipboard API
    ├── Publish     → POST /api/publish (OCI bundle)
    └── Import      → GET /api/registry/* (browse + inject mapping ref)
```

## Assistant Capabilities

The studio assistant is a single BYO ADK agent focused on audit preparation. It replaces three previously planned specialist agents (threat-modeler, policy-composer, gap-analyst) that were cut in the [audit dashboard pivot](../decisions/audit-dashboard-pivot.md).

**Inputs:** Policy (YAML or policy_id), audit timeline, MappingDocuments (optional).

**Outputs:** AuditLog artifacts grounded in ClickHouse evidence data.

**Skills:** gemara-mcp, evidence-schema, audit-methodology, coverage-mapping.

**Tools:**

| MCP Server | Tools Used | Purpose |
|:--|:--|:--|
| clickhouse-mcp | `run_select_query`, `list_tables` | Query evidence, policies, mappings |
| gemara-mcp | `validate_gemara_artifact`, `migrate_gemara_artifact` | Validate output, access schema/lexicon resources |

## Evidence Query Patterns

The assistant uses these ClickHouse queries via clickhouse-mcp:

| Query | Purpose |
|:--|:--|
| `SELECT DISTINCT target_id, target_name, count(*) FROM evidence WHERE policy_id = ? AND collected_at BETWEEN ? AND ?` | Derive target inventory for audit scope |
| `SELECT * FROM evidence WHERE policy_id = ? AND target_id = ? AND collected_at BETWEEN ? AND ?` | Per-target evidence for assessment |
| `SELECT * FROM policies WHERE policy_id = ?` | Load policy content |
| `SELECT * FROM mapping_documents WHERE policy_id = ?` | Load cross-framework crosswalks |

## Cross-Framework Coverage

When MappingDocuments are available, the assistant maps internal audit results to external framework entries.

| AuditResult Type | Mapping Strength | Framework Coverage |
|:--|:--|:--|
| Strength | 8-10 | Covered |
| Strength | 5-7 | Partially Covered |
| Strength | 1-4 | Weakly Covered |
| Finding | any | Not Covered (finding) |
| Gap | any | Not Covered (no evidence) |
| Observation | any | Needs Review |
| (no mapping) | — | Unmapped |
