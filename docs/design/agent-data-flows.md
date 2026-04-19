# Agent Data Flow Architecture

Data flow diagrams for ComplyTime Studio's multi-agent platform, traced from the workbench UI through the gateway to each specialist agent and its backend services.

## System Architecture

```mermaid
graph TB
    subgraph Browser
        subgraph Workbench["Workbench (Preact SPA)"]
            MV[JobsView<br/>agent picker, new job]
            WV[WorkspaceView<br/>YAML editor, validate]
            CD[ChatDrawer<br/>SSE stream, reply]
            ID[ImportDialog<br/>OCI browser, import ref]

            MV & WV & CD & ID --> State
            State["Shared State<br/>jobs (localStorage)<br/>editor (Preact signals)"]
            State --> API["API Layer (apiFetch)"]
        end
    end

    API -->|HTTPS<br/>studio_session cookie| GW

    subgraph GW["Gateway (Go binary, cmd/gateway)"]
        Auth["Auth Middleware<br/>cookie → HMAC verify → decode Session → inject ctx<br/>401 on /api/* if missing/expired"]
        Auth --> Routes

        subgraph Routes["Route Table"]
            A2A["/api/a2a/{name}<br/>ReverseProxy → agent pod<br/>inject Bearer token<br/>SSE passthrough"]
            VAL["/api/validate, /api/migrate<br/>gemara-mcp proxy"]
            REG["/api/registry/*<br/>oras-mcp proxy or direct HTTP"]
            PUB["/api/publish<br/>oras-go + optional cosign"]
            CFG["/api/config<br/>platform configuration"]
            AG["/api/agents<br/>static JSON"]
            SPA["/<br/>embedded SPA (go:embed)"]
        end
    end

    A2A -->|"http://{name}:8080/invoke"| Agents
    VAL -->|MCP stdio via sidecar| GemaraMCP
    REG -->|MCP stdio via sidecar| OrasMCP

    subgraph Agents["Agent Pods (kagent)"]
        TM[studio-threat-modeler]
        PC[studio-policy-composer]
        GA[studio-gap-analyst]
    end

    subgraph MCP["MCP Servers"]
        GemaraMCP["gemara-mcp<br/>(stdio)"]
        GitHubMCP["github-mcp<br/>(http, OBO)"]
        OrasMCP["oras-mcp<br/>(stdio)"]
        CHMCP["clickhouse-mcp<br/>(stdio)"]
    end

    subgraph Infra["Infrastructure"]
        CH[("ClickHouse<br/>(StatefulSet)")]
        OCI[("OCI Registry<br/>(external)")]
    end

    TM & PC & GA --> GemaraMCP
    TM & PC & GA --> GitHubMCP
    GA --> CHMCP
    CHMCP --> CH
    PUB --> OCI
    OrasMCP --> OCI
```

## Authentication & On-Behalf-Of Flow

All agent communication routes through the gateway. The user's GitHub token propagates end-to-end. github-mcp uses `http` transport — each request carries the calling user's `Authorization: Bearer` header. No static `GITHUB_PERSONAL_ACCESS_TOKEN` exists in the deployment.

```mermaid
sequenceDiagram
    participant B as Browser
    participant GW as Gateway
    participant AP as Agent Pod
    participant GH as github-mcp

    B->>GW: Request with studio_session cookie<br/>(HMAC-signed JWT)
    GW->>GW: Decode cookie<br/>Extract Session.GitHubToken

    GW->>AP: Forward request<br/>Authorization: Bearer {token}

    AP->>GH: MCP tool call<br/>allowedHeaders: [Authorization]<br/>kagent propagates Bearer token

    GH->>GH: GitHub API calls<br/>on behalf of user

    GH-->>AP: Tool result
    AP-->>GW: SSE event stream
    GW-->>B: SSE event stream (passthrough)
```

## Workbench Data Flow

### Job Lifecycle

