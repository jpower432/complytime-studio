// SPDX-License-Identifier: Apache-2.0

import { signal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import "./store/theme";
import { purgeHistory } from "./store/jobs";
import { Header } from "./components/header";
import { Sidebar } from "./components/sidebar";
import { JobsView } from "./components/jobs-view";
import { WorkspaceView } from "./components/workspace-view";
import { fetchMe, redirectToLogin, type UserInfo } from "./api/auth";

export type View = "workspace" | "jobs";

export const currentView = signal<View>("workspace");
export const currentJobId = signal<string | null>(null);
export const currentUser = signal<UserInfo | null>(null);
export const authChecked = signal(false);

export function navigate(view: View, jobId?: string) {
  currentView.value = view;
  currentJobId.value = jobId ?? null;
}

fetchMe().then((user) => {
  currentUser.value = user;
  authChecked.value = true;
});

purgeHistory();

const PURGE_INTERVAL_MS = 60 * 60 * 1000;

export function App() {
  const view = currentView.value;
  const user = currentUser.value;
  const checked = authChecked.value;

  useEffect(() => {
    const id = setInterval(purgeHistory, PURGE_INTERVAL_MS);
    return () => clearInterval(id);
  }, []);

  if (!checked) {
    return <div class="app-loading">Loading...</div>;
  }

  if (!user) {
    return (
      <div class="login-screen">
        <h1 class="login-title">ComplyTime Studio</h1>
        <p class="login-tagline">Gemara Artifact Workbench</p>
        <button class="btn btn-primary login-btn" onClick={redirectToLogin}>Login with GitHub</button>
      </div>
    );
  }

  return (
    <div class="app-shell">
      <Header user={user} />
      <div class="app-body">
        <Sidebar />
        <main class="app-main">
          {view === "workspace" && <WorkspaceView />}
          {view === "jobs" && <JobsView />}
        </main>
      </div>
    </div>
  );
}
