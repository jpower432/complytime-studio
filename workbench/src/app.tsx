// SPDX-License-Identifier: Apache-2.0

import { signal } from "@preact/signals";
import { useState, useEffect, useCallback } from "preact/hooks";
import "./store/theme";
import { Header } from "./components/header";
import { Sidebar } from "./components/sidebar";
import { DashboardView } from "./components/dashboard-view";
import { ProgramsView } from "./components/programs-view";
import { ProgramDetailView } from "./components/program-detail-view";
import { PoliciesView } from "./components/policies-view";
import { PolicyDetailView } from "./components/policy-detail-view";
import { InventoryView } from "./components/inventory-view";
import { EvidenceView } from "./components/evidence-view";
import { ReviewsView } from "./components/reviews-view";
import { SettingsView } from "./components/settings-view";
import { AuditWorkspaceView } from "./components/audit-workspace-view";
import { ChatAssistant } from "./components/chat-assistant";
import { fetchMe, redirectToLogin, type UserInfo } from "./api/auth";
import { apiFetch } from "./api/fetch";
import { registerNames } from "./lib/format";

export type View =
  | "dashboard"
  | "programs"
  | "program-detail"
  | "policies"
  | "inventory"
  | "evidence"
  | "reviews"
  | "review-workspace"
  | "settings";

export const currentView = signal<View>("dashboard");
export const currentUser = signal<UserInfo | null>(null);
export const authChecked = signal(false);

export const selectedPolicyId = signal<string | null>(null);
export const selectedTimeRange = signal<{ start: string; end: string } | null>(null);
export const selectedControlId = signal<string | null>(null);
export const selectedRequirementId = signal<string | null>(null);
export const selectedEvalResult = signal<string | null>(null);
export const selectedPolicyDetail = signal<string | null>(null);
export const selectedAuditId = signal<string | null>(null);
export const selectedProgramId = signal<string | null>(null);
export const selectedProgramFilter = signal<string | null>(null);
export const activeTab = signal<"requirements" | "history">("requirements");
export const selectedEvidenceTargetId = signal<string | null>(null);

// Monotonic counter; mounted views watch this to refetch. Same browser tab/session only —
// not shared across tabs or windows. Out-of-band backend changes (other clients, direct DB)
// require focus/navigation or manual refresh to pick up unless the user triggers invalidation
// from this session (e.g. artifact callback).
export const viewInvalidation = signal(0);
export function invalidateViews() {
  viewInvalidation.value++;
}

export const inboxVersion = signal(0);
export function invalidateInbox() {
  inboxVersion.value++;
}

const VALID_VIEWS: View[] = [
  "dashboard",
  "programs",
  "program-detail",
  "policies",
  "inventory",
  "evidence",
  "reviews",
  "review-workspace",
  "settings",
];

