// SPDX-License-Identifier: Apache-2.0

import { signal, effect } from "@preact/signals";
import { Header } from "./components/header";
import { Sidebar } from "./components/sidebar";
import { MissionsView } from "./components/missions-view";
import { MissionDetail } from "./components/mission-detail";
import { RegistryBrowser } from "./components/registry-browser";
import { fetchMe, redirectToLogin, type UserInfo } from "./api/auth";

export type View = "missions" | "detail" | "registry";

export const currentView = signal<View>("missions");
export const currentMissionId = signal<string | null>(null);
export const currentUser = signal<UserInfo | null>(null);
export const authChecked = signal(false);

export function navigate(view: View, missionId?: string) {
  currentView.value = view;
  currentMissionId.value = missionId ?? null;
}

fetchMe().then((user) => {
  currentUser.value = user;
  authChecked.value = true;
  if (!user) redirectToLogin();
});

export function App() {
  const view = currentView.value;
  const user = currentUser.value;
  const checked = authChecked.value;

  if (!checked) {
    return <div class="app-loading">Loading...</div>;
  }

  if (!user) {
    return null;
  }

  return (
    <div class="app-shell">
      <Header user={user} />
      <div class="app-body">
        <Sidebar />
        <main class="app-main">
          {view === "missions" && <MissionsView />}
          {view === "detail" && currentMissionId.value && (
            <MissionDetail missionId={currentMissionId.value} />
          )}
          {view === "registry" && <RegistryBrowser />}
        </main>
      </div>
    </div>
  );
}