```mermaid
sequenceDiagram
    participant U as User
    participant JV as JobsView
    participant API as API Layer
    participant WV as WorkspaceView
    participant CD as ChatDrawer
    participant LS as localStorage

    Note over JV: 1. DISCOVER AGENTS
    JV->>API: fetchAgents()
    API-->>JV: AgentCard[] {name, description, skills[]}

    Note over JV: 2. CREATE JOB
    U->>JV: Select agent + type prompt
    JV->>API: sendMessage(text, agentName)<br/>POST /api/a2a/{agent}<br/>JSON-RPC 2.0: message/send
    API-->>JV: { result: { id: taskId } }

    Note over JV: 3. STORE JOB
    JV->>LS: createJob(taskId, title, agentName)
    JV->>LS: addMessage(taskId, "user", text)
    JV->>WV: navigate("workspace", jobId)

    Note over CD: 4. STREAM RESPONSES
    CD->>API: streamTask(taskId)<br/>EventSource: GET /api/a2a/{agent}?taskId={id}

    loop SSE Events
        API-->>CD: status / message / artifact events

        Note over CD: 5. PROCESS EVENTS
        CD->>CD: extractArtifacts(text)
        CD->>LS: addMessage (prose)
        CD->>LS: addArtifact (YAML)
        CD->>WV: setEditorArtifact → editor updates live
    end

    Note over CD: 6. USER REPLIES (multi-turn)
    U->>CD: Type reply
    CD->>LS: addMessage(taskId, "user", text)
    CD->>API: sendReply(taskId, text, agentName)<br/>POST /api/a2a/{agent} with taskId
```

### Artifact Detection Pipeline

When agent responses arrive, the workbench extracts Gemara artifacts from markdown code blocks.

```mermaid
flowchart TD
    A[Agent response text] --> B["extractArtifacts(text)"]
    B --> C{"Scan for<br/>```yaml``` blocks"}
    C --> D{isGemaraArtifact?<br/>matches: threats, controls,<br/>capabilities, guidances, policy,<br/>results, risks, mappings, metadata}
    D -->|YES| E["detectDefinition(yaml)<br/>1. Parse metadata.type field<br/>2. Fallback: match top-level keys<br/>3. Map to CUE definition"]
    E --> F["inferArtifactName(yaml)"]
    F --> G["{name, yaml, definition}"]
    G --> H["addArtifact(jobId, ...)"]
    G --> I["setEditorArtifact(...)"]
    D -->|NO| J[Keep in prose text]
    H & I & J --> K["{text: cleaned prose,<br/>artifacts: ExtractedArtifact[]}"]
```

### Post-Authoring Actions

```mermaid
flowchart LR
    Editor[YAML in Editor] --> Validate["Validate<br/>POST /api/validate"]
    Editor --> Download["Download YAML<br/>browser Blob API"]
    Editor --> Copy["Copy<br/>clipboard API"]
    Editor --> Publish["Publish<br/>POST /api/publish"]
    Editor --> Import["Import<br/>GET /api/registry/*"]
    Import --> Inject["injectMappingReference()<br/>into editor YAML"]
```

## Agent Data Flows

### Shared Agent Infrastructure

All agents share this deployment pattern on Kubernetes.

```mermaid
graph TB
    subgraph CRD["kagent Declarative Agent CRD"]
        direction TB
        RT["runtime: go<br/>modelConfig: studio-model → ModelConfig CRD"]

        subgraph Prompt["Prompt Composition"]
            P1["1. platform.md<br/>(ConfigMap: studio-platform-prompts)<br/>Identity, constraints, validation rules"]
            P2["2. prompt.md<br/>(embedded via .Files.Get)<br/>Agent-specific workflow steps"]
            P3["3. skills/<br/>(mounted via gitRefs)<br/>Domain knowledge loaded on demand"]
        end

        subgraph Tools["MCP Tool Access"]
            T1["gemara-mcp | stdio | None (static)"]
            T2["github-mcp | http | OBO (allowedHeaders)"]
            T3["clickhouse-mcp | stdio | Static credentials (Secret)"]
        end

        A2A["A2A skills[] → /api/agents → workbench agent picker"]
    end
```

### Agent 1: studio-threat-modeler (Layer 2 — Controls)

STRIDE-based threat analysis. Consumes GitHub repository content, produces ThreatCatalog and ControlCatalog artifacts.

