## ADDED Requirements

### Requirement: Helm chart deploys a single-node ClickHouse instance

The Helm chart SHALL include a StatefulSet for ClickHouse with a single replica, a PersistentVolumeClaim for data storage, and a Service for in-cluster access. The StatefulSet SHALL use the official `clickhouse/clickhouse-server` image.

#### Scenario: Fresh Helm install provisions ClickHouse

- **WHEN** `helm install` is run with ClickHouse enabled
- **THEN** a ClickHouse StatefulSet, PVC, and Service are created in the target namespace

#### Scenario: ClickHouse disabled by default

- **WHEN** `helm install` is run without setting `clickhouse.enabled=true`
- **THEN** no ClickHouse resources are created

### Requirement: Helm chart deploys mcp-clickhouse as a Deployment

The Helm chart SHALL include a Deployment for the official `mcp-clickhouse` MCP server. The Deployment SHALL be configured with environment variables for the ClickHouse connection (`CLICKHOUSE_HOST`, `CLICKHOUSE_PORT`, `CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`) sourced from a Kubernetes Secret. The MCP server SHALL run in `--readonly` mode.

#### Scenario: mcp-clickhouse connects to in-cluster ClickHouse

- **WHEN** both ClickHouse and mcp-clickhouse are deployed
- **THEN** mcp-clickhouse connects to `studio-clickhouse.{namespace}:8123` using credentials from the Secret

### Requirement: Gap Analyst BYO Agent CRD references clickhouse-mcp

The Gap Analyst's Agent CRD SHALL include a `type: McpServer` tool entry referencing the `mcp-clickhouse` Deployment. The tool entry SHALL filter to expose only `run_select_query`, `list_databases`, and `list_tables` tools.

#### Scenario: Gap Analyst can query ClickHouse

- **WHEN** the Gap Analyst agent is running and the clickhouse-mcp server is healthy
- **THEN** the agent can invoke `run_select_query` to read from `evaluation_logs` and `enforcement_actions`

#### Scenario: Write operations are not exposed

- **WHEN** the agent attempts any operation other than SELECT
- **THEN** the mcp-clickhouse server rejects the query (readonly mode)

### Requirement: Schema initialization via init container or migration job

The ClickHouse StatefulSet SHALL use an init container or a Kubernetes Job to execute DDL statements that create the `evaluation_logs` and `enforcement_actions` tables if they do not exist. The DDL SHALL be stored in a ConfigMap.

#### Scenario: First deployment creates tables

- **WHEN** ClickHouse starts for the first time
- **THEN** the init container creates both tables with correct schema, partitioning, sort keys, and TTL

#### Scenario: Subsequent restarts do not duplicate tables

- **WHEN** ClickHouse restarts after tables already exist
- **THEN** the init container runs `CREATE TABLE IF NOT EXISTS` and completes without error

### Requirement: ClickHouse credentials stored in Kubernetes Secret

The Helm chart SHALL create a Secret containing ClickHouse credentials (`CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`). Both the ClickHouse StatefulSet and the mcp-clickhouse Deployment SHALL reference this Secret. The password SHALL be configurable via `values.yaml` with a default suitable for development.

#### Scenario: Credentials shared between ClickHouse and MCP server

- **WHEN** both resources are deployed
- **THEN** both reference the same Secret and the MCP server authenticates successfully
