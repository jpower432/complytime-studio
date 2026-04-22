## ADDED Requirements

### Requirement: Session stores Google Groups membership
The `ServerSession` SHALL include a `Groups []string` field populated at login time. The `Session` context object SHALL also carry `Groups` for downstream middleware.

#### Scenario: Login with groups claim
- **WHEN** the Google OIDC token includes a `groups` claim
- **THEN** `ServerSession.Groups` is populated with the group identifiers from the claim

#### Scenario: Login without groups claim
- **WHEN** the Google OIDC token does not include a `groups` claim
- **THEN** `ServerSession.Groups` is set to an empty slice
- **THEN** the login still succeeds (email-only matching is the fallback)

### Requirement: Access set resolved from policy_contacts
The system SHALL resolve a user's access set by querying `policy_contacts` for rows matching the user's email or any of their Google Groups. The highest RACI role per policy wins.

#### Scenario: User matches via group name
- **WHEN** a user's `Groups` contains `"platform-team"` and `policy_contacts` has a row `(policy_id="ampel-bp", raci_role="responsible", contact_name="platform-team")`
- **THEN** the access set includes `"ampel-bp": "responsible"`

#### Scenario: User matches via email
- **WHEN** a user's email is `alice@acme.com` and `policy_contacts` has a row `(policy_id="soc2-corp", raci_role="informed", contact_email="alice@acme.com")`
- **THEN** the access set includes `"soc2-corp": "informed"`

#### Scenario: User matches multiple roles for one policy
- **WHEN** a user matches `consulted` via email and `accountable` via group for the same `policy_id`
- **THEN** the access set returns the highest role: `"accountable"`

#### Scenario: User matches no contacts
- **WHEN** a user's email and groups match zero `policy_contacts` rows
- **THEN** the access set is empty

### Requirement: Access set injected into request context
The resolved `AccessSet map[string]string` (policy_id → raci_role) SHALL be injected into the request context by a middleware that runs after authentication.

#### Scenario: Middleware populates context
- **WHEN** an authenticated request reaches the access resolution middleware
- **THEN** the access set is resolved and stored in the request context
- **THEN** downstream handlers can read it via `AccessSetFrom(ctx)`

### Requirement: Graceful degradation when policy_contacts is empty
If the `policy_contacts` table has zero rows (fresh install, no RACI data), the access middleware SHALL allow all policies for all authenticated users.

#### Scenario: Empty policy_contacts table
- **WHEN** an authenticated user makes an API request and `policy_contacts` has zero rows
- **THEN** the access set contains all policy_ids with role `"accountable"` (full access)
