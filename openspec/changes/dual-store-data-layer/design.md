# Design: Dual-Store Data Layer

## Bounded Context Map

### PostgreSQL (transactional lifecycle)

| Table | Access Pattern | Notes |
|:--|:--|:--|
| `programs` | CRUD, optimistic locking (`version`), soft delete (`deleted_at`) | 5–50 rows. FK target for runs. |
| `runs` | Insert on command execution, update status on completion | FK to programs. Hundreds of rows. |
| `users` | CRUD, role transitions | Migrated from ClickHouse. Tens of rows. |
| `role_changes` | Append + read | Migrated from ClickHouse. Audit trail for RBAC. |
| `notifications` | Insert on event, mark-read UPDATE, delete after TTL | Migrated from ClickHouse. Hundreds of rows. |
| `rag_embeddings` | PGVector similarity search | Managed by BYO RAG service — Studio does not own schema. |

### ClickHouse (analytics + evidence)

| Table | Access Pattern | Notes |
|:--|:--|:--|
| `evidence` | Append millions, aggregate, TTL, export | Unchanged. |
| `policies` | Read-heavy, import from OCI | Unchanged. |
| `mapping_documents` / `mapping_entries` | Read-heavy, import | Unchanged. |
| `catalogs` / `controls` / `threats` / `risks` | Read-heavy reference data | Unchanged. |
| `audit_logs` / `draft_audit_logs` | Append-only, time-range scans, export | Unchanged. |
| `certifications` | Event-driven append | Unchanged. |
| `evidence_assessments` | Append | Unchanged. |
| `posture` (materialized view) | Auto-refresh on evidence insert | Unchanged. |

### Cross-Store Query Orchestration

No direct joins. Gateway orchestrates with batched key resolution.

| Query | Pattern |
|:--|:--|
| Evidence for program X | Postgres: `SELECT policy_ids FROM programs WHERE id = ?` → ClickHouse: `WHERE policy_id IN (...)` |
| Posture for program X | Same: program → policy_ids → ClickHouse posture |
| Program health | Postgres: program row + ClickHouse: evidence coverage aggregation |

**N+1 mitigation:** Batch resolve policy_ids in one Postgres query, then single ClickHouse `IN(...)` query. No per-policy fan-out.

### Partial Failure Semantics

| Scenario | Behavior |
|:--|:--|
| ClickHouse up, PostgreSQL down | Evidence dashboard works. Program views return 503 with subsystem identifier. |
| PostgreSQL up, ClickHouse down | Program CRUD works. Evidence tabs show degraded state with clear messaging. |
| Both down | Full 503. |

API responses include `X-Studio-Degraded: clickhouse` or `X-Studio-Degraded: postgres` header when a subsystem is unavailable.

## PostgreSQL Schema

### `programs`

```sql
CREATE TABLE programs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    framework   TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'intake',
    health      TEXT,
    owner       TEXT,
    description TEXT,
    metadata    JSONB NOT NULL DEFAULT '{}',
    policy_ids  TEXT[] NOT NULL DEFAULT '{}',
    version     INTEGER NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
CREATE INDEX idx_programs_status ON programs(status) WHERE deleted_at IS NULL;
```

`policy_ids` links a program to ClickHouse policy rows. Array of policy ID strings — no FK enforcement across stores. Postgres is source of intent; ClickHouse is denormalized for analytics.

### `runs`

```sql
CREATE TABLE runs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id  UUID NOT NULL REFERENCES programs(id),
    command     TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    input_args  JSONB NOT NULL DEFAULT '{}',
    output      TEXT,
    status      TEXT NOT NULL DEFAULT 'running',
    tokens_in   INTEGER,
    tokens_out  INTEGER,
    duration_ms INTEGER,
    quality_gate_passed BOOLEAN,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_runs_program ON runs(program_id, created_at DESC);
```

### `users` (migrated from ClickHouse)

