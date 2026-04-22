## ADDED Requirements

### Requirement: policy_contacts table exists
The system SHALL create a `policy_contacts` ClickHouse table with columns `policy_id`, `raci_role`, `contact_name`, `contact_email`, `contact_affiliation`, and `imported_at`. The table SHALL use `ReplacingMergeTree(imported_at)` ordered by `(policy_id, raci_role, contact_name)`.

#### Scenario: Schema initialization
- **WHEN** the gateway starts and initializes the ClickHouse schema
- **THEN** a `policy_contacts` table exists with the specified columns and engine

### Requirement: Policy import parses RACI contacts into structured rows
When a policy is imported via `POST /api/policies/import`, the handler SHALL parse `Policy.contacts` from the YAML content using `go-gemara` types and batch-insert rows into `policy_contacts`.

#### Scenario: Policy with RACI contacts
- **WHEN** a policy is imported with `contacts` containing two `responsible` entries and one `informed` entry
- **THEN** three rows are inserted into `policy_contacts` with the correct `policy_id`, `raci_role`, `contact_name`, `contact_email`, and `contact_affiliation`

#### Scenario: Policy with no contacts
- **WHEN** a policy is imported with no `contacts` field or an empty contacts list
- **THEN** zero rows are inserted into `policy_contacts`
- **THEN** the policy blob is still stored successfully

#### Scenario: RACI parse failure
- **WHEN** the YAML content fails to parse as a Gemara Policy
- **THEN** a warning is logged with the policy_id and error
- **THEN** the raw policy blob is still stored (no HTTP error returned)

### Requirement: Re-import deduplicates contacts
When the same policy is re-imported, `ReplacingMergeTree` SHALL deduplicate on `(policy_id, raci_role, contact_name)` using the latest `imported_at`.

#### Scenario: Re-import with updated RACI
- **WHEN** a policy is re-imported with a changed `contacts` list
- **THEN** new contact rows are inserted
- **THEN** after ClickHouse merge, only the latest rows per `(policy_id, raci_role, contact_name)` remain

### Requirement: Retroactive population on startup
On gateway startup, the system SHALL backfill `policy_contacts` from existing `policies.content` for any policies that have zero corresponding `policy_contacts` rows.

#### Scenario: Existing policies without contacts rows
- **WHEN** the gateway starts and finds policies in the `policies` table with no matching `policy_contacts` rows
- **THEN** the system parses each policy's YAML content and inserts `policy_contacts` rows
- **THEN** policies that already have `policy_contacts` rows are skipped
