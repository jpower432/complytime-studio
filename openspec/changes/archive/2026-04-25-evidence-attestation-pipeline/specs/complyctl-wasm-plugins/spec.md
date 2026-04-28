## ADDED Requirements

### Requirement: complyctl loads WASM plugins via wazero
complyctl SHALL load scanning provider plugins as WASM modules using the wazero runtime. WASM plugins SHALL be loaded from the local filesystem or pulled from an OCI registry.

#### Scenario: Load plugin from filesystem
- **WHEN** a plugin is configured with `path: "./plugins/openscap.wasm"`
- **THEN** complyctl SHALL load the WASM module from the local path

#### Scenario: Load plugin from OCI
- **WHEN** a plugin is configured with `path: "oci://registry.example.com/plugins/openscap:v1"`
- **THEN** complyctl SHALL pull the WASM module from OCI and cache it locally

#### Scenario: Content-addressed identity
- **WHEN** a WASM plugin is loaded from any source
- **THEN** complyctl SHALL compute SHA-256 of the WASM blob and use the hash as the plugin's identity

### Requirement: WASM plugin interface mirrors existing gRPC contract
WASM plugins SHALL export functions equivalent to the existing gRPC Plugin service: `describe`, `generate`, `scan`, and optionally `export`. Input and output are JSON-serialized equivalents of the protobuf messages.

#### Scenario: Describe
- **WHEN** complyctl calls the plugin's `describe` function
- **THEN** the plugin SHALL return a JSON object equivalent to `DescribeResponse` (healthy, version, required variables, supports_export)

#### Scenario: Generate
- **WHEN** complyctl calls the plugin's `generate` function with JSON-serialized global_variables, target_variables, and configurations
- **THEN** the plugin SHALL prepare policies and return success/failure

#### Scenario: Scan
- **WHEN** complyctl calls the plugin's `scan` function with JSON-serialized targets
- **THEN** the plugin SHALL return a JSON array of `AssessmentLog` entries with requirement_id, steps, message, and confidence

#### Scenario: Export (optional)
- **WHEN** a plugin declares `supports_export: true` in its describe response AND complyctl calls `export` with collector config
- **THEN** the plugin SHALL ship evidence to the Beacon collector endpoint

### Requirement: Host injects credentials as variables
complyctl SHALL resolve credentials using the existing variable model (global_variables, target_variables) and pass them to the WASM plugin via the `generate` and `scan` function inputs. Plugins SHALL NOT resolve, refresh, or store credentials.

#### Scenario: Credentials injected via target_variables
- **WHEN** a target in `complytime.yaml` specifies `variables: { api_token: "${MY_TOKEN}" }`
- **THEN** complyctl SHALL resolve `${MY_TOKEN}` from the environment and pass the resolved value to the plugin

#### Scenario: Plugin never sees credential resolution
- **WHEN** a WASM plugin is executed
- **THEN** the plugin SHALL receive only resolved credential values, never environment variable names or credential file paths

### Requirement: WASM plugins are sandboxed
WASM plugins SHALL run in a sandboxed environment with no filesystem access, no network access, and no environment variable access unless explicitly granted by the host.

#### Scenario: Default sandbox
- **WHEN** a plugin is loaded with default configuration
- **THEN** the plugin SHALL have no filesystem access, no network access, and no access to host environment variables

#### Scenario: Explicit filesystem grant
- **WHEN** a plugin is configured with `allow_read: ["/path/to/scan-target"]`
- **THEN** the plugin SHALL have read-only access to the specified path and no other filesystem access

### Requirement: complyctl produces attestation bundles
complyctl SHALL produce in-toto attestation bundles during `scan` execution and push them to the OCI registry. Each bundle SHALL contain signed links for the pipeline steps (policy fetch, scan execution).

#### Scenario: Attestation bundle produced after scan
- **WHEN** `complyctl scan --policy-id nist` completes successfully
- **THEN** complyctl SHALL produce an attestation bundle with signed links for each pipeline step and push it to the OCI registry

#### Scenario: Attestation ref included in evidence
- **WHEN** complyctl exports evidence to the OTel Collector after producing an attestation bundle
- **THEN** the `compliance.attestation.ref` attribute SHALL contain the OCI digest of the attestation bundle

#### Scenario: Attestation production is optional
- **WHEN** complyctl is run without attestation configuration (no signing key configured)
- **THEN** scan SHALL complete normally without producing attestation bundles and `compliance.attestation.ref` SHALL be absent from the evidence attributes

### Requirement: WASM and gRPC plugins coexist during migration
complyctl SHALL support loading both WASM plugins and legacy gRPC provider binaries simultaneously. Plugin type SHALL be determined by the configuration entry (WASM path vs provider binary name).

#### Scenario: Mixed plugin configuration
- **WHEN** `complytime.yaml` references both a WASM plugin (`path: "oci://registry/plugins/kyverno:v1"`) and a gRPC provider (`provider: complyctl-provider-openscap`)
- **THEN** complyctl SHALL load the WASM plugin via wazero and the gRPC provider via hashicorp/go-plugin, and both SHALL participate in scan execution
