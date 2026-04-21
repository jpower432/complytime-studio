## REMOVED Requirements

### Requirement: Multi-artifact workspace with localStorage persistence
**Reason**: Replaced by policy-store (ClickHouse-backed). No client-side artifact management.
**Migration**: Policies imported from OCI registry and stored server-side in ClickHouse.

### Requirement: Artifact rename and management
**Reason**: No client-side artifacts to manage
**Migration**: Server-side policies identified by policy_id and version from OCI metadata.
