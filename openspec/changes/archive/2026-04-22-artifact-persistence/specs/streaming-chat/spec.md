## MODIFIED Requirements

### Requirement: Artifact save button is a confirmation action

Previously: The "Save to Audit History" button was the **only** path to persist an agent-produced artifact.

The "Save to Audit History" button SHALL remain available on artifact cards for admin users. When server-side auto-persistence is enabled, the button SHALL function as an idempotent confirmation or re-save action. The artifact card SHALL display an "Auto-saved" indicator when the artifact was persisted server-side.

#### Scenario: Auto-persist enabled, artifact displayed
- **WHEN** an artifact card is rendered and `AUTO_PERSIST_ARTIFACTS` is enabled
- **THEN** the card SHALL display an "Auto-saved" text indicator
- **AND** the "Save to Audit History" button SHALL still be available for admin users

#### Scenario: Auto-persist disabled, artifact displayed
- **WHEN** an artifact card is rendered and `AUTO_PERSIST_ARTIFACTS` is disabled
- **THEN** the card SHALL NOT display an "Auto-saved" indicator
- **AND** the "Save to Audit History" button SHALL be the primary save action (current behavior)

#### Scenario: Manual save after auto-persist
- **WHEN** an admin clicks "Save to Audit History" on an auto-persisted artifact
- **THEN** the `POST /api/audit-logs` request SHALL succeed
- **AND** `ReplacingMergeTree` SHALL deduplicate the row if content is unchanged
