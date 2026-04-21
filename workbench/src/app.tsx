// SPDX-License-Identifier: Apache-2.0

import { signal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import "./store/theme";
import { Header } from "./components/header";
import { Sidebar } from "./components/sidebar";
import { PostureView } from "./components/posture-view";
import { PoliciesView } from "./components/policies-view";
import { EvidenceView } from "./components/evidence-view";
import { AuditHistoryView } from "./components/audit-history-view";
import { ChatAssistant } from "./components/chat-assistant";
import { fetchMe, redirectToLogin, type UserInfo } from "./api/auth";

export type View = "posture" | "policies" | "evidence" | "audit-history";

export const currentView = signal<View>("posture");
export const currentUser = signal<UserInfo | null>(null);
export const authChecked = signal(false);

export const selectedPolicyId = signal<string | null>(null);
export const selectedTimeRange = signal<{ start: string; end: string } | null>(null);

function parseHash(hash: string): View {
  const stripped = hash.replace(/^#\/?/, "");
  const valid: View[] = ["posture", "policies", "evidence", "audit-history"];
  return valid.includes(stripped as View) ? (stripped as View) : "posture";
}

export function navigate(view: View) {
  currentView.value = view;
  const hash = `#/${view}`;
  if (window.location.hash !== hash) {
    window.location.hash = hash;
  }
}

function syncFromHash() {
  currentView.value = parseHash(window.location.hash);
}

syncFromHash();

fetchMe().then((user) => {
  currentUser.value = user;
  authChecked.value = true;
});

export function App() {
  const view = currentView.value;
  const user = currentUser.value;
  const checked = authChecked.value;

  useEffect(() => {
    const onHashChange = () => syncFromHash();
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  if (!checked) {
    return <div class="app-loading">Loading...</div>;
  }

  if (!user) {
    return (
      <div class="login-screen">
        <h1 class="login-title">ComplyTime Studio</h1>
        <p class="login-tagline">Audit Dashboard</p>
        <button class="btn btn-primary login-btn" onClick={redirectToLogin}>Login with Google</button>
      </div>
    );
  }

  return (
    <div class="app-shell">
      <Header user={user} />
      <div class="app-body">
        <Sidebar />
        <main class="app-main">
          {view === "posture" && <PostureView />}
          {view === "policies" && <PoliciesView />}
          {view === "evidence" && <EvidenceView />}
          {view === "audit-history" && <AuditHistoryView />}
        </main>
      </div>
      <ChatAssistant />
    </div>
  );
}
