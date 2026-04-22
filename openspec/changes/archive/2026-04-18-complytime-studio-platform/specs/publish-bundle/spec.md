## ADDED Requirements

### Requirement: Native publish_bundle tool on the orchestrator
The system SHALL provide a `publish_bundle` function registered as an ADK `tool.Func` on the orchestrator agent, using `oras-go` for OCI push operations.

#### Scenario: Tool available to orchestrator
- **WHEN** the orchestrator initializes
- **THEN** `publish_bundle` SHALL be registered as a callable tool alongside the oras-mcp read toolset

### Requirement: Bundle assembly from artifact YAML
The system SHALL assemble an OCI manifest from one or more validated Gemara artifact YAML strings, mapping each artifact's `metadata.type` to the correct OCI media type.

#### Scenario: Single artifact push
- **WHEN** `publish_bundle` is called with one artifact (e.g., a ThreatCatalog)
- **THEN** the tool SHALL create an OCI manifest with one layer using media type `application/vnd.gemara.threat-catalog.layer.v1+yaml`

#### Scenario: Multi-artifact push
- **WHEN** `publish_bundle` is called with multiple artifacts (e.g., ThreatCatalog + ControlCatalog + Policy)
- **THEN** the tool SHALL create an OCI manifest with one layer per artifact, each with its correct media type

#### Scenario: Unknown artifact type
- **WHEN** an artifact's `metadata.type` does not match a known Gemara artifact type
- **THEN** the tool SHALL return an error identifying the unrecognized type

### Requirement: OCI push to target registry
The system SHALL push the assembled manifest and layers to an OCI-compliant registry using `oras-go`.

#### Scenario: Push with user-specified reference
- **WHEN** `publish_bundle` is called with a `target` and `tag` argument
- **THEN** the tool SHALL push to `<target>:<tag>`

#### Scenario: Push with metadata-derived reference
- **WHEN** `publish_bundle` is called without a `target` argument
- **THEN** the tool SHALL derive the repository path from `metadata.id` and the tag from `metadata.version`

#### Scenario: Registry authentication
- **WHEN** pushing to a registry that requires authentication
- **THEN** the tool SHALL use credentials from the standard `docker login` / `oras login` credential stores

#### Scenario: Push failure
- **WHEN** the push fails (auth error, network error, registry unavailable)
- **THEN** the tool SHALL return a structured error with the failure reason

### Requirement: Bundle signing after push
The system SHALL sign the pushed manifest digest using `notation-go` or `cosign-go` after a successful push.

#### Scenario: Signing enabled (default)
- **WHEN** `publish_bundle` completes a push and signing is not explicitly disabled
- **THEN** the tool SHALL sign the manifest digest and return `{ reference, digest, signature_digest }`

#### Scenario: Signing disabled
- **WHEN** `publish_bundle` is called with `sign: false`
- **THEN** the tool SHALL skip signing and return `{ reference, digest }` only

#### Scenario: Signing key not configured
- **WHEN** signing is enabled but no signing key or OIDC identity is configured
- **THEN** the tool SHALL return an error indicating signing configuration is required

### Requirement: Media type mapping table
The system SHALL maintain a mapping from Gemara `metadata.type` values to OCI media types as a centralized constant.

#### Scenario: All Gemara artifact types mapped
- **WHEN** the media type table is referenced
- **THEN** it SHALL include entries for at minimum: CapabilityCatalog, ControlCatalog, ThreatCatalog, GuidanceCatalog, RiskCatalog, Policy, MappingDocument, AuditLog, EvaluationLog, EnforcementLog, VectorCatalog, PrincipleCatalog
