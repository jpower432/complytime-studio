<!-- SPDX-License-Identifier: Apache-2.0 -->

# Agent MCP surface — studio-mcp vs postgres-mcp

**Status:** Accepted  
**Date:** 2026-05-12

## Context

Direct SQL from agents duplicates policy already enforced in the gateway store layer and leaks ClickHouse/PostgreSQL schema details into prompts.

## Decision

**Agents read and write platform-aligned data through `studio-mcp` only** (resources + `ingest_evidence` / `save_draft_audit_log`). **`postgres-mcp` is not used** for ComplyTime Studio platform data access.

Gemara validation (`studio-gemara-mcp` / `gemara-mcp`) and OCI (`oras-mcp`) remain separate concerns.

## Consequences

| Topic | Effect |
|:--|:--|
| Security | No arbitrary SQL from agent tooling against production stores. |
| Prompts | References migrate from SQL examples to `studio://…` URIs and tool calls. |
| Gaps | Missing resource coverage is fixed by extending studio-mcp, not ad hoc queries. |

## Related

- [studio-mcp server](studio-mcp-server.md)
- `agents/assistant/agent.yaml` (MCP block)
