## Context

ComplyTime Studio's Gap Analyst specialist currently performs L7 audit by scoring MappingDocument coverage relationships. The architectural redesign (declarative-orchestrator) established a pattern of BYO specialists with MCP tool access. This change extends that pattern by adding ClickHouse as a queryable evidence store for pre-evaluated L5/L6 data, using the official `ClickHouse/mcp-clickhouse` MCP server.

The Gap Analyst's role shifts from "mapping coverage scorer" to "evidence synthesizer" — it receives policy criteria and queries pre-computed evaluation and enforcement results to produce AuditLogs grounded in real measurement data.

## Goals / Non-Goals

**Goals:**

- Give the Gap Analyst read-only query access to pre-evaluated L5/L6 evidence via `mcp-clickhouse`
- Define a ClickHouse schema that is a direct flattened projection of Gemara `EvaluationLog` and `EnforcementLog` structures
- Build a deterministic ingestion path that validates Gemara YAML and loads flattened rows
- Deploy ClickHouse and `mcp-clickhouse` via the existing Helm chart

**Non-Goals:**

- The Gap Analyst does not perform evaluation (L5) or enforcement (L6) — it only consumes their outputs
- No real-time streaming ingestion; batch load of completed artifacts is sufficient
- No custom MCP server; use the official `ClickHouse/mcp-clickhouse` as-is
- No ClickHouse Cloud integration; self-hosted instance in the cluster for now
- No UI/dashboard for ClickHouse data; the agent is the sole consumer

## Decisions

### D1: Use official `mcp-clickhouse` rather than building a custom evidence MCP server

The official server exposes `run_select_query`, `list_databases`, and `list_tables`. The Gap Analyst needs read-only SQL access against a known schema — nothing more. A custom server would add maintenance burden for zero additional capability.

**Alternative considered:** Custom MCP server with high-level tools like `query_evidence(control_id, time_range)`. Rejected because it couples the server to the Gemara schema, requires maintenance as the schema evolves, and limits the agent's flexibility to construct ad-hoc queries when investigating edge cases.

### D2: One row per assessment (not per control)

A `ControlEvaluation` contains multiple `AssessmentLog` entries, one per assessment requirement. Storing at the assessment level preserves requirement-granularity, which is exactly what the AuditLog needs — each `AuditResult` maps to a specific criteria entry.

**Alternative considered:** One row per control with nested JSON for assessments. Rejected because ClickHouse queries against nested JSON are less ergonomic for the agent and lose the columnar compression benefits on assessment-level fields.

### D3: Denormalized tables, no joins

Both tables carry `target_id`, `policy_id`, `control_id` on every row. ClickHouse's columnar compression handles repeated strings efficiently. The agent issues a single `run_select_query` per table and gets everything needed — no multi-step join queries for the LLM to construct.

**Alternative considered:** Normalized tables with a shared `audit_runs` dimension table. Rejected because it forces the agent to construct JOINs, increasing query complexity and error probability.

### D4: `policy_id` and `target_id` as primary query dimensions

The natural audit question is "audit this policy against this target over this time window." The ClickHouse sort key `(target_id, policy_id, control_id, collected_at)` makes this the efficient access pattern.

### D5: Ingestion as a standalone Go CLI command

The loader validates a Gemara YAML artifact, flattens nested structures, and writes rows to ClickHouse via the native protocol. This is a deterministic pipeline step — no LLM involvement. Implemented as a `complyctl ingest` subcommand or a standalone binary.

**Alternative considered:** Kubernetes Job triggered by OCI registry webhook. Viable for production but adds complexity for initial development. Start with CLI, add automation later.

### D6: Attach `clickhouse-mcp` to the Gap Analyst, not the orchestrator

The orchestrator delegates to specialists and assembles bundles — it has no reason to query evidence. Only the Gap Analyst needs ClickHouse access. This follows the principle of minimal tool surface per agent.

### D7: Audit time window derived from Policy adherence frequency

The Gap Analyst derives the query time range from `Policy.adherence.assessment-plans[].frequency` rather than requiring the user to specify a date range. If a plan says "monthly," the agent queries the last 30 days. The AuditLog's `metadata.date` records when the audit was produced; the evidence window is implicit in `evidence[].collected` timestamps.

## Risks / Trade-offs

| Risk | Mitigation |
|:-----|:-----------|
| Agent constructs incorrect SQL | Schema is simple (2 tables, no joins). Sort key matches natural query pattern. Agent prompt includes example queries. |
| ClickHouse adds operational complexity | Single-node deployment with persistent volume. No replication or sharding needed at expected volume (< 100K rows/year per target). |
| Ingestion loader must stay in sync with Gemara schema | Loader validates input against Gemara CUE schema before flattening. Schema drift caught at ingestion time, not query time. |
| Large result sets exceed agent context | Add `LIMIT` and time-range constraints in the prompt. Typical audit scope is 50-200 controls × 1-5 requirements = 250-1000 rows — well within context limits. |
| ClickHouse PVC storage growth | Partition by month (`toYYYYMM`). Add TTL policy for automatic expiry (default: 24 months). |

## Open Questions

- **Ingestion trigger:** Manual CLI invocation for now. Should a future iteration watch the OCI registry for new L5/L6 artifacts and auto-ingest?
- **Multi-target audits:** Current design scopes one audit to one target. Cross-target comparison (e.g., "which clusters have the most gaps?") is possible with the schema but not designed for in the prompt.
- **ClickHouse authentication:** The `mcp-clickhouse` server needs credentials. Secret management approach TBD (Kubernetes Secret referenced in the MCP server deployment).
