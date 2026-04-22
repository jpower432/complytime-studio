## MODIFIED Requirements

### Requirement: Decision record for architecture

A decision record SHALL be created under `docs/decisions/` documenting:
- **Chosen path:** modulith gateway + extracted A2A proxy
- **Rationale:** data plane and agent plane scale differently, fail differently, share no state
- **Rejected alternatives:** evidence service extraction (premature, shared CH schema), full decomposition (overkill)
- **Future options:** evidence extraction enabled by interface boundaries

#### Scenario: Decision record exists
- **WHEN** this change is complete
- **THEN** a decision document exists in `docs/decisions/`
- **THEN** the document names the selected architecture and rejected alternatives with rationale

### Requirement: Module boundary coupling audit

Each package under `internal/` SHALL be assessed for coupling to other
packages, producing a documented coupling matrix.

#### Scenario: Per-package coupling matrix
- **WHEN** the decomposition evaluation runs
- **THEN** every top-level `internal/` package has a documented coupling assessment
- **THEN** the assessment confirms `internal/agents/` has zero ClickHouse or session coupling

### Requirement: Public contract for A2A proxy

The extracted A2A proxy SHALL define its public contract: paths, methods,
headers, status codes, and streaming behavior. The contract SHALL be
documented so that ingress configuration and workbench integration can be
verified against it.

#### Scenario: Contract is documented
- **WHEN** the A2A proxy is extracted
- **THEN** its HTTP surface is documented (paths, methods, expected headers, error codes)
- **THEN** the workbench's `a2a.ts` can be verified against the contract
