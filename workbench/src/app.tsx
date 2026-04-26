// SPDX-License-Identifier: Apache-2.0

import { signal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import "./store/theme";
import { Header } from "./components/header";
import { Sidebar } from "./components/sidebar";
import { PostureView } from "./components/posture-view";
import { PoliciesView } from "./components/policies-view";
import { EvidenceView } from "./components/evidence-view";
import { PolicyDetailView } from "./components/policy-detail-view";
import { InboxView } from "./components/inbox-view";
import { ChatAssistant } from "./components/chat-assistant";
import { fetchMe, redirectToLogin, type UserInfo } from "./api/auth";

export type View = "posture" | "posture-detail" | "policies" | "evidence" | "inbox";

export const currentView = signal<View>("posture");
export const currentUser = signal<UserInfo | null>(null);
export const authChecked = signal(false);

export const selectedPolicyId = signal<string | null>(null);
export const selectedTimeRange = signal<{ start: string; end: string } | null>(null);
export const selectedControlId = signal<string | null>(null);
export const selectedRequirementId = signal<string | null>(null);
export const selectedEvalResult = signal<string | null>(null);
export const selectedPolicyDetail = signal<string | null>(null);
export const activeTab = signal<"requirements" | "evidence" | "history">("requirements");

// Monotonic counter; mounted views watch this to refetch. Same browser tab/session only —
// not shared across tabs or windows. Out-of-band backend changes (other clients, direct DB)
// require focus/navigation or manual refresh to pick up unless the user triggers invalidation
// from this session (e.g. artifact callback).
export const viewInvalidation = signal(0);
export function invalidateViews() { viewInvalidation.value++; }

const VALID_VIEWS: View[] = ["posture", "posture-detail", "policies", "evidence", "inbox"];

function parseHash(hash: string): { view: View; params: Record<string, string> } {
  const stripped = hash.replace(/^#\/?/, "");
  const qIdx = stripped.indexOf("?");
  const pathPart = qIdx >= 0 ? stripped.slice(0, qIdx) : stripped;
  const params: Record<string, string> = {};
  if (qIdx >= 0) {
    new URLSearchParams(stripped.slice(qIdx + 1)).forEach((v, k) => { params[k] = v; });
  }

  // Nested posture route: posture/{policy_id}
  if (pathPart.startsWith("posture/")) {
    const policyId = pathPart.slice("posture/".length);
    if (policyId) return { view: "posture-detail", params: { ...params, policyDetail: policyId } };
  }

  const view = VALID_VIEWS.includes(pathPart as View) ? (pathPart as View) : "posture";
  return { view, params };
}

function buildHash(view: View): string {
  const parts: string[] = [];

  if (view === "posture-detail" && selectedPolicyDetail.value) {
    const base = `posture/${encodeURIComponent(selectedPolicyDetail.value)}`;
    if (activeTab.value && activeTab.value !== "requirements") parts.push(`tab=${activeTab.value}`);
    if (selectedTimeRange.value?.start) parts.push(`start=${encodeURIComponent(selectedTimeRange.value.start)}`);
    if (selectedTimeRange.value?.end) parts.push(`end=${encodeURIComponent(selectedTimeRange.value.end)}`);
    if (selectedControlId.value) parts.push(`control=${encodeURIComponent(selectedControlId.value)}`);
    return parts.length ? `#/${base}?${parts.join("&")}` : `#/${base}`;
  }

  if (selectedPolicyId.value) parts.push(`policy=${encodeURIComponent(selectedPolicyId.value)}`);
  if (selectedTimeRange.value?.start) parts.push(`start=${encodeURIComponent(selectedTimeRange.value.start)}`);
  if (selectedTimeRange.value?.end) parts.push(`end=${encodeURIComponent(selectedTimeRange.value.end)}`);
  if (selectedControlId.value) parts.push(`control=${encodeURIComponent(selectedControlId.value)}`);
  if (selectedRequirementId.value) parts.push(`req=${encodeURIComponent(selectedRequirementId.value)}`);
  return parts.length ? `#/${view}?${parts.join("&")}` : `#/${view}`;
}

export function navigate(view: View) {
  currentView.value = view;
  const hash = buildHash(view);
  if (window.location.hash !== hash) {
    window.location.hash = hash;
  }
}

export function navigateToPolicy(policyId: string, tab: "requirements" | "evidence" | "history" = "requirements") {
  selectedPolicyDetail.value = policyId;
  selectedPolicyId.value = policyId;
  activeTab.value = tab;
  currentView.value = "posture-detail";
  const hash = buildHash("posture-detail");
  if (window.location.hash !== hash) window.location.hash = hash;
}

export function updateHash() {
  const hash = buildHash(currentView.value);
  if (window.location.hash !== hash) {
    history.replaceState(null, "", hash);
  }
}

function syncFromHash() {
  const { view, params } = parseHash(window.location.hash);
  currentView.value = view;
  if (params.policyDetail) {
    selectedPolicyDetail.value = params.policyDetail;
    selectedPolicyId.value = params.policyDetail;
  }
  if (params.tab) activeTab.value = params.tab as "requirements" | "evidence" | "history";
  if (params.policy) selectedPolicyId.value = params.policy;
  if (params.start || params.end) {
    selectedTimeRange.value = { start: params.start || "", end: params.end || "" };
  }
  if (params.control) selectedControlId.value = params.control;
  if (params.req) selectedRequirementId.value = params.req;
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
      <a href="#main-content" class="skip-link">Skip to main content</a>
      <Header user={user} />
      <div class="app-body">
        <Sidebar />
        <main id="main-content" class="app-main" data-view={view}>
          {view === "posture" && <PostureView />}
          {view === "posture-detail" && <PolicyDetailView />}
          {view === "policies" && <PoliciesView />}
          {view === "evidence" && <EvidenceView />}
          {view === "inbox" && <InboxView />}
        </main>
      </div>
      <ChatAssistant />
    </div>
  );
}