```sql
CREATE TABLE users (
    email       TEXT PRIMARY KEY,
    name        TEXT NOT NULL DEFAULT '',
    avatar_url  TEXT NOT NULL DEFAULT '',
    role        TEXT NOT NULL DEFAULT 'reviewer',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### `role_changes` (migrated from ClickHouse)

```sql
CREATE TABLE role_changes (
    id          BIGSERIAL PRIMARY KEY,
    changed_by  TEXT NOT NULL,
    target_email TEXT NOT NULL,
    old_role    TEXT NOT NULL,
    new_role    TEXT NOT NULL,
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_role_changes_target ON role_changes(target_email);
```

### `notifications` (migrated from ClickHouse)

```sql
CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id TEXT NOT NULL,
    type            TEXT NOT NULL,
    policy_id       TEXT NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    read            BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_unread ON notifications(read, created_at DESC) WHERE NOT read;
```

## Gateway Architecture

### New package: `internal/postgres`

```go
type Config struct {
    URL string // DATABASE_URL, e.g. postgres://user:pass@host:5432/studio
}

type Client struct {
    pool *pgxpool.Pool
}

func New(ctx context.Context, cfg Config) (*Client, error)
func (c *Client) EnsureSchema(ctx context.Context) error
func (c *Client) Close()
```

`EnsureSchema` runs migrations embedded via `embed.FS`.

### Store interface expansion

`internal/store/store.go` `Stores` struct adds PostgreSQL-backed stores:

```go
type Stores struct {
    // ClickHouse-backed (existing)
    Policies       PolicyStore
    Evidence       EvidenceStore
    AuditLogs      AuditLogStore
    // ... all existing ...

    // PostgreSQL-backed (new)
    Programs       ProgramStore
    Runs           RunStore
    Users          UserStore       // migrated from ClickHouse
    Notifications  NotificationStore // migrated from ClickHouse
}
```

### `cmd/gateway/main.go` changes

```
IF POSTGRES_URL set:
    pgClient = postgres.New(ctx, cfg)
    pgClient.EnsureSchema(ctx)
    stores.Programs = pgClient
    stores.Runs = pgClient
    stores.Users = pgClient          // replaces ClickHouse-backed store
    stores.Notifications = pgClient  // replaces ClickHouse-backed store
    authHandler.SetUserStore(pgClient)
ELSE:
    log.Warn("POSTGRES_URL not set — program lifecycle disabled")
    // users/notifications fall back to ClickHouse (existing behavior)
```

PostgreSQL is **required for program lifecycle features**. Without it, Studio works as today — evidence dashboard + assistant. ClickHouse-only mode remains supported for analytics-only deployments.

## Helm Values

```yaml
postgres:
  enabled: true
  image:
    repository: postgres
    tag: "17"
  auth:
    database: studio
    user: studio
    password: complytime-dev  # dev only — use existingSecret for production
    existingSecret: ""
  storage:
    size: 1Gi
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "1"
```

## Migration Path

### Phase 1: Add PostgreSQL, new tables only

- Deploy PostgreSQL alongside ClickHouse
- `programs`, `runs` created in PostgreSQL
- `users`/`notifications` remain in ClickHouse (no migration yet)
- Gateway talks to both

### Phase 2: Migrate users/notifications

- Idempotent migration job: ClickHouse users/notifications → PostgreSQL (stable UUIDs)
- Switch `authHandler.SetUserStore` to PostgreSQL-backed store
- Switch notification handlers to PostgreSQL-backed store
- Drop ClickHouse `users`/`role_changes`/`notifications` tables

### Phase 3: External runtimes

- BYO RAG service creates `rag_embeddings` in the same PostgreSQL instance (PGVector extension)
- LangGraph agent creates `chat_checkpoints` tables via `AsyncPostgresSaver`
- Studio does not own these schemas — they are managed by their respective runtimes

## Environment Variables

| Variable | Default | Notes |
|:--|:--|:--|
| `POSTGRES_URL` | `""` | Full connection string. Empty = PostgreSQL disabled. |

## Rejected Alternatives

**ClickHouse only**: No LangGraph checkpointer, no PGVector, CRUD semantics require `ReplacingMergeTree` + `FINAL` on every read, no real transactions for program state transitions.

**PostgreSQL only**: Requires migrating millions of evidence rows, rebuilding materialized views, losing TTL, losing columnar compression, rebuilding export pipeline. Months of work for worse analytics performance.
