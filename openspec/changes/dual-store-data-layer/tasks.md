# Tasks: Dual-Store Data Layer

## Phase 1: PostgreSQL + new tables
- [ ] Add `internal/postgres/client.go` with `Config`, `Client`, `New()`, `EnsureSchema()`, `Close()`
- [ ] Add embedded SQL migrations for `programs`, `runs` tables
- [ ] Implement `ProgramStore` interface and PostgreSQL-backed implementation
- [ ] Implement `RunStore` interface and PostgreSQL-backed implementation
- [ ] Add programs CRUD handlers to `internal/store/handlers.go`
- [ ] Add runs handlers (list by program, get by ID)
- [ ] Wire PostgreSQL client in `cmd/gateway/main.go` (optional — gated on `POSTGRES_URL`)
- [ ] Add PostgreSQL to Helm chart (`templates/postgres.yaml`, `values.yaml`)
- [ ] Add `POSTGRES_URL` to gateway deployment env vars
- [ ] Add PostgreSQL to `docker-compose.yaml`
- [ ] Define partial-failure degradation semantics (X-Studio-Degraded header)
- [ ] Tests: programs CRUD, optimistic locking, soft delete, runs lifecycle, cross-store batched query

## Phase 2: Migrate users/notifications
- [ ] Implement `UserStore` interface on PostgreSQL client
- [ ] Implement `NotificationStore` interface on PostgreSQL client
- [ ] Add embedded SQL migrations for `users`, `role_changes`, `notifications` tables
- [ ] Add idempotent migration job: ClickHouse → PostgreSQL
- [ ] Switch `authHandler.SetUserStore` to PostgreSQL-backed store when available
- [ ] Switch notification handlers to PostgreSQL-backed store when available
- [ ] Mark ClickHouse `users`/`role_changes`/`notifications` tables as deprecated
- [ ] Tests: user CRUD, role transitions, notification mark-read

## Phase 3: External runtimes
- [ ] Document PGVector extension requirement for BYO RAG services
- [ ] Document LangGraph `AsyncPostgresSaver` connection to same PostgreSQL instance
- [ ] Verify shared PostgreSQL instance handles concurrent access from gateway + agent runtimes
