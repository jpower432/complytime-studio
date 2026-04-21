## ADDED Requirements

### Requirement: Upload evidence file
The system SHALL accept CSV and JSON file uploads via `POST /api/evidence/upload` and ingest rows into the ClickHouse evidence table.

#### Scenario: CSV upload
- **WHEN** the user uploads a CSV file with columns matching the evidence schema (policy_id, target_id, control_id, requirement_id, result, collected_at)
- **THEN** the system parses the CSV, validates required columns exist, and inserts rows into ClickHouse

#### Scenario: JSON upload
- **WHEN** the user uploads a JSON file containing an array of evidence objects
- **THEN** the system validates each object against the evidence schema and inserts valid rows into ClickHouse

#### Scenario: Partial failure
- **WHEN** some rows in an uploaded file fail validation
- **THEN** the system inserts valid rows and returns a response listing failed rows with error descriptions

### Requirement: Upload via dashboard
The system SHALL provide a file upload button in the Evidence view that accepts drag-and-drop or file picker selection.

#### Scenario: Drag-and-drop upload
- **WHEN** the user drags a CSV or JSON file onto the Evidence view upload area
- **THEN** the system uploads the file and displays a progress indicator followed by an import summary (rows imported, rows failed)
