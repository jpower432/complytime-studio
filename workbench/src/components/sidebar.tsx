// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { currentView, currentUser, navigate, inboxVersion, type View } from "../app";
import { apiFetch } from "../api/fetch";

const SettingsIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.066 2.573c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.573 1.066c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.066 -2.573c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z"/><path d="M9 12a3 3 0 1 0 6 0a3 3 0 0 0 -6 0"/></svg>
);

const DashboardIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M4 4m0 2a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v2a2 2 0 0 1 -2 2h-2a2 2 0 0 1 -2 -2z"/><path d="M14 4m0 2a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v2a2 2 0 0 1 -2 2h-2a2 2 0 0 1 -2 -2z"/><path d="M4 14m0 2a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v2a2 2 0 0 1 -2 2h-2a2 2 0 0 1 -2 -2z"/><path d="M14 14m0 2a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v2a2 2 0 0 1 -2 2h-2a2 2 0 0 1 -2 -2z"/></svg>
);

const ProgramsIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 3a12 12 0 0 0 8.5 3a12 12 0 0 1 -8.5 15a12 12 0 0 1 -8.5 -15a12 12 0 0 0 8.5 -3"/><path d="M9 12l2 2l4 -4"/></svg>
);

const PoliciesIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 3l8 4.5l0 9l-8 4.5l-8 -4.5l0 -9l8 -4.5"/><path d="M12 12l8 -4.5"/><path d="M12 12l0 9"/><path d="M12 12l-8 -4.5"/></svg>
);

const InventoryIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M3 4m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v2a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z"/><path d="M3 12m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v2a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z"/><path d="M7 8v0"/><path d="M7 16v0"/></svg>
);

const EvidenceIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M10 10m-7 0a7 7 0 1 0 14 0a7 7 0 1 0 -14 0"/><path d="M21 21l-6 -6"/></svg>
);

const ReviewsIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M9 5h-2a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-12a2 2 0 0 0 -2 -2h-2"/><path d="M9 3m0 2a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v0a2 2 0 0 1 -2 2h-2a2 2 0 0 1 -2 -2z"/><path d="M9 12l2 2l4 -4"/></svg>
);

const NAV_ITEMS: {
  view: "dashboard" | "programs" | "policies" | "inventory" | "evidence" | "reviews";
  icon: () => any;
  label: string;
}[] = [
  { view: "dashboard", icon: DashboardIcon, label: "Dashboard" },
  { view: "programs", icon: ProgramsIcon, label: "Programs" },
  { view: "policies", icon: PoliciesIcon, label: "Policies" },
  { view: "inventory", icon: InventoryIcon, label: "Inventory" },
  { view: "evidence", icon: EvidenceIcon, label: "Evidence" },
  { view: "reviews", icon: ReviewsIcon, label: "Reviews" },
];

function navItemActive(itemView: (typeof NAV_ITEMS)[number]["view"], view: View): boolean {
  if (view === itemView) {
    return true;
  }
  if (itemView === "programs" && view === "program-detail") {
    return true;
  }
  if (itemView === "reviews" && view === "review-workspace") {
    return true;
  }
  return false;
}

export function Sidebar() {
  const view = currentView.value;
  const [draftReviewCount, setDraftReviewCount] = useState(0);

  const version = inboxVersion.value;

  useEffect(() => {
    const fetchDraftCount = () => {
      apiFetch("/api/draft-audit-logs?status=pending_review")
        .then((r) => r.json())
        .then((drafts: unknown[]) => {
          setDraftReviewCount(Array.isArray(drafts) ? drafts.length : 0);
        })
        .catch(() => setDraftReviewCount(0));
    };
    fetchDraftCount();
    const interval = setInterval(fetchDraftCount, 30000);
    return () => clearInterval(interval);
  }, [version]);

  return (
    <aside class="sidebar">
      <nav class="sidebar-nav">
        {NAV_ITEMS.map((item) => (
          <button
            key={item.view}
            class={`sidebar-item ${navItemActive(item.view, view) ? "active" : ""}`}
            onClick={() => navigate(item.view)}
          >
            <span class="sidebar-icon">
              <item.icon />
            </span>
            {item.label}
            {item.view === "reviews" && draftReviewCount > 0 && (
              <span class="inbox-badge">{draftReviewCount}</span>
            )}
          </button>
        ))}
      </nav>
      {currentUser.value?.role === "admin" && (
        <div class="sidebar-footer">
          <button
            class={`sidebar-settings-btn ${view === "settings" ? "active" : ""}`}
            onClick={() => navigate("settings")}
            title="Settings"
            aria-label="Settings"
          >
            <SettingsIcon />
          </button>
        </div>
      )}
    </aside>
  );
}
