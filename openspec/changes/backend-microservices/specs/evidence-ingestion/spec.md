## DEFERRED Requirements

> Evidence ingestion extraction is deferred. The interface boundaries
> established in this change make it a packaging change when needed.

### Requirement: Evidence ingestion deploy topology is configurable

The HTTP evidence ingestion endpoints SHALL be buildable and deployable either
as part of the gateway modulith or as a standalone service, selected
**without** changing the public URL paths or JSON contracts observed by
clients. The modulith interface boundaries (`EvidenceStore` interface) make
this possible without code changes beyond wiring.

#### Scenario: Modulith build (current)
- **WHEN** the deployment uses the modulith configuration
- **THEN** the gateway binary registers and serves evidence ingestion routes in-process

#### Scenario: Future standalone build
- **WHEN** the deployment uses a standalone evidence-ingestion configuration
- **THEN** a dedicated binary exposes the same ingest routes and semantics
- **THEN** clients or ingress rules are updated only at the network layer
