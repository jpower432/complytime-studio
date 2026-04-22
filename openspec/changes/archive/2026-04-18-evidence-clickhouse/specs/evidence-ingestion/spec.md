## ADDED Requirements

### Requirement: Loader validates Gemara artifacts before ingestion

The ingestion loader SHALL validate each input artifact against the Gemara CUE schema (`#EvaluationLog` or `#EnforcementLog`) before flattening. If validation fails, the loader SHALL reject the artifact with a descriptive error and insert zero rows.

#### Scenario: Valid EvaluationLog artifact

- **WHEN** a valid Gemara EvaluationLog YAML file is provided to the loader
- **THEN** the loader validates successfully and proceeds to flatten and insert rows

#### Scenario: Invalid artifact rejected

- **WHEN** a YAML file that fails `#EvaluationLog` schema validation is provided
- **THEN** the loader exits with a non-zero status and prints validation errors without writing any rows

### Requirement: Loader flattens nested EvaluationLog structure into rows

The loader SHALL iterate each `ControlEvaluation` and each nested `AssessmentLog` to produce one row per assessment. Each row SHALL carry the parent evaluation's control-level fields as denormalized columns. The `policy_id` SHALL be derived from the `metadata.mapping-references` that correspond to a Policy artifact.

#### Scenario: EvaluationLog with nested assessments

- **WHEN** an EvaluationLog has 3 ControlEvaluations with 2, 1, and 3 AssessmentLogs respectively
- **THEN** the loader produces 6 rows total, each with correct parent control context

#### Scenario: AssessmentLog with optional fields absent

- **WHEN** an AssessmentLog omits `plan`, `recommendation`, and `confidence-level`
- **THEN** the corresponding row columns are NULL

### Requirement: Loader flattens nested EnforcementLog structure into rows

The loader SHALL iterate each `ActionResult` and each nested `AssessmentFinding` within `justification.assessments` to produce one row per finding. Each row SHALL carry the parent action's `disposition`, `method`, and timestamps as denormalized columns.

#### Scenario: ActionResult with multiple assessment findings

- **WHEN** an ActionResult has 3 AssessmentFindings in its justification
- **THEN** the loader produces 3 rows, each with the parent action's disposition and method

#### Scenario: ActionResult with exceptions

- **WHEN** an ActionResult has `justification.exceptions` with 2 entries
- **THEN** each row sets `has_exception = true` and `exception_refs` contains the 2 reference-id values

### Requirement: Loader is idempotent on re-ingestion

The loader SHALL use `log_id` + a deterministic row identifier (e.g., `control_id` + `requirement_id` + assessment index) to prevent duplicate rows on re-ingestion of the same artifact. Re-ingesting an artifact SHALL produce the same final state as the initial ingestion.

#### Scenario: Same EvaluationLog ingested twice

- **WHEN** the same EvaluationLog YAML is ingested a second time
- **THEN** no duplicate rows exist in `evaluation_logs`

### Requirement: Loader accepts file path or stdin

The loader SHALL accept a Gemara YAML artifact as a file path argument or from stdin. This enables both interactive use (`complyctl ingest eval-log.yaml`) and pipeline use (`cat eval-log.yaml | complyctl ingest -`).

#### Scenario: File path argument

- **WHEN** user runs `complyctl ingest /path/to/evaluation-log.yaml`
- **THEN** the loader reads and processes the file

#### Scenario: Stdin input

- **WHEN** user pipes YAML to `complyctl ingest -`
- **THEN** the loader reads from stdin and processes the artifact
