# Audit Dashboard Pivot

**Date**: 2026-04-18
**Status**: Accepted — implemented

## Decision

Pivot ComplyTime Studio from "GRC artifact editor with agents" to "audit dashboard with agentic gap analysis."

## Context

Artifact authoring (ThreatCatalogs, ControlCatalogs, RiskCatalogs, Policies) is converging on local developer tooling — Cursor, Claude Code, and the Gemara MCP server. Engineers already have a software factory for producing GRC artifacts. Building a competing editor inside Studio duplicates that ecosystem with inferior ergonomics.

The gap: no platform synthesizes evidence, maps artifacts to compliance frameworks, tracks audit posture over time, and surfaces gaps with agentic help.

## What Was Cut

| Component | Reason |
|:--|:--|
| `studio-threat-modeler` agent | Engineers use Cursor + gemara-mcp |
| `studio-policy-composer` agent | Governance uses Cursor + gemara-mcp |
| Workbench artifact editor | Replaced by dashboard with read-only viewers |
| Artifact chaining UX | Happens in engineer's local tool |
| OCI publish from editor | Publishing in CI/CD |
| `skills/gemara-authoring` | Belongs in gemara-mcp ecosystem |
| `skills/risk-reasoning` | Belongs in gemara-mcp ecosystem |
| Jobs view / agent picker | Replaced by persistent chat assistant |

## What Was Added

| Component | Purpose |
|:--|:--|
| Policy store (ClickHouse) | Import and store policies from OCI registries |
| Evidence REST API + file upload | Multi-channel evidence ingestion |
| AuditLog history (ClickHouse) | Historical audit tracking and trend analysis |
| Crosswalk mappings | Link internal criteria to external frameworks |
| Dashboard views | Posture, Policies, Evidence, Audit History |
| Chat assistant overlay | Persistent gap analyst access (Gemini-in-Gmail pattern) |
| BYO ADK gap analyst | Deterministic gates + structured artifact emission |

## Consequences

- Studio is no longer an authoring tool. It consumes artifacts produced elsewhere.
- ClickHouse becomes a required dependency, not optional.
- Single agent simplifies operations but limits Studio's scope.
- Engineers need to adopt local tooling (Cursor + gemara-mcp) for artifact creation.
