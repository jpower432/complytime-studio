## ADDED Requirements

### Requirement: Threat list API with optional filters
The system SHALL expose `GET /api/threats` returning `[]ThreatRow` JSON. Optional query parameters: `catalog_id`, `policy_id`, `limit`. When no filters are provided the endpoint SHALL return rows up to the default query limit (100). Limit is clamped to `consts.MaxQueryLimit` (1000).

#### Scenario: Filter by catalog_id
- **WHEN** a client sends `GET /api/threats?catalog_id=tc-cnsc`
- **THEN** the response SHALL contain only threats with `catalog_id = "tc-cnsc"`, status 200

#### Scenario: No filters returns capped results
- **WHEN** a client sends `GET /api/threats` with no query parameters
- **THEN** the response SHALL contain at most `DefaultQueryLimit` (100) rows, status 200

#### Scenario: Empty result
- **WHEN** a client sends `GET /api/threats?catalog_id=nonexistent`
- **THEN** the response SHALL be an empty JSON array `[]`, status 200

### Requirement: Control-threat junction API
The system SHALL expose `GET /api/control-threats` returning `[]ControlThreatRow` JSON. Optional query parameters: `catalog_id`, `control_id`, `limit`. Limit clamped to `consts.MaxQueryLimit`.

#### Scenario: Filter by control_id
- **WHEN** a client sends `GET /api/control-threats?control_id=ctrl-1`
- **THEN** the response SHALL contain only rows where `control_id = "ctrl-1"`

### Requirement: Risk list API with optional filters
The system SHALL expose `GET /api/risks` returning `[]RiskRow` JSON. Optional query parameters: `catalog_id`, `policy_id`, `limit`. Limit clamped to `consts.MaxQueryLimit`.

#### Scenario: Filter by policy_id
- **WHEN** a client sends `GET /api/risks?policy_id=pol-1`
- **THEN** the response SHALL contain only risks with `policy_id = "pol-1"`, status 200

### Requirement: Risk-threat junction API
The system SHALL expose `GET /api/risk-threats` returning `[]RiskThreatRow` JSON. Optional query parameters: `catalog_id`, `risk_id`, `limit`. Limit clamped to `consts.MaxQueryLimit`.

#### Scenario: Filter by risk_id
- **WHEN** a client sends `GET /api/risk-threats?risk_id=r-1`
- **THEN** the response SHALL contain only rows where `risk_id = "r-1"`

### Requirement: All list endpoints enforce query limit cap
All four endpoints SHALL clamp the `limit` query parameter using `consts.ClampLimit`. This is consistent with the accepted [Query Limit Cap](../../../docs/decisions/query-limit-cap.md) ADR.

#### Scenario: Excessive limit is silently clamped
- **WHEN** a client sends `GET /api/threats?limit=9999`
- **THEN** the store query SHALL use `LIMIT 1000` (MaxQueryLimit)

### Requirement: go-gemara SDK Load for catalog parsing
Catalog parsing (`ParseControlCatalog`, `ParseThreatCatalog`, `ParseRiskCatalog`) SHALL use the `go-gemara` `sdk.Load` API with a `MemoryFetcher` instead of raw YAML unmarshal, providing schema-aware parsing.

#### Scenario: ControlCatalog import via SDK
- **WHEN** a ControlCatalog YAML is imported via `POST /api/catalogs/import`
- **THEN** `ParseControlCatalog` SHALL invoke `sdk.Load` with a `MemoryFetcher` and return structured rows
