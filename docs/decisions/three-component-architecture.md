<!-- SPDX-License-Identifier: Apache-2.0 -->

# Three-component architecture (monorepo)

**Status:** Accepted  
**Date:** 2026-05-12

## Context

The gateway modulith coupled the SPA, REST API, OAuth, A2A/MCP plumbing, and operational services. External consumers (standalone dashboards, external agents) could not depend on the API or UI without inheriting the whole binary and deployment shape.

## Decision

Split ComplyTime into three deployable boundaries **within this repository**:

1. **Platform** — Go gateway (`cmd/`, `internal/`), PostgreSQL-backed store, optional NATS where configured, certifier pipeline and REST API.
2. **Studio** — Preact SPA under `studio/`, served by Nginx with runtime `PLATFORM_URL` (`env.js`).
3. **Agents** — kagent/BYO workloads (e.g. LangGraph assistant) with MCP sidecars.

Cross-boundary integration uses **REST** (OpenAPI) or **MCP** (studio-mcp resources/tools), not shared Go imports from Studio into Platform or vice versa.

## Consequences

| Topic | Effect |
|:--|:--|
| Deployments | Platform, Studio, and Agent groups scale and roll independently (Helm toggles such as `studio.enabled`). |
| CORS | Studio calls Platform cross-origin; gateway must allow Studio origins explicitly. |
| Agents | Agents consume platform data via **studio-mcp**, not database MCPs — stable contract, smaller blast radius. |
| Helm | Single chart can still bundle all three; chart rename to `complytime` tracked separately. |

## Related

- `openspec/changes/three-component-architecture/design.md`
- [Studio SPA extraction](studio-spa-extraction.md)
- [studio-mcp server](studio-mcp-server.md)
