# Design: Helm Chart — Platform Expansion

## New Templates

### `templates/postgres.yaml`

StatefulSet + Service + Secret (when `postgres.enabled`).

```yaml
# Gated: {{ if .Values.postgres.enabled }}
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: studio-postgres
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: postgres
          image: "{{ .Values.postgres.image.repository }}:{{ .Values.postgres.image.tag }}"
          env:
            - name: POSTGRES_DB
              value: "{{ .Values.postgres.auth.database }}"
            - name: POSTGRES_USER
              value: "{{ .Values.postgres.auth.user }}"
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef: ...
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
          startupProbe:
            exec:
              command: ["pg_isready", "-U", "{{ .Values.postgres.auth.user }}"]
            failureThreshold: 30
            periodSeconds: 2
          readinessProbe:
            exec:
              command: ["pg_isready", "-U", "{{ .Values.postgres.auth.user }}"]
            periodSeconds: 10
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: [ReadWriteOnce]
        resources:
          requests:
            storage: "{{ .Values.postgres.storage.size }}"
```

Production: use managed PostgreSQL (RDS, Cloud SQL, CloudNativePG operator). In-chart STS is for dev/staging only.

### `templates/langgraph-agents.yaml`

Single ranged template for all enabled personas:

```yaml
{{- range $key, $agent := .Values.langgraphAgents }}
{{- if $agent.enabled }}
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: studio-{{ $key }}
spec:
  description: {{ $agent.description }}
  type: BYO
  byo:
    deployment:
      image: "{{ $agent.image.repository }}:{{ $agent.image.tag }}"
      env:
        - name: AGENT_TYPE
          value: "{{ $agent.agentType }}"
        - name: LLM_PROVIDER
          value: "{{ $agent.model.provider | default $.Values.model.provider }}"
        - name: LLM_MODEL
          value: "{{ $agent.model.name | default $.Values.model.name }}"
        - name: GEMARA_MCP_URL
          value: "http://studio-gemara-mcp:8080"
        - name: CLICKHOUSE_MCP_URL
          value: "http://studio-clickhouse-mcp:8080"
        {{- if $.Values.postgres.enabled }}
        - name: POSTGRES_URL
          valueFrom:
            secretKeyRef:
              name: studio-postgres-credentials
              key: url
        {{- end }}
        {{- if $.Values.rag.enabled }}
        - name: KNOWLEDGE_BASE_MCP_URL
          value: "http://studio-knowledge-base-mcp:8080"
        {{- end }}
---
{{- end }}
{{- end }}
```

### `templates/command-specs-configmap.yaml`

Mounts command spec markdown files for agent spec loading and gateway `/api/commands` endpoint.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: studio-command-specs
data:
  {{- range $path, $content := .Files.Glob "commands/*.md" }}
  {{ $path | base }}: |
    {{ $content | toString | indent 4 }}
  {{- end }}
```

Verify total size stays under etcd 1 MiB object limit. Split ConfigMaps if needed.

### `templates/knowledge-base-mcp.yaml` (optional)

```yaml
# Gated: {{ if .Values.rag.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: studio-knowledge-base-mcp
spec:
  template:
    spec:
      containers:
        - name: knowledge-base-mcp
          image: "{{ .Values.rag.image.repository }}:{{ .Values.rag.image.tag }}"
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: studio-postgres-credentials
                  key: url
          ports:
            - containerPort: 8080
```

## Modified Templates

### `templates/gateway.yaml`

New env vars:

| Env var | Source | Condition |
|:--|:--|:--|
| `OIDC_ISSUER_URL` | `auth.oidc.issuerUrl` | Always (empty = auth disabled) |
| `OIDC_CLIENT_ID` | `auth.oidc.clientId` | When `auth.oidc.issuerUrl` set |
| `OIDC_CLIENT_SECRET` | Secret ref | When `auth.oidc.issuerUrl` set |
| `OIDC_CALLBACK_URL` | `auth.oidc.callbackUrl` | When `auth.oidc.issuerUrl` set |
| `OIDC_SCOPES` | `auth.oidc.scopes` | When `auth.oidc.issuerUrl` set |
| `OIDC_ROLES_CLAIM` | `auth.oidc.rolesClaim` | When `auth.oidc.issuerUrl` set |
| `GOOGLE_CLIENT_ID` | `auth.google.clientId` | Legacy compat |
| `GOOGLE_CLIENT_SECRET` | Secret ref | Legacy compat |
| `POSTGRES_URL` | Secret ref | When `postgres.enabled` |

initContainer: wait for PostgreSQL readiness when `postgres.enabled`:
```yaml
initContainers:
  - name: wait-postgres
    image: postgres:17
    command: ["sh", "-c", "until pg_isready -h studio-postgres -U studio; do sleep 2; done"]
