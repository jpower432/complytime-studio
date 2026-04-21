## MODIFIED Requirements

### Requirement: Domain-specific store interfaces

The monolithic `Store` struct SHALL be split into domain-specific interfaces:
`EvidenceStore`, `PolicyStore`, `AuditLogStore`, `MappingStore`. Handlers
SHALL depend on the interface, not the concrete struct. Concrete
implementations backed by ClickHouse satisfy the interfaces.

#### Scenario: Handler depends on interface
- **WHEN** an HTTP handler needs evidence operations
- **THEN** it receives an `EvidenceStore` interface, not a `*Store` concrete type
- **THEN** the interface can be satisfied by a ClickHouse implementation or a test mock

#### Scenario: Future extraction enabled
- **WHEN** a domain (e.g. evidence) needs to move to a standalone service
- **THEN** the handler's interface dependency can be swapped from a ClickHouse-backed implementation to an HTTP client without changing handler code

### Requirement: No cross-imports between sibling internal packages

No `internal/` package SHALL import another sibling `internal/` package
directly, except for `internal/httputil` and `internal/consts` as shared
leaf dependencies. Cross-cutting behavior SHALL flow through exported Go
interfaces.

#### Scenario: Boundary interfaces are explicit
- **WHEN** package A depends on behavior from package B
- **THEN** A depends on an interface type that B's constructors satisfy
- **THEN** A does not reference B's concrete handler or store structs

#### Scenario: httputil and consts remain shared leaves
- **WHEN** any `internal/` package needs `WriteJSON`, `EnvOr`, or `TokenProvider`
- **THEN** it imports `internal/httputil` or `internal/consts`
- **THEN** those packages do not import non-leaf domain packages
