// SPDX-License-Identifier: Apache-2.0
import { currentView, navigate } from "../app";
export function Sidebar() {
  const view = currentView.value;
  return (
    <aside class="sidebar">
      <nav class="sidebar-nav">
        <button class={`sidebar-item ${view === "workspace" ? "active" : ""}`} onClick={() => navigate("workspace")}>
          <span class="sidebar-icon">&#9998;</span>Workspace
        </button>
        <button class={`sidebar-item ${view === "jobs" ? "active" : ""}`} onClick={() => navigate("jobs")}>
          <span class="sidebar-icon">&#9776;</span>Jobs
        </button>
      </nav>
    </aside>
  );
}
