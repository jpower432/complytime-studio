# Proposal: Agent Governance

## User Story

As a compliance officer, I need behavioral consistency across all AI agents and validated output quality so that agent-produced artifacts meet structural and format standards before they reach audit workflows.

## Problem

Studio's assistant has minimal governance: a `before_tool` SQL injection guard and `admin`/`reviewer` RBAC. Adding more agents without shared behavioral rules or output validation risks inconsistent, unverifiable outputs.

## Solution

Two layers:

1. **Constitution as skill** — a shared markdown skill loaded by all agents defining invariant behavioral rules (never fabricate evidence, cite sources, refuse out-of-scope requests). Low cost, high consistency.
2. **Quality gates as MCP tool** — agents call a `validate_command_output` tool after producing command outputs. The tool checks structural completeness (expected sections present) and format correctness (valid JSON blocks, heading style). Results returned to the agent for self-correction or inclusion in response metadata.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| `skills/constitution/SKILL.md` (behavioral mandates) | Trust model (deferred — needs threat model justification) |
| `validate_command_output` MCP tool on gemara-mcp | Gateway SSE stream interception for validation |
| Workbench rendering of quality gate results | Hash-chained audit provenance (deferred — needs threat model justification) |
| | Automated trust promotion/demotion |
