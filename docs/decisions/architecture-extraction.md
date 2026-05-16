# ADR 0032: Architecture Extraction — Core + Studio

**Status:** Accepted
**Date:** 2026-05-15

## Context

Two user groups need different functionality:
- **Group 1** needs a standalone evidence API — headless data platform, no UI, no programs.
- **Group 2** needs a full audit workbench — programs, coverage analysis, agent-driven audit production, UI.

The monolith conflates the data platform with the workbench. Serving Group 1 requires deploying program tables, agent dependencies, and UI routes they do not use.

## Decision

Extract the system into two independent products that compose together.

| Repo | Role | Language |
|:---|:---|:---|
| `complytime-core` | Evidence data platform. Self-contained, self-deploying. Includes NATS and `complytime-mcp`. | Go |
| `complytime-studio` | Audit workbench + agent + `studio-mcp`. | Python |
| `studio-ui` | Preact SPA. Primary client of the workbench. | TypeScript |
| `studio-deploy` | Full stack composition (core + studio + UI). | YAML |

### Data Ownership

Two databases in one Postgres instance. No cross-database queries. All cross-service data flows through APIs.

**core DB:** policies, evidence, catalogs, controls, assessment_requirements, mapping_documents, mapping_entries, threats, risks, risk_threats, control_threats, certifications, evidence_assessments, audit_logs (finalized), users, role_changes, guidance_entries.

**workbench DB:** programs, jobs, program_members, program_findings, draft_audit_logs, LangGraph checkpoints. Future: coverage_snapshots, recommendation_state.

### Serving Contracts

**complytime-core:** REST `/api/*` (full CRUD), MCP `complytime-mcp` (read-only evidence surface, URI prefix `complytime://`), SQL `core_reader` role (SELECT-only).

**complytime-studio:** REST `/workbench/*` (programs, drafts, coverage), MCP `studio-mcp` (programs, draft writes, URI prefix `studio://`).

### Hard Rule

No service writes to another service's database.

## Consequences

- Group 1 can deploy `complytime-core` alone without workbench/agent/UI overhead.
- Group 2 composes both via `studio-deploy`.
- Program migration requires rewriting Go handlers in Python (workbench owns programs now).
- MCP surface split: `complytime-mcp` (core, read-only) and `studio-mcp` (workbench, read + draft write).
- Agent communicates via MCP only — no in-process function calls to workbench.
- Full design spec: `studio-deploy/docs/superpowers/specs/2026-05-15-architecture-extraction-design.md`.
