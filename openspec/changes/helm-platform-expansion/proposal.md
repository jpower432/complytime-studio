# Proposal: Helm Chart — Platform Expansion

> **Status: Partially Implemented (2026-05)**
>
> This spec was written before implementation. The auth approach pivoted to
> OAuth2 Proxy sidecar (see `openspec/changes/generic-oidc-auth/`). Items
> below marked "Shipped" are live on `feat/infra-oidc-postgres-byo`; items
> marked "Deferred" are tracked for future work.

## User Story

As a platform operator, I need to deploy Studio's expanded stack (PostgreSQL, LangGraph agents, OIDC auth, BYO RAG) through a single `helm upgrade` with sensible defaults and feature flags so I can progressively enable capabilities without breaking the existing deployment.

## Problem

The preceding specs (OIDC auth, dual-store, sub-agent registry, LangGraph runtime, workbench views, governance, BYO RAG) each introduce new infrastructure components. This spec consolidates the Helm chart changes needed to deploy the expanded platform.

## Shipped vs Deferred

| Item | Status | Notes |
|:--|:--|:--|
| PostgreSQL StatefulSet + Service + Secret | Shipped | `templates/postgres.yaml` |
| Gateway OAuth2 Proxy sidecar + auth values | Shipped | Via `generic-oidc-auth` — replaces gateway OIDC env vars |
| NetworkPolicy templates | Shipped | `templates/network-policies.yaml` |
| `docker-compose.yaml` additions | Shipped | Postgres, NATS, ORAS MCP |
| `values.yaml` expansion (postgres, auth, nats) | Shipped | |
| LangGraph agent BYO CRD templates | Deferred | Blocked on LangGraph runtime design |
| Command specs ConfigMap | Deferred | |
| BYO RAG MCP server template | Deferred | |
| Preset values files (minimal/standard/full) | Deferred | |

## Scope

| In Scope | Out of Scope |
|:--|:--|
| PostgreSQL StatefulSet + Service + Secret | PGVector extension install (RAG service manages) |
| LangGraph agent BYO CRD templates (ranged) | kagent controller deployment (operator-managed) |
| Command specs ConfigMap | Proprietary agent integrations (operator-managed) |
| OIDC auth values restructure | External secret management (operator responsibility) |
| BYO RAG MCP server template (optional) | Production hardening (TLS, resource tuning) |
| `docker-compose.yaml` additions | |
| `values.yaml` expansion | |
