# Proposal: Helm Chart — Platform Expansion

## User Story

As a platform operator, I need to deploy Studio's expanded stack (PostgreSQL, LangGraph agents, OIDC auth, BYO RAG) through a single `helm upgrade` with sensible defaults and feature flags so I can progressively enable capabilities without breaking the existing deployment.

## Problem

The preceding specs (OIDC auth, dual-store, sub-agent registry, LangGraph runtime, workbench views, governance, BYO RAG) each introduce new infrastructure components. This spec consolidates the Helm chart changes needed to deploy the expanded platform.

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
