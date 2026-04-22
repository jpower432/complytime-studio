## Why

The Gap Analyst specialist currently operates on MappingDocuments alone — classifying coverage from mapping relationship strengths. Real audit requires synthesizing pre-evaluated evidence (L5 EvaluationLogs, L6 EnforcementLogs) against policy criteria (L3) to produce AuditLogs (L7). The agent needs access to a queryable evidence store containing flattened L5/L6 results so it can perform evidence-backed audits rather than document-only coverage scoring.

## What Changes

- Deploy ClickHouse as the evidence store, with the official `mcp-clickhouse` MCP server exposing `run_select_query`, `list_databases`, and `list_tables` tools.
- Define two ClickHouse tables (`evaluation_logs`, `enforcement_actions`) as flattened projections of Gemara `EvaluationLog` and `EnforcementLog` artifacts.
- Add `clickhouse-mcp` as a tool for the Gap Analyst specialist (BYO Agent CRD) and update the Helm chart to deploy ClickHouse and the MCP server.
- Rewrite the Gap Analyst prompt to operate as an evidence synthesizer: load criteria from a Policy, query L5/L6 evidence from ClickHouse, classify findings, and assemble a validated AuditLog.
- Build a deterministic ingestion path (`complyctl ingest` or loader job) that validates Gemara L5/L6 YAML artifacts and writes flattened rows into ClickHouse.

## Capabilities

### New Capabilities

- `evidence-store-schema`: ClickHouse table definitions for flattened L5/L6 Gemara artifacts (`evaluation_logs`, `enforcement_actions`), partitioning strategy, and sort key design.
- `evidence-ingestion`: Deterministic loader that validates Gemara EvaluationLog/EnforcementLog YAML, flattens nested structures, and inserts rows into ClickHouse.
- `clickhouse-deployment`: Helm templates for ClickHouse StatefulSet, `mcp-clickhouse` sidecar/deployment, and `McpServer` CRD registration for the Gap Analyst.

### Modified Capabilities

- (none — the Gap Analyst prompt rewrite is an implementation detail, not a spec-level requirement change)

## Impact

- **New infrastructure**: ClickHouse StatefulSet + PVC in the Helm chart. Storage and retention policy needed.
- **New dependency**: `ClickHouse/mcp-clickhouse` MCP server image.
- **Agent changes**: Gap Analyst BYO Agent CRD gains `clickhouse-mcp` tool. Prompt rewrite changes classification logic from mapping-strength-based to evidence-result-based.
- **Ingestion tooling**: New CLI command or job for loading L5/L6 artifacts into ClickHouse. Could live in complyctl or as a standalone loader.
- **Orchestrator routing skill**: `skills/orchestrator-routing/SKILL.md` needs updated Gap Analyst documentation reflecting new required inputs (policy_id, target_id) and optional MappingDocument.