function parseHash(hash: string): { view: View; params: Record<string, string> } {
  const stripped = hash.replace(/^#\/?/, "");
  const qIdx = stripped.indexOf("?");
  const pathPart = qIdx >= 0 ? stripped.slice(0, qIdx) : stripped;
  const params: Record<string, string> = {};
  if (qIdx >= 0) {
    new URLSearchParams(stripped.slice(qIdx + 1)).forEach((v, k) => {
      params[k] = v;
    });
  }

  if (pathPart.startsWith("programs/")) {
    const programId = pathPart.slice("programs/".length);
    if (programId) {
      return {
        view: "program-detail",
        params: { ...params, programId: decodeURIComponent(programId) },
      };
    }
  }

  if (pathPart.startsWith("reviews/")) {
    const id = pathPart.slice("reviews/".length);
    if (id) {
      return {
        view: "review-workspace",
        params: { ...params, auditId: decodeURIComponent(id) },
      };
    }
  }

  // Legacy deep links
  if (pathPart.startsWith("audit/")) {
    const id = pathPart.slice("audit/".length);
    if (id) {
      return {
        view: "review-workspace",
        params: { ...params, auditId: decodeURIComponent(id) },
      };
    }
  }

  const view = VALID_VIEWS.includes(pathPart as View) ? (pathPart as View) : "dashboard";
  return { view, params };
}

function buildHash(view: View): string {
  if (view === "review-workspace" && selectedAuditId.value) {
    return `#/reviews/${encodeURIComponent(selectedAuditId.value)}`;
  }

  if (view === "program-detail" && selectedProgramId.value) {
    return `#/programs/${encodeURIComponent(selectedProgramId.value)}`;
  }

  const parts: string[] = [];
  const includePolicyFilters =
    view === "policies" || view === "evidence" || view === "inventory";

  if (includePolicyFilters) {
    if (selectedPolicyId.value) {
      parts.push(`policy=${encodeURIComponent(selectedPolicyId.value)}`);
    }
    if (selectedTimeRange.value?.start) {
      parts.push(`start=${encodeURIComponent(selectedTimeRange.value.start)}`);
    }
    if (selectedTimeRange.value?.end) {
      parts.push(`end=${encodeURIComponent(selectedTimeRange.value.end)}`);
    }
    if (selectedControlId.value) {
      parts.push(`control=${encodeURIComponent(selectedControlId.value)}`);
    }
    if (selectedRequirementId.value) {
      parts.push(`req=${encodeURIComponent(selectedRequirementId.value)}`);
    }
  }

  if (view === "policies" && activeTab.value && activeTab.value !== "requirements") {
    parts.push(`tab=${activeTab.value}`);
  }

  if (view === "evidence" && selectedEvidenceTargetId.value) {
    parts.push(`target=${encodeURIComponent(selectedEvidenceTargetId.value)}`);
  }

  if (
    (view === "evidence" || view === "policies")
    && selectedProgramFilter.value
  ) {
    parts.push(
      `program=${encodeURIComponent(selectedProgramFilter.value)}`,
    );
  }

  return parts.length ? `#/${view}?${parts.join("&")}` : `#/${view}`;
}

/** Views that may keep `selectedPolicyId` in the hash when navigating here via the shell. */
const VIEWS_THAT_KEEP_POLICY_FILTER: View[] = ["evidence", "inventory", "review-workspace"];

const VIEWS_THAT_KEEP_PROGRAM_FILTER: View[] = ["evidence", "policies"];

export function navigate(view: View) {
  if (view !== "evidence") {
    selectedEvidenceTargetId.value = null;
  }
  if (!VIEWS_THAT_KEEP_POLICY_FILTER.includes(view) && view !== "program-detail") {
    selectedPolicyId.value = null;
  }
  if (!VIEWS_THAT_KEEP_PROGRAM_FILTER.includes(view)) {
    selectedProgramFilter.value = null;
  }
  if (view !== "program-detail") {
    selectedProgramId.value = null;
  }
  if (view !== "review-workspace") {
    selectedAuditId.value = null;
  }
  currentView.value = view;
  const hash = buildHash(view);
  if (window.location.hash !== hash) {
    window.location.hash = hash;
  }
}

/** Opens a draft or published audit in the review workspace (canonical route: #/reviews/{id}). */
export function navigateToReview(id: string) {
  selectedAuditId.value = id;
  currentView.value = "review-workspace";
  const hash = buildHash("review-workspace");
  if (window.location.hash !== hash) {
    window.location.hash = hash;
  }
}

/** @deprecated Prefer navigateToReview; kept for call sites that still use the old name. */
export function navigateToAudit(id: string) {
  navigateToReview(id);
}

export function navigateToProgram(id: string) {
  selectedProgramId.value = id;
  currentView.value = "program-detail";
  const hash = buildHash("program-detail");
  if (window.location.hash !== hash) {
    window.location.hash = hash;
  }
}

export function navigateToPolicy(
  policyId: string,
  tab: "requirements" | "history" = "requirements",
) {
  selectedEvidenceTargetId.value = null;
  selectedPolicyDetail.value = null;
  selectedPolicyId.value = policyId;
  activeTab.value = tab;
  currentView.value = "policies";
  const hash = buildHash("policies");
  if (window.location.hash !== hash) {
    window.location.hash = hash;
  }
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

  if (params.programId) {
    selectedProgramId.value = params.programId;
  } else if (view !== "program-detail") {
    selectedProgramId.value = null;
  }

  if (params.auditId) {
    selectedAuditId.value = params.auditId;
  } else if (view !== "review-workspace") {
    selectedAuditId.value = null;
  }

  const VALID_TABS = ["requirements", "history"] as const;
  if (params.tab && VALID_TABS.includes(params.tab as (typeof VALID_TABS)[number])) {
    activeTab.value = params.tab as (typeof VALID_TABS)[number];
  }
  if (params.policy) {
    selectedPolicyId.value = params.policy;
  } else if (view === "policies") {
    selectedPolicyId.value = null;
  }

  if (view === "evidence" || view === "policies") {
    selectedProgramFilter.value = params.program ?? null;
  } else {
    selectedProgramFilter.value = null;
  }
  if (params.start || params.end) {
    selectedTimeRange.value = { start: params.start || "", end: params.end || "" };
  }
  if (params.control) {
    selectedControlId.value = params.control;
  }
  if (params.req) {
    selectedRequirementId.value = params.req;
  }

  if (view === "evidence") {
    selectedEvidenceTargetId.value = params.target ?? null;
  } else {
    selectedEvidenceTargetId.value = null;
  }
}

syncFromHash();

fetchMe().then((user) => {
  currentUser.value = user;
  authChecked.value = true;
  if (user?.email) {
    registerNames([{ email: user.email, name: user.name }]);
  }
});

function SetupBanner() {
  const [needsSetup, setNeedsSetup] = useState(false);
  const [claiming, setClaiming] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch("/api/setup-status")
      .then((r) => r.json())
      .then((d: { needs_setup: boolean }) => setNeedsSetup(d.needs_setup))
      .catch(() => {});
  }, []);

  const claim = useCallback(async () => {
    setClaiming(true);
    setError("");
    try {
      const res = await apiFetch("/api/bootstrap", { method: "POST" });
      if (res.ok) {
        const me = await fetchMe();
        currentUser.value = me;
        if (me?.email) {
          registerNames([{ email: me.email, name: me.name }]);
        }
        setNeedsSetup(false);
      } else {
        const body = await res.json().catch(() => ({ error: "Setup failed" }));
        setError(body.error || `Setup failed (${res.status})`);
      }
    } catch {
      setError("Network error — could not reach the server.");
    } finally {
      setClaiming(false);
    }
  }, []);

  if (!needsSetup) {
    return null;
  }

  return (
    <div class="setup-banner">
      <span>No admin configured. Complete initial setup to get started.</span>
      {error && (
        <span class="setup-banner-error" role="alert">
          {error}
        </span>
      )}
      <button class="btn btn-primary btn-sm" disabled={claiming} onClick={claim}>
        {claiming ? "Setting up..." : "Complete Setup"}
      </button>
    </div>
  );
}

