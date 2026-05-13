<!-- SPDX-License-Identifier: Apache-2.0 -->

# Studio SPA extraction from gateway

**Status:** Accepted  
**Date:** 2026-05-12

## Context

The workbench SPA lived inside the Go binary (`go:embed`) and assumed same-origin `/api/*`. That blocked standalone Studio hosting and forced every UI deploy to ride gateway releases.

## Decision

Extract the SPA into `studio/` as a **standalone static site** delivered by an **Nginx** container.

- **Runtime configuration:** `env.js` generated from `env.js.template` via `envsubst` at container start (`window.__STUDIO_CONFIG__.platformUrl`).
- **Local dev:** `VITE_PLATFORM_URL` / dev server proxy patterns remain available for `npm run dev`.
- **Platform:** Gateway does not serve SPA assets; non-API routes return JSON 404 for the root listener.

## Consequences

| Topic | Effect |
|:--|:--|
| CORS | Mandatory for browser traffic from Studio origin to Platform API and A2A routes. |
| Caching | `env.js` served with no-cache headers so URL changes propagate without rebuilds. |
| Images | Separate container build (`studio/Dockerfile`) from gateway image. |

## Related

- [Three-component architecture](three-component-architecture.md)
- `studio/nginx.conf`, `studio/docker-entrypoint.sh`
