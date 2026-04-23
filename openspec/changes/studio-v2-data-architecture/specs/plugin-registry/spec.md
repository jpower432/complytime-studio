## ADDED Requirements

### Requirement: Plugins are OCI artifacts pulled via ORAS
Ingestor plugins SHALL be packaged as OCI artifacts with media type `application/vnd.complytime.ingestor.wasm.v1`. The Gateway SHALL pull plugins from OCI registries using the existing ORAS infrastructure. Plugin references SHALL follow the format `<registry>/<repo>/<name>:<version>`.

#### Scenario: Gateway pulls plugin on first request
- **WHEN** an ingest request references plugin "nessus-xml" and the module is not cached locally
- **THEN** the Gateway SHALL pull the `.wasm` artifact from the configured OCI registry and cache it

#### Scenario: Plugin pulled from customer registry
- **WHEN** a customer configures a plugin reference `ghcr.io/acme/ingestors/custom-scanner:1.0.0`
- **THEN** the Gateway SHALL pull the `.wasm` artifact from that registry using ORAS with the customer's credentials

### Requirement: Plugin registry table tracks available ingestors
The system SHALL maintain an `ingestors` table in PostgreSQL with columns: `name`, `version`, `oci_reference`, `input_formats` (array), `semconv_version`, `author`, `loaded_at`, `status` (active/disabled). This table SHALL be the source of truth for available plugins.

#### Scenario: Plugin registered on pull
- **WHEN** the Gateway successfully pulls and validates a `.wasm` module
- **THEN** the system SHALL upsert a row in `ingestors` with metadata from the plugin's `metadata()` export

#### Scenario: List available ingestors
- **WHEN** the UI or API requests available ingestors
- **THEN** the system SHALL return all rows from `ingestors` where `status = 'active'`

### Requirement: Plugin validation on load
The Gateway SHALL validate every `.wasm` module before caching: (1) module exports `metadata` and `transform` functions, (2) `metadata()` returns valid `IngestorMetadata` with non-empty `name` and `version`, (3) module compiles without error under wazero. Invalid modules SHALL be rejected with a descriptive error.

#### Scenario: Valid plugin
- **WHEN** a `.wasm` module exports both required functions and metadata is valid
- **THEN** the Gateway SHALL cache the compiled module and register it in the `ingestors` table

#### Scenario: Missing transform export
- **WHEN** a `.wasm` module does not export a `transform` function
- **THEN** the Gateway SHALL reject the module with error "plugin missing required export: transform"
