## ADDED Requirements

### Requirement: studio-mcp exposes read-only resources for platform data
The `studio-mcp` server SHALL expose platform data as MCP resources with typed URIs. Resources are read-only views over the platform's PostgreSQL store.

#### Scenario: List policies
- **WHEN** an agent reads `studio://policies`
- **THEN** the server returns a JSON array of all policies (id, title, catalog references, created_at)

#### Scenario: Get single policy
- **WHEN** an agent reads `studio://policies/{id}`
- **THEN** the server returns the full policy including criteria and linked catalog IDs

#### Scenario: Query evidence with filters
- **WHEN** an agent reads `studio://evidence?policy_id=ac-1&limit=50`
- **THEN** the server returns up to 50 evidence records matching the policy filter

#### Scenario: Get posture aggregates
- **WHEN** an agent reads `studio://posture?policy_id=ac-1`
- **THEN** the server returns total, passed, and failed counts for the policy

#### Scenario: List audit logs
- **WHEN** an agent reads `studio://audit-logs?policy_id=ac-1`
- **THEN** the server returns audit log entries ordered by creation date descending

#### Scenario: Query mappings
- **WHEN** an agent reads `studio://mappings?source_catalog=nist-800-53`
- **THEN** the server returns crosswalk mapping entries for the specified source catalog

#### Scenario: List catalogs
- **WHEN** an agent reads `studio://catalogs`
- **THEN** the server returns all raw catalog artifacts (id, type, title, version)

#### Scenario: Query threats
- **WHEN** an agent reads `studio://threats?catalog_id=cloud-native-threats`
- **THEN** the server returns threat entries for the specified catalog

#### Scenario: Query risks
- **WHEN** an agent reads `studio://risks?catalog_id=cloud-native-risks`
- **THEN** the server returns risk entries for the specified catalog

### Requirement: studio-mcp exposes write tools
The `studio-mcp` server SHALL expose MCP tools for agent write operations.

#### Scenario: Ingest evidence via tool
- **WHEN** an agent calls the `ingest_evidence` tool with a valid evidence payload
- **THEN** the server inserts records into the evidence store and returns the insert count

#### Scenario: Save draft audit log via tool
- **WHEN** an agent calls the `save_draft_audit_log` tool with YAML content and policy_id
- **THEN** the server inserts a draft audit log record and returns the draft ID

### Requirement: studio-mcp connects to PostgreSQL using platform store interfaces
The `studio-mcp` server SHALL import `internal/store` and `internal/postgres` to access data. It SHALL NOT duplicate query logic.

#### Scenario: Schema compatibility
- **WHEN** the platform PostgreSQL schema is updated
- **THEN** `studio-mcp` uses the same store interfaces and receives the update without separate migration

### Requirement: studio-mcp supports stdio and HTTP transport
The server SHALL support stdio transport (for sidecar deployment in agent pods) and HTTP transport (for standalone deployment).

#### Scenario: Sidecar mode
- **WHEN** `studio-mcp` is deployed as a sidecar with `--transport stdio`
- **THEN** it communicates via stdin/stdout with the agent container

#### Scenario: Standalone HTTP mode
- **WHEN** `studio-mcp` is deployed standalone with `--transport http --port 3000`
- **THEN** it serves MCP over HTTP on the specified port

### Requirement: Resource pagination
Resources that can return large result sets SHALL support `limit` and `offset` query parameters.

#### Scenario: Paginated evidence query
- **WHEN** an agent reads `studio://evidence?policy_id=ac-1&limit=20&offset=40`
- **THEN** the server returns records 41-60 matching the filter