export function App() {
  const view = currentView.value;
  const activeProgramId = selectedProgramId.value;
  const user = currentUser.value;
  const checked = authChecked.value;
  const [chatOpen, setChatOpen] = useState(false);

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
        <button class="btn btn-primary login-btn" onClick={redirectToLogin}>
          Login with Google
        </button>
      </div>
    );
  }

  return (
    <div class="app-shell">
      <a href="#main-content" class="skip-link">
        Skip to main content
      </a>
      <Header user={user} onImportSuccess={invalidateViews} chatOpen={chatOpen} onChatToggle={() => setChatOpen((v) => !v)} />
      <SetupBanner />
      <div class="app-body">
        <Sidebar />
        <main id="main-content" class="app-main" data-view={view} data-program={activeProgramId ?? ""}>
          {view === "dashboard" && <DashboardView />}
          {view === "programs" && <ProgramsView />}
          {view === "program-detail" && <ProgramDetailView />}
          {view === "policies" && (selectedPolicyId.value
            ? <PolicyDetailView />
            : <PoliciesView />
          )}
          {view === "inventory" && (
            <InventoryView policyIdOverride={selectedPolicyId.value ?? undefined} />
          )}
          {view === "evidence" && <EvidenceView />}
          {view === "reviews" && <ReviewsView />}
          {view === "review-workspace" && <AuditWorkspaceView />}
          {view === "settings" && <SettingsView />}
        </main>
      </div>
      <ChatAssistant open={chatOpen} onClose={() => setChatOpen(false)} />
    </div>
  );
}
