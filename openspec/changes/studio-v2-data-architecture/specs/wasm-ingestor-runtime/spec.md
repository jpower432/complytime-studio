## ADDED Requirements

### Requirement: Gateway embeds wazero WASM runtime
The Gateway SHALL embed a `wazero` WASM runtime for executing ingestor plugins. The runtime SHALL be initialized at startup and shared across requests. Compiled WASM modules SHALL be cached in memory to avoid recompilation on each invocation.

#### Scenario: First invocation compiles and caches
- **WHEN** an ingest request arrives for plugin "nessus-xml" and the module is not cached
- **THEN** the runtime SHALL compile the `.wasm` bytes, cache the compiled module, and execute the transform

#### Scenario: Subsequent invocations use cache
- **WHEN** an ingest request arrives for plugin "nessus-xml" and the module is cached
- **THEN** the runtime SHALL instantiate from the cached compiled module without recompilation

### Requirement: WASM sandbox restricts plugin capabilities
Each plugin invocation SHALL run in a capability-restricted WASI sandbox. The sandbox SHALL NOT provide network access, filesystem access, environment variable access, or process spawning. The host SHALL enforce a configurable memory limit (default: 64MB) and execution timeout (default: 30s) per invocation.

#### Scenario: Plugin exceeds memory limit
- **WHEN** a plugin attempts to allocate memory beyond the configured limit
- **THEN** the runtime SHALL terminate the plugin and return an error to the ingestion pipeline

#### Scenario: Plugin exceeds execution timeout
- **WHEN** a plugin's `transform` call runs longer than the configured timeout
- **THEN** the runtime SHALL terminate the plugin and return a timeout error

#### Scenario: Plugin attempts network access
- **WHEN** a plugin attempts to open a network socket via WASI
- **THEN** the runtime SHALL deny the capability and the call SHALL fail

### Requirement: Plugin invocation returns structured results
The runtime SHALL deserialize plugin output into the host's `EvidenceBatch` struct containing `rows` ([]EvidenceRow), `warnings` ([]string), and `stats` (total_parsed, total_skipped, duration_ms). Deserialization failure SHALL be treated as a plugin error.

#### Scenario: Plugin returns valid output
- **WHEN** a plugin's `transform` call returns serialized bytes matching the `EvidenceBatch` schema
- **THEN** the runtime SHALL deserialize the output and pass `EvidenceBatch` to the ingestion pipeline

#### Scenario: Plugin returns malformed output
- **WHEN** a plugin's `transform` call returns bytes that do not match the `EvidenceBatch` schema
- **THEN** the runtime SHALL return a deserialization error with the plugin name and version
