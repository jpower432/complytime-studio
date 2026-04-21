// SPDX-License-Identifier: Apache-2.0

import { currentView, navigate } from "../app";

const PostureIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M3 12m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v6a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M9 8m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v10a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M15 4m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v14a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M4 20h14"/></svg>
);

const PoliciesIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 3l8 4.5l0 9l-8 4.5l-8 -4.5l0 -9l8 -4.5"/><path d="M12 12l8 -4.5"/><path d="M12 12l0 9"/><path d="M12 12l-8 -4.5"/></svg>
);

const EvidenceIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M10 10m-7 0a7 7 0 1 0 14 0a7 7 0 1 0 -14 0"/><path d="M21 21l-6 -6"/></svg>
);

const AuditIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M11.795 21h-1.795a2 2 0 0 1 -2 -2v-12a2 2 0 0 1 2 -2h8a2 2 0 0 1 2 2v4"/><path d="M18 18m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0"/><path d="M15 3v4"/><path d="M7 3v4"/><path d="M3 11h16"/><path d="M18 16.496v1.504l1 1"/></svg>
);

const NAV_ITEMS: { view: "posture" | "policies" | "evidence" | "audit-history"; icon: () => any; label: string }[] = [
  { view: "posture", icon: PostureIcon, label: "Posture" },
  { view: "policies", icon: PoliciesIcon, label: "Policies" },
  { view: "evidence", icon: EvidenceIcon, label: "Evidence" },
  { view: "audit-history", icon: AuditIcon, label: "Audit History" },
];

export function Sidebar() {
  const view = currentView.value;
  return (
    <aside class="sidebar">
      <nav class="sidebar-nav">
        {NAV_ITEMS.map((item) => (
          <button
            key={item.view}
            class={`sidebar-item ${view === item.view ? "active" : ""}`}
            onClick={() => navigate(item.view)}
          >
            <span class="sidebar-icon"><item.icon /></span>
            {item.label}
          </button>
        ))}
      </nav>
    </aside>
  );
}
