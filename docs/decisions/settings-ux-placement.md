# Settings UX: Sidebar Bottom Gear Icon

**Status:** Accepted
**Date:** 2026-04-27

## Context

Settings (user management, role changes audit log) was initially placed as a main sidebar nav item alongside Posture, Policies, Evidence, and Inbox. A subsequent iteration moved a gear icon to the header bar. Both placements elevated admin-only functionality to the same visual hierarchy as core workflow views, creating noise for non-admin users and cluttering the header.

## Decision

Place settings access as a **standalone gear icon in the bottom-left corner of the sidebar**, separated from the main navigation items. Visible only to admin-role users.

## Rationale

- **Information hierarchy**: Admin settings are secondary to the audit workflow. They don't belong alongside Posture/Policies/Evidence.
- **Convention**: Bottom-left gear is a well-established pattern (VS Code, Slack, Discord, GitHub Desktop). Users expect system settings there.
- **Minimal footprint**: Icon-only (no label) keeps the sidebar clean. `title` and `aria-label` provide discoverability.
- **Role gating**: Non-admins never see the icon. Zero UI noise for reviewers.

## Alternatives Considered

| Approach | Rejected Because |
|:--|:--|
| Sidebar nav item | Elevates admin function to same level as core views |
| Header gear icon | Clutters header, competes with external links and theme toggle |
| User avatar dropdown menu | Hides settings behind an extra click; less discoverable |
