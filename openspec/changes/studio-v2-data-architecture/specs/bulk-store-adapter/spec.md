## ADDED Requirements

### Requirement: Bulk store adapter abstracts the storage backend
The system SHALL define a `BulkStore` interface with methods: `Append(ctx, rows []EvidenceRow) error`, `Query(ctx, query BulkQuery) ([]EvidenceRow, error)`, `Close() error`. Two implementations SHALL exist: `OpenSearchBulkStore` (default) and `ClickHouseBulkStore` (opt-in).

#### Scenario: OpenSearch default
- **WHEN** the Helm value `bulkStore.backend` is unset or `"opensearch"`
- **THEN** the Gateway SHALL initialize `OpenSearchBulkStore` and route bulk writes and analytical queries through OpenSearch

#### Scenario: ClickHouse opt-in
- **WHEN** the Helm value `bulkStore.backend` is `"clickhouse"`
- **THEN** the Gateway SHALL initialize `ClickHouseBulkStore` and route bulk writes and analytical queries through ClickHouse

### Requirement: OpenSearch bulk store appends to evidence index
The `OpenSearchBulkStore` SHALL append evidence rows to the `compliance-evidence` OpenSearch index using bulk API. This is the same index used for search — the bulk store and search index are unified in the OpenSearch-default deployment.

#### Scenario: Bulk append
- **WHEN** the pipeline writes 500 evidence rows
- **THEN** the `OpenSearchBulkStore` SHALL issue an OpenSearch bulk index request for all 500 documents

### Requirement: ClickHouse bulk store uses time-partitioned table
The `ClickHouseBulkStore` SHALL write to a `ReplacingMergeTree` evidence table partitioned by `toYYYYMM(collected_at)` with configurable TTL. The schema SHALL match the current `evidence` table structure for analytical query compatibility.

#### Scenario: Analytical query via ClickHouse
- **WHEN** the agent queries evidence with `GROUP BY control_id` aggregations over a 12-month window
- **THEN** the `ClickHouseBulkStore` SHALL execute the query against the ClickHouse evidence table and return results

### Requirement: Agent queries route through bulk store adapter
The agent's evidence queries (via MCP tool) SHALL route through the `BulkStore` interface. The agent SHALL NOT need to know which backend is in use. Query capabilities available on both backends: time-range filtering, policy/control/target filtering, result aggregation.

#### Scenario: Agent evidence query
- **WHEN** the agent calls `run_select_query` with an evidence query
- **THEN** the MCP server SHALL route the query through the active `BulkStore` implementation
