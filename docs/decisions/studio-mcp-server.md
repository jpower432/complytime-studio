<!-- SPDX-License-Identifier: Apache-2.0 -->

# studio-mcp server

**Status:** Accepted  
**Date:** 2026-05-12

## Context

Agents previously reached PostgreSQL through `postgres-mcp` with raw SQL. That tied prompts and tools to physical schema, complicated upgrades, and enlarged the attack surface.

## Decision

Introduce **`studio-mcp`** (`cmd/studio-mcp`): a Go MCP server that uses `internal/store` and exposes **typed `studio://` resources** (policies, evidence, posture, audit logs, mappings, catalogs, threats, risks) plus tools **`ingest_evidence`** and **`save_draft_audit_log`**.

Transports: **stdio** (sidecar) and **HTTP** (standalone).

## Consequences

| Topic | Effect |
|:--|:--|
| Contract | Resource URIs and JSON shapes become the agent-facing API for reads/writes exposed here. |
| Schema drift | Store implementation may evolve behind stable resource semantics. |
| Ops | New binary and image (`Dockerfile.studio-mcp`); compose/Helm wire `--postgres-url` / `POSTGRES_URL` into the server. |

## Related

- [Agent MCP surface](agent-mcp-surface.md)
- [studio-mcp reference](../api/studio-mcp.md)
