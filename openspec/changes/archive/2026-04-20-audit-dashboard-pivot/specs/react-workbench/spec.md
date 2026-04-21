## MODIFIED Requirements

### Requirement: Workbench layout
The workbench SHALL use a sidebar navigation layout with four views (Posture, Policies, Evidence, Audit History) instead of the current editor-centric layout with artifact tabs and toolbar.

#### Scenario: Initial load
- **WHEN** the user loads the workbench
- **THEN** the system displays a sidebar with navigation items (Posture, Policies, Evidence, Audit History) and renders the Posture view as the default

#### Scenario: Navigation
- **WHEN** the user clicks a sidebar navigation item
- **THEN** the system renders the corresponding view in the main content area

## REMOVED Requirements

### Requirement: Artifact tabs
**Reason**: Artifact authoring moved to engineer's local toolchain (Cursor/Claude Code + gemara-mcp)
**Migration**: Policies imported via OCI registry into policy-store. No in-browser artifact editing.

### Requirement: Workspace toolbar
**Reason**: Toolbar actions (validate, publish, copy, auto-detect type) are authoring features
**Migration**: Validation happens in engineer's local toolchain. Publishing happens in CI/CD.

### Requirement: Multi-artifact workspace
**Reason**: The workspace was an in-browser artifact editor. Studio is now a dashboard.
**Migration**: Policies and AuditLogs stored in ClickHouse, viewed read-only in dashboard.
