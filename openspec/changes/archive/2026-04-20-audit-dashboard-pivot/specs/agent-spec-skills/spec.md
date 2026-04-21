## REMOVED Requirements

### Requirement: Agent skill references via gitRefs
**Reason**: BYO agent bundles skills in container image. No kagent CRD skill mounting.
**Migration**: Skills copied into Docker image at build time.

### Requirement: Gemara authoring skill
**Reason**: Authoring skills belong to the gemara-mcp ecosystem, not Studio
**Migration**: `skills/gemara-authoring` and `skills/risk-reasoning` removed from this repo.
