## ADDED Requirements

### Requirement: Go module path is complytime/complytime-studio
The Go module SHALL use `github.com/complytime/complytime-studio` as its module path.

#### Scenario: Module declaration
- **WHEN** the `go.mod` file is read
- **THEN** the module line SHALL be `module github.com/complytime/complytime-studio`

### Requirement: Agent names use studio prefix
All agent names SHALL use the `studio-` prefix instead of `gide-`.

#### Scenario: Orchestrator name
- **WHEN** the orchestrator agent is created
- **THEN** its name SHALL be `studio-orchestrator`

#### Scenario: Specialist names
- **WHEN** specialist agents are created
- **THEN** their names SHALL be `studio-threat-modeler`, `studio-gap-analyst`, and `studio-policy-composer`

### Requirement: Prompt files reference ComplyTime Studio
All embedded prompt markdown files SHALL reference "ComplyTime Studio" instead of "GIDE".

#### Scenario: Orchestrator prompt
- **WHEN** the orchestrator prompt is read
- **THEN** it SHALL contain "ComplyTime Studio" and not contain "GIDE"

#### Scenario: Specialist prompts
- **WHEN** any specialist prompt is read
- **THEN** it SHALL contain "ComplyTime Studio" and not contain "GIDE"

### Requirement: Helm chart renamed to complytime-studio
The Helm chart SHALL be located at `charts/complytime-studio/` with chart name `complytime-studio`.

#### Scenario: Chart.yaml name
- **WHEN** `Chart.yaml` is read
- **THEN** the `name` field SHALL be `complytime-studio`

#### Scenario: Agent CRD metadata
- **WHEN** the kagent Agent CRD is rendered
- **THEN** `metadata.name` SHALL be `studio-orchestrator`

### Requirement: Docker image names use studio prefix
Container image names SHALL use the `studio-` prefix.

#### Scenario: Agents image
- **WHEN** the agents container image is built
- **THEN** the image name SHALL be `studio-agents`

### Requirement: Workbench branding shows ComplyTime Studio
The workbench UI SHALL display "ComplyTime Studio" in the header and page title.

#### Scenario: Page title
- **WHEN** the workbench loads in a browser
- **THEN** the page title SHALL be "ComplyTime Studio"

#### Scenario: Header logo
- **WHEN** the workbench header renders
- **THEN** it SHALL display "ComplyTime Studio" instead of "GIDE"

### Requirement: Environment variable names preserved
Environment variable names (PORT, MCP_TRANSPORT, VERTEX_PROJECT_ID, etc.) SHALL remain unchanged to avoid breaking deployment configurations.

#### Scenario: Config loading
- **WHEN** the config loader reads environment variables
- **THEN** all variable names SHALL match the existing GIDE convention (no GIDE prefix existed, so no rename needed)
