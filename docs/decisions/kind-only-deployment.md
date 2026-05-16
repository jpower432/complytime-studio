# ADR 0035: Kind + Helm as Sole Deployment Path

**Status:** Accepted
**Date:** 2026-05-16

## Context

The `studio-deploy` repo maintained two parallel deployment paths:

1. **Docker Compose** — full-stack local development
2. **Kind + Helm** — Kubernetes-native deployment

The Compose stack duplicated the Helm chart's service definitions, networking, and auth configuration. Every architectural change (unified ingest, repo renames, auth model) required updating both. The Compose stack lacked production-equivalent auth (used a `DEV_EMAIL` header injection hack instead of OAuth2 Proxy) and had no network isolation.

## Decision

Drop the full-stack Docker Compose. Kind + Helm is the single deployment path.

Retain a minimal `docker-compose.yaml` with only PostgreSQL and NATS for developers running gateway or workbench binaries locally (`go run`, `python`).

## Consequences

- **One source of truth:** Helm chart is the only service definition to maintain.
- **Auth parity:** Local dev uses the same OAuth2 Proxy sidecar as production. No fake identity headers.
- **Network isolation:** Gateway binds to `127.0.0.1` inside the Pod when auth is enabled. Same pattern as production.
- **Slower rebuild cycle:** Image builds + Kind load takes minutes vs seconds with Compose. Acceptable for integration testing; local binary debugging uses `make infra-up` against raw postgres/nats.
- **Higher prerequisites:** Contributors need `kind`, `kubectl`, `helm` in addition to a container runtime.

## Alternatives Considered

- **Add OAuth2 Proxy to Compose with network isolation:** Feasible but perpetuates dual-config maintenance. Compose networks approximate but do not replicate K8s NetworkPolicy or sidecar localhost binding.
- **Keep Compose for quick iteration, Helm for integration:** Tried this. The configs diverged within days.
