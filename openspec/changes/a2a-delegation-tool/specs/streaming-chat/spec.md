## MODIFIED Requirements

### Requirement: Stream message to agent
The `streamMessage` function SHALL always route to `studio-assistant`. The `agentName` parameter SHALL be removed.

#### Scenario: User sends a message
- **WHEN** the user submits a message in the chat
- **THEN** `streamMessage` SHALL POST to `/a2a/studio-assistant`
- **THEN** no agent selection logic SHALL influence the routing

### Requirement: Stream reply to agent
The `streamReply` function SHALL always route to `studio-assistant`. The `agentName` parameter SHALL be removed.

#### Scenario: User replies to an active task
- **WHEN** the user sends a reply to an existing task
- **THEN** `streamReply` SHALL POST to `/a2a/studio-assistant`
- **THEN** no agent selection logic SHALL influence the routing

## REMOVED Requirements

### Requirement: Agent picker switches session target
**Reason**: Switching agents wiped session context. The assistant now owns the session permanently and delegates to BYO agents via A2A tool calls.
**Migration**: Remove `agentName` parameter from `streamMessage` and `streamReply`. Remove `handleNewSession()` call from picker `onChange`. Always route to `studio-assistant`.
