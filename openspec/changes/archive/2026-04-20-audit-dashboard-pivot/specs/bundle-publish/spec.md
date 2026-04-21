## REMOVED Requirements

### Requirement: Publish artifacts to OCI registry from workbench
**Reason**: Publishing happens in engineer's CI/CD pipeline, not in Studio
**Migration**: Engineers publish artifacts using `oras push` or `gemara publish` in their CI/CD. Studio imports from registries.
