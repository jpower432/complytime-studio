## MODIFIED Requirements

### Requirement: Artifact panel has download action
The artifact panel SHALL include a "Download YAML" button that exports the selected artifact content as a local file.

#### Scenario: Download active artifact
- **WHEN** user clicks "Download YAML" in the workspace toolbar
- **THEN** the browser downloads the active workspace artifact's YAML
- **THEN** the filename is the artifact name (e.g., `threat-catalog.yaml`)

#### Scenario: Download from job history
- **WHEN** user clicks "Download YAML" on a job artifact in the jobs view
- **THEN** the browser downloads that specific artifact's YAML

### Requirement: Download all workspace artifacts
The workspace toolbar SHALL include a "Download All" action that exports all workspace artifacts as individual YAML files.

#### Scenario: Download all as zip
- **WHEN** user clicks "Download All" with multiple artifacts in the workspace
- **THEN** the browser downloads a zip file containing all workspace artifacts as individual YAML files
- **THEN** the zip filename includes a timestamp (e.g., `complytime-workspace-2026-04-18.zip`)

#### Scenario: Single artifact workspace
- **WHEN** user clicks "Download All" with one artifact in the workspace
- **THEN** the browser downloads the single YAML file directly (no zip wrapper)
