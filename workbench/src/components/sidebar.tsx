// SPDX-License-Identifier: Apache-2.0
import { currentView, navigate } from "../app";
export function Sidebar() {
  const view = currentView.value;
  return (
    <aside class="sidebar">
      <nav class="sidebar-nav">
        <button class={`sidebar-item ${view === "missions" || view === "detail" ? "active" : ""}`} onClick={() => navigate("missions")}>
          <span class="sidebar-icon">&#9776;</span>Missions
        </button>
        <button class={`sidebar-item ${view === "registry" ? "active" : ""}`} onClick={() => navigate("registry")}>
          <span class="sidebar-icon">&#9881;</span>Registry
        </button>
      </nav>
    </aside>
  );
}