```mermaid
flowchart TB
    Input["User prompt:<br/>'Analyze threats for github.com/kyverno/kyverno'<br/><br/>No prerequisite Gemara artifacts required.<br/>Upstream L1 artifacts accepted as optional context."]

    Input --> S1

    subgraph Workflow["Agent Workflow"]
        S1["Step 1: GATHER CONTEXT<br/>github-mcp → get_file_contents<br/>Dockerfiles, K8s manifests, CI configs, go.mod<br/>github-mcp → search_code<br/>Dependency files, security patterns<br/>Authorization: user's GitHub token (OBO)"]

        S2["Step 2: ANALYZE<br/>Load skill: stride-analysis<br/>(git clone from openssf/stride-skills)<br/><br/>Map capabilities from repo content<br/>Per capability → evaluate STRIDE:<br/>S, T, R, I, D, E<br/>Skip categories with no meaningful threat"]

        S3["Step 3: AUTHOR THREATCATALOG<br/>gemara-mcp → threat_assessment prompt<br/>Write YAML<br/>gemara-mcp → validate(#ThreatCatalog)<br/>fix → re-validate (max 3x)"]

        S4["Step 4: AUTHOR CONTROLCATALOG (when requested)<br/>gemara-mcp → control_catalog prompt<br/>Each control references threats it mitigates<br/>gemara-mcp → validate(#ControlCatalog)<br/>fix → re-validate (max 3x)"]

        S5[Step 5: Return validated YAML]

        S1 --> S2 --> S3 --> S4 --> S5
    end

    S5 --> Output

    Output["Artifacts: ThreatCatalog, ControlCatalog<br/>Conversation: ~1-2 turns<br/>Tools: github-mcp, gemara-mcp<br/>Skills: gemara-layers, stride-analysis"]
```

### Agent 2: studio-policy-composer (Layer 3 — Policy)

RiskCatalog and Policy authoring through guided two-phase conversation. Facilitates; derives defaults from input artifacts.

```mermaid
flowchart TB
    Input["REQUIRED:<br/>ThreatCatalog (from threat-modeler)<br/>ControlCatalog (from threat-modeler)<br/><br/>OPTIONAL:<br/>GuidanceCatalog<br/><br/>If either required input missing → agent halts"]

    Input --> P1

    subgraph Phase1["Phase 1: Risk Catalog"]
        P1["Step 1: Derive RiskCategories<br/>Examine ThreatCatalog.groups<br/>→ propose candidates with appetite<br/>→ present table for user confirmation"]

        P2["Step 2: Derive Risk Entries<br/>Per threat → derive Risk with severity<br/>→ present summary table"]

        P3["Step 3: Validate<br/>gemara-mcp → validate(#RiskCatalog)<br/>Confirm with user before Phase 2"]

        P1 --> P2 --> P3
    end

    P3 --> P4

    subgraph Phase2["Phase 2: Policy"]
        P4["Step 4: RACI contacts"]
        P5["Step 5: Scope (derive from ControlCatalog)"]
        P6["Step 6: Import ControlCatalog"]
        P7["Step 7: Risk-to-Control Linkage<br/>Load skill: policy-risk-linkage<br/>Join risks ↔ controls on shared threat refs<br/>Mitigated: ≥1 shared threat ref<br/>Accepted: no controls + justification<br/>Unmitigated: awaiting decision"]
        P8["Step 8: Assessment plans"]
        P9["Step 9: Enforcement"]
        P10["Step 10: Implementation timeline"]
        P11["Step 11: Validate<br/>gemara-mcp → validate(#Policy)"]

        P4 --> P5 --> P6 --> P7 --> P8 --> P9 --> P10 --> P11
    end

    P11 --> Output

    Output["Artifacts: RiskCatalog, Policy<br/>Conversation: ~8-10 turns (guided)<br/>Tools: github-mcp, gemara-mcp<br/>Skills: gemara-layers, policy-risk-linkage, assessment-defaults"]
```

### Agent 3: studio-gap-analyst (Layer 7 — Audit)

Combined audit preparation assistant. Derives target inventory from evidence, assesses criteria coverage per target, translates coverage through MappingDocuments to external compliance frameworks using strength and confidence scores.