```

Volume mount: `studio-command-specs` ConfigMap at `/etc/studio/commands/`.

### `templates/platform-prompts-configmap.yaml`

Add sub-agent directory block to assistant system prompt (generated from `agentDirectory` entries where `delegatable: true`).

## Values Expansion

```yaml
# --- Auth (expanded from current google-only) ---
auth:
  oidc:
    issuerUrl: ""
    clientId: ""
    clientSecret: ""
    callbackUrl: "http://localhost:8080/auth/callback"
    scopes: "openid email profile"
    rolesClaim: ""
    bootstrapEmails: []
    discoveryRefresh: "24h"
  google:
    clientId: ""          # deprecated — use auth.oidc with Google issuer
    secretName: studio-oauth-credentials
    secretKey: client-secret
    callbackURL: "http://localhost:8080/auth/callback"
  cookieSecretName: ""
  admins: []
  apiToken: "dev-seed-token"

# --- PostgreSQL (new) ---
postgres:
  enabled: true
  image:
    repository: postgres
    tag: "17"
  auth:
    database: studio
    user: studio
    password: complytime-dev  # dev only
    existingSecret: ""        # production: use this instead
  storage:
    size: 1Gi
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "1"

# --- LangGraph agents (new, separate namespace from agents.assistant) ---
langgraphAgents:
  program-agent:
    enabled: false
    agentType: program
    image:
      repository: studio-langgraph-agent
      tag: latest
    description: >-
      Program lifecycle management — intake, monitoring, pipeline runs,
      and state transitions
    model: {}
  evidence-agent:
    enabled: false
    agentType: evidence
    image:
      repository: studio-langgraph-agent
      tag: latest
    description: >-
      Evidence staleness monitoring, gap detection, and validation
    model: {}
  coordinator:
    enabled: false
    agentType: coordinator
    image:
      repository: studio-langgraph-agent
      tag: latest
    description: >-
      Portfolio aggregation, cross-program signals, work routing
    model: {}

# --- BYO RAG (new, optional) ---
rag:
  enabled: false
  image:
    repository: ""    # operator provides their RAG MCP server image
    tag: latest
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "2Gi"
      cpu: "2"

# --- Agent directory (expanded) ---
agentDirectory:
  - id: studio-assistant
    name: Studio Assistant
    description: >-
      Audit preparation, evidence synthesis, cross-framework coverage
      analysis, and compliance guidance
    url: "http://studio-assistant:8080"
    role: assistant
    framework: adk
    delegatable: false
    skills:
      - id: compliance-assistant
        name: Studio Assistant
        description: >-
          Audit preparation, evidence synthesis, cross-framework coverage
          analysis, policy guidance, and AuditLog generation.
        tags: [assistant, audit, compliance]
  # Additional entries auto-generated from langgraphAgents.*.enabled
```

## Docker Compose Additions

```yaml
services:
  postgres:
    image: postgres:17
    environment:
      POSTGRES_DB: studio
      POSTGRES_USER: studio
      POSTGRES_PASSWORD: complytime-dev
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  gateway:
    environment:
      POSTGRES_URL: "postgres://studio:complytime-dev@postgres:5432/studio"
      # OIDC env vars when testing auth
```

Docker Compose is a local dev subset, not a mirror of Helm. Documented as such in README.

## Deployment Profiles

| Profile | Components | Use case |
|:--|:--|:--|
| **Minimal** | Gateway + ClickHouse + Assistant | Current behavior, analytics + audit only |
| **Standard** | + PostgreSQL + program-agent | Program lifecycle enabled |
| **Full** | + all LangGraph agents + BYO RAG | Complete compliance fleet |

Profiles are documented presets (`values-minimal.yaml`, `values-standard.yaml`, `values-full.yaml`), not formal Helm concepts.
