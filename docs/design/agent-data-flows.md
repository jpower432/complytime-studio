# Agent Data Flows

Data flow from the browser through Nginx and the Python Workbench to the co-located LangGraph assistant and MCP servers. The Go gateway serves only headless REST data APIs (no A2A, no SSE proxy, no artifact interception).

## End-to-End Request Flow

```mermaid
sequenceDiagram
    participant B as Browser
    participant N as Nginx (studio-ui)
    participant W as Workbench (Starlette :8090)
    participant A as Studio Assistant (LangGraph)
    participant SM as studio-mcp
    participant GM as gemara-mcp
    participant OM as oras-mcp

    B->>N: POST /workbench/a2a/studio-assistant
    N->>W: route /workbench/*
    W->>A: A2A routing (co-located)
    A->>SM: studio:// resources (evidence, policies, …)
    SM-->>A: JSON results
    A->>GM: validate_gemara_artifact / migrate_gemara_artifact
    GM-->>A: validation result
    A->>OM: registry / OCI publish (when publishing)
    OM-->>A: push result
    A-->>W: SSE events
    W-->>B: SSE (direct; not via gateway)
    Note over A,SM: Writes: ingest_evidence, save_draft_audit_log
    Note over B: Chat UI renders markdown + YAML blocks
```

## Authentication Flow

```mermaid
sequenceDiagram
    participant B as Browser
    participant N as Nginx (studio-ui)
    participant O as OAuth2 Proxy
    participant G as Gateway (Go)
    participant W as Workbench

    B->>N: user navigates (UI + API)
    N->>O: data/API paths to gateway stack
    O->>G: X-Forwarded-* headers, session
    Note over G: Headless REST only

    B->>N: /workbench/*
    N->>W: no OAuth2 Proxy on this hop

    W->>G: internal network (cluster)
    Note over W,G: Platform reads for agent/tooling;<br/>not cookie-forwarded A2A
```

| Path | Flow | Notes |
|:--|:--|:--|
| Browser → data API | Nginx → OAuth2 Proxy (gateway sidecar) → Gateway | Session-bound user context |
| Browser → workbench | Nginx → Workbench :8090 | Chat, A2A, SSE served here |
| Workbench / agent → platform | Internal → Gateway (REST APIs); parallel → studio-mcp | In-cluster; no OAuth2 Proxy on agent→gateway calls |

When OAuth is disabled, external API auth behavior follows deployment config; MCP servers use static credentials from Secrets where applicable.

## Job Lifecycle

1. User opens chat in Workbench UI, sends prompt
2. Workbench handles `POST /workbench/a2a/studio-assistant` (e.g. JSON-RPC `message/send`)
3. Co-located LangGraph assistant runs in-process to Workbench (not a separate Agent pod via gateway)
4. Assistant reads platform data via studio-mcp typed resources
5. Assistant validates / migrates YAML via gemara-mcp; publishes bundles via oras-mcp when needed
6. SSE streams from Workbench to browser
7. Chat UI renders markdown with YAML / mermaid blocks
8. Assistant persists drafts and evidence via studio-mcp tools (`save_draft_audit_log`, `ingest_evidence`)
9. User sees save / sync affordances per Workbench UX (no gateway artifact interceptor)

## Agent Response Rendering

Responses stream as SSE with markdown payloads. The chat UI renders via `renderMarkdown()`. YAML code blocks appear as formatted code in the thread.

**Artifact persistence:** The assistant persists AuditLog (and related) YAML through studio-mcp (`save_draft_audit_log`), not through the gateway. Validation uses gemara-mcp before or after persistence per agent workflow. OCI publish uses oras-mcp. Manual “save” actions in the UI remain idempotent where content-addressed keys apply.

## Assistant Capabilities

The studio assistant is a single **LangGraph** agent (not ADK) focused on audit preparation, co-located with the Workbench service. It replaces three previously planned specialist agents (threat-modeler, policy-composer, gap-analyst) cut in the [audit dashboard pivot](../decisions/audit-dashboard-pivot.md).

**Inputs:** Policy (YAML or policy_id), audit timeline, MappingDocuments (optional).

**Outputs:** AuditLog artifacts grounded in PostgreSQL evidence data.

**Skills:** studio-audit, posture-check, research, gemara.

**Tools:**

| MCP Server | Tools / Resources | Purpose |
|:--|:--|:--|
| studio-mcp | `studio://policies`, `studio://evidence`, `studio://posture`, `studio://audit-logs`, `studio://mappings`, `studio://catalogs`, `studio://threats`, `studio://risks` | Read platform data via typed resources |
| studio-mcp | `ingest_evidence`, `save_draft_audit_log` | Write evidence rows, persist draft AuditLog YAML |
| gemara-mcp | `validate_gemara_artifact`, `migrate_gemara_artifact` | Validate output, access schema/lexicon resources |
| oras-mcp | registry tools (publish, list, fetch, …) | OCI publish / browse |

## Evidence Access Patterns

The assistant reads platform data via studio-mcp typed resources (not raw SQL):

| Resource URI | Purpose |
|:--|:--|
| `studio://evidence?policy_id={id}&limit=100` | Evidence records for a policy (paginated) |
| `studio://policies/{id}` | Load policy content |
| `studio://mappings?source_catalog={catalog}` | Load cross-framework crosswalks |
| `studio://posture?policy_id={id}` | Posture aggregates for audit scope |
| `studio://catalogs` | Catalog index for control lookups |

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
