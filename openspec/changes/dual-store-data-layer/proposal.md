# Proposal: Dual-Store Data Layer

## User Story

As a program manager, I need to create and manage compliance programs with reliable state transitions so that program status, health, and evidence tracking are always consistent.

As a platform operator, I need the data layer to handle both high-volume evidence analytics and transactional program lifecycle without degrading either workload.

## Problem

Studio uses ClickHouse for all persistence. Adding program lifecycle management introduces entities that require transactional CRUD (programs, runs), and two hard PostgreSQL dependencies: LangGraph `AsyncPostgresSaver` for chat checkpointing and PGVector for BYO RAG embeddings. Forcing these into ClickHouse means building custom transactional semantics on an append-only engine and losing access to mature PostgreSQL integrations.

Additionally, two existing ClickHouse entities (`users`/`role_changes`, `notifications`) have CRUD-heavy access patterns that are unnatural in ClickHouse (mark-read UPDATE, role transitions).

## Solution

Add PostgreSQL as a second datastore with a strict bounded context: PostgreSQL owns transactional lifecycle, conversational state, and embeddings. ClickHouse owns high-volume observational data, aggregates, and analytics exports. Each entity has exactly one authoritative store. No dual-writes, no replication. The gateway orchestrates reads from both stores when cross-cutting queries require it.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| PostgreSQL Helm deployment (optional, enabled by default) | Migrating existing ClickHouse evidence/policy data to Postgres |
| Go PostgreSQL client (`pgx`) in gateway | Cross-store replication or change data capture |
| `programs`, `runs` tables in PostgreSQL | RAG embedding schema (owned by RAG service, separate spec) |
| Migrate `users`/`role_changes`/`notifications` from ClickHouse to PostgreSQL | LangGraph checkpointer tables (owned by agent runtime) |
| Cross-store query orchestration in gateway handlers | ClickHouse schema changes (existing tables unchanged) |
| Explicit partial-failure degradation semantics | |
| Bounded context documentation | |
