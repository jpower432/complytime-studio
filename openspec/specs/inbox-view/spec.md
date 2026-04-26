## Requirements

### Requirement: Inbox displays agent drafts and notifications
The system SHALL display a unified inbox view showing agent-produced draft audit logs, posture change notifications, and evidence arrival summaries. Items SHALL be sorted by creation time, newest first.

#### Scenario: Inbox shows pending draft audit logs
- **WHEN** the agent produces a draft audit log via `publish_audit_log`
- **THEN** the inbox displays it as a card with policy name, date range, summary counts, and status "Pending Review"

#### Scenario: Inbox shows posture change notification
- **WHEN** the agent completes an event-triggered posture check that detects a pass rate change
- **THEN** the inbox displays a notification card with the policy name, previous pass rate, new pass rate, and delta

#### Scenario: Inbox shows evidence arrival summary
- **WHEN** new evidence records arrive for a policy within a 30-second window
- **THEN** the inbox displays a summary card with the policy name, record count, and timestamp

### Requirement: Inbox badge shows unread count
The system SHALL display a badge on the Inbox nav item showing the count of unread notifications. The badge SHALL update when new items arrive (on poll or navigation).

#### Scenario: Badge reflects unread count
- **WHEN** the user navigates to any view and there are 5 unread inbox items
- **THEN** the Inbox sidebar item displays a badge with "5"

#### Scenario: Badge clears when items are read
- **WHEN** the user opens the inbox and views all items
- **THEN** the badge disappears or shows "0"

### Requirement: Inbox items are markable as read
The system SHALL allow the user to mark individual inbox items as read. Opening an item's detail SHALL automatically mark it as read.

#### Scenario: Opening a draft marks it as read
- **WHEN** the user clicks a draft audit log card in the inbox
- **THEN** the item is marked as read and the unread badge decrements

### Requirement: Notifications stored with TTL
The system SHALL store inbox notifications in a `notifications` ClickHouse table with automatic expiration. Read notifications SHALL expire after 30 days. All notifications SHALL expire after 90 days.

#### Scenario: Old notifications are pruned
- **WHEN** a notification is older than 90 days
- **THEN** the system removes it from the notifications table