```mermaid
flowchart TB
    Input["REQUIRED:<br/>Policy (YAML or policy_id)<br/>Audit timeline (start, end) — user-provided<br/>MappingDocuments (1..N, one per framework)<br/><br/>Without MappingDocuments: internal-only analysis<br/>Without Policy or timeline: agent halts"]

    Input --> S1

    subgraph Phase1["Phase 1: Scope & Inventory"]
        S1["Step 1: Confirm audit scope<br/>Policy, timeline, frameworks → summary table"]

        S2["Step 2: Derive target inventory<br/>clickhouse-mcp → run_select_query<br/>SELECT DISTINCT target_id, target_name,<br/>min(collected_at), max(collected_at), count(*)<br/>FROM evidence<br/>WHERE policy_id = ? AND collected_at BETWEEN ? AND ?<br/>→ present inventory, user confirms targets"]

        S3["Step 3: Load criteria<br/>Parse Policy → imports.catalogs[]<br/>→ extract controls + assessment requirements<br/>= complete criteria set"]

        S1 --> S2 --> S3
    end

    S3 --> S4

    subgraph Phase2["Phase 2: Evidence Assessment (per target)"]
        S4["Step 4: Query evidence<br/>clickhouse-mcp → run_select_query<br/>evidence table (by policy_id, target_id, timeline)<br/>Evaluations and remediations co-located in single table"]

        S5["Step 5: Validate assessment cadence<br/>Policy.adherence.assessment-plans[].frequency<br/>→ compute expected cycles across audit window<br/>→ compare actual timestamps<br/>→ MISSING CYCLES = FINDING (non-compliance)<br/><br/>Example: policy=daily, window=90d<br/>Expected: 90 — Found: 87<br/>Gaps: Feb 12, Mar 3, Mar 17 → 3 Findings"]

        S6["Step 6: Classify each criteria entry<br/>Load skill: audit-classification"]

        S7["Step 7: Assemble AuditLog<br/>One AuditResult per criteria entry<br/>gemara-mcp → validate(#AuditLog)<br/>fix → re-validate (max 3x)"]

        S4 --> S5 --> S6 --> S7
    end

    S7 --> S8

    subgraph Phase3["Phase 3: Cross-Framework Coverage"]
        S8["Step 8: Join AuditResults with MappingDocuments<br/>AuditResult.criteria-reference → Mapping.source<br/>→ Mapping.targets[].entry-id (external framework)<br/>→ read .strength (1-10) and .confidence-level"]

        S9["Step 9: Classify framework coverage<br/>Multiple controls → same entry: strongest wins"]

        S10["Step 10: Present coverage matrix<br/>Per-framework summary (counts by status)<br/>Attention items (sorted by risk)<br/>Embed framework context in AuditResult.recommendations[]"]

        S8 --> S9 --> S10
    end

    S10 --> S11

    subgraph Phase4["Phase 4: Output"]
        S11["Step 11: Emit multi-YAML-doc<br/>One AuditLog per target, separated by ---<br/>Each doc independently valid against #AuditLog"]

        S12["Step 12: Present cross-framework summary<br/>Conversational (not a Gemara artifact — it's a view)<br/>Highlight items needing human attention"]

        S11 --> S12
    end

    S12 --> Output

    Output["Artifacts: AuditLog (per target, multi-YAML-doc)<br/>Conversation: ~4-6 turns<br/>Tools: clickhouse-mcp, gemara-mcp, github-mcp<br/>Skills: gemara-layers, audit-classification"]
```

#### Cross-Framework Coverage Classification

```mermaid
flowchart LR
    subgraph Join["Join Logic"]
        AR["AuditResults<br/>(per target)"] -->|"criteria-reference<br/>matches source"| MD["MappingDocument<br/>.mappings[]"]
        MD -->|".targets[]<br/>.entry-id<br/>.strength<br/>.confidence"| FW["External Framework<br/>Entries"]
    end
```

