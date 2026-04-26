// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { currentView, navigate, inboxVersion } from "../app";
import { apiFetch } from "../api/fetch";

const PostureIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M3 12m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v6a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M9 8m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v10a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M15 4m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v14a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M4 20h14"/></svg>
);

const PoliciesIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 3l8 4.5l0 9l-8 4.5l-8 -4.5l0 -9l8 -4.5"/><path d="M12 12l8 -4.5"/><path d="M12 12l0 9"/><path d="M12 12l-8 -4.5"/></svg>
);

const EvidenceIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M10 10m-7 0a7 7 0 1 0 14 0a7 7 0 1 0 -14 0"/><path d="M21 21l-6 -6"/></svg>
);

const InboxIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M4 4m0 2a2 2 0 0 1 2 -2h12a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2z"/><path d="M4 13h3l3 3h4l3 -3h3"/></svg>
);

const NAV_ITEMS: { view: "posture" | "policies" | "evidence" | "inbox"; icon: () => any; label: string }[] = [
  { view: "posture", icon: PostureIcon, label: "Posture" },
  { view: "policies", icon: PoliciesIcon, label: "Policies" },
  { view: "evidence", icon: EvidenceIcon, label: "Evidence" },
  { view: "inbox", icon: InboxIcon, label: "Inbox" },
];

export function Sidebar() {
  const view = currentView.value;
  const [unreadCount, setUnreadCount] = useState(0);

  const version = inboxVersion.value;

  useEffect(() => {
    const fetchCount = () => {
      Promise.all([
        apiFetch("/api/notifications/unread-count")
          .then((r) => r.json()).catch(() => ({ count: 0 })),
        apiFetch("/api/draft-audit-logs?status=pending_review")
          .then((r) => r.json()).catch(() => []),
      ]).then(([notifs, drafts]: [{ count: number }, any[]]) => {
        setUnreadCount(notifs.count + drafts.length);
      });
    };
    fetchCount();
    const interval = setInterval(fetchCount, 30000);
    return () => clearInterval(interval);
  }, [version]);

  return (
    <aside class="sidebar">
      <nav class="sidebar-nav">
        {NAV_ITEMS.map((item) => (
          <button
            key={item.view}
            class={`sidebar-item ${view === item.view || (item.view === "posture" && view === "posture-detail") ? "active" : ""}`}
            onClick={() => navigate(item.view)}
          >
            <span class="sidebar-icon"><item.icon /></span>
            {item.label}
            {item.view === "inbox" && unreadCount > 0 && (
              <span class="inbox-badge">{unreadCount}</span>
            )}
          </button>
        ))}
      </nav>
    </aside>
  );
}