| AuditResult Type | Mapping Strength | Confidence | Framework Coverage |
|:--|:--|:--|:--|
| Strength | 8-10 | High | Covered |
| Strength | 5-7 | Medium/High | Partially Covered |
| Strength | 1-4 | any | Weakly Covered |
| Finding | any | any | Not Covered (finding) |
| Gap | any | any | Not Covered (no evidence) |
| Observation | any | any | Needs Review |
| (no mapping) | — | — | Unmapped |

## Evidence Ingestion Pipeline

Evidence reaches ClickHouse through three paths. Evaluations and remediations are co-located in a single `evidence` table.

```mermaid
flowchart LR
    subgraph PathA["Path A — Gemara-native (complyctl + ProofWatch)"]
        CL["complyctl<br/>ProofWatch instrumentation"] -->|"OTLP<br/>full compliance context"| Collector
    end

    subgraph PathB["Path B — Raw policy engine (OPA, Kyverno, etc.)"]
        PE["Policy engine"] -->|"OTLP<br/>raw signals"| Collector
    end

    subgraph OTel["OTel Collector"]
        Collector["Receiver<br/>(OTLP)"]
        TB["truthbeam processor<br/>(Path B only —<br/>enriches with compliance context)"]
        EX["ClickHouse exporter"]
        Collector --> TB --> EX
    end

    subgraph Local["Path Local — Direct insert (dev only)"]
        Ingest["cmd/ingest<br/>Parse YAML → insert"]
    end

    EX --> CH
    Ingest --> CH

    subgraph CH["ClickHouse"]
        EV[("evidence<br/>MergeTree<br/>partition: toYYYYMM(collected_at)<br/>sort: target_id, policy_id,<br/>control_id, collected_at<br/>24-month TTL auto-expiry<br/><br/>Evaluations + remediations<br/>co-located (nullable cols)")]
    end
```

## Artifact Pipeline Across Agents

Agents form a pipeline. Artifacts produced by upstream agents are consumed by downstream agents.

```mermaid
flowchart LR
    subgraph Criteria["Criteria Phase"]
        GH[("GitHub<br/>Repos")] --> TM

        subgraph TM["studio-threat-modeler<br/>(Layer 2)"]
            TC[ThreatCatalog]
            CC[ControlCatalog]
        end

        TC & CC --> PC

        subgraph PC["studio-policy-composer<br/>(Layer 3)"]
            RC[RiskCatalog]
            POL[Policy]
        end
    end

    subgraph Eval["Evaluation Phase"]
        CLU["complyctl / Lula<br/>(L5/L6 evaluation)"] -->|"OTLP or YAML"| Ingest2["OTel Collector<br/>or cmd/ingest"] --> CHStore[("ClickHouse<br/>evidence table")]
    end

    subgraph Audit["Audit Phase"]
        subgraph GA["studio-gap-analyst<br/>(Layer 7)"]
            AL["AuditLog<br/>(per target)"]
        end
    end

    POL -->|Policy| GA
    MDs["MappingDocuments<br/>(user-provided)"] --> GA
    CHStore -->|evidence queries| GA

    subgraph Publish["OCI Registry"]
        OCI[("Gemara bundles<br/>typed layers<br/>optional signing")]
    end

    TM & PC & GA -->|"POST /api/publish<br/>oras-go → push → sign"| OCI
```

## Summary

| Dimension | threat-modeler | policy-composer | gap-analyst |
|:--|:--|:--|:--|
| **Gemara Layer** | L2 (Controls) | L3 (Policy) | L7 (Audit) |
| **Inputs** | GitHub repo content | ThreatCatalog + ControlCatalog | Policy + ClickHouse evidence + MappingDocuments |
| **Outputs** | ThreatCatalog, ControlCatalog | RiskCatalog, Policy | AuditLog (per target, multi-doc) |
| **Conversation** | ~1-2 turns | ~8-10 turns (guided) | ~4-6 turns (guided) |
| **MCP: gemara** | validate, prompts | validate | validate |
| **MCP: github** | get_file_contents, search_code | get_file_contents, search_code | get_file_contents, search_code |
| **MCP: clickhouse** | — | — | run_select_query, list_tables |
| **Skills** | gemara-layers, stride-analysis | gemara-layers, policy-risk-linkage, assessment-defaults | gemara-layers, audit-classification |
| **OBO token** | Yes | Yes | Yes |
