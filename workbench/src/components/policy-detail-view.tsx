// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { selectedPolicyDetail, activeTab, navigate, updateHash } from "../app";
import { apiFetch } from "../api/fetch";
import { RequirementMatrixView } from "./requirement-matrix-view";
import { InventoryView } from "./inventory-view";
import { EvidenceView } from "./evidence-view";
import { AuditHistoryView } from "./audit-history-view";

interface PolicyInfo {
  policy_id: string;
  title: string;
  version?: string;
}

type TabId = "requirements" | "inventory" | "evidence" | "history";

const TABS: { id: TabId; label: string }[] = [
  { id: "requirements", label: "Requirements" },
  { id: "inventory", label: "Inventory" },
  { id: "evidence", label: "Evidence" },
  { id: "history", label: "History" },
];

export function PolicyDetailView() {
  const policyId = selectedPolicyDetail.value;
  const tab = activeTab.value;
  const [policy, setPolicy] = useState<PolicyInfo | null>(null);
  const [evidenceTargetFilter, setEvidenceTargetFilter] = useState<string | undefined>();
  const [evidenceControlFilter, setEvidenceControlFilter] = useState<string | undefined>();

  const handleTargetClick = (targetId: string, targetName: string) => {
    setEvidenceTargetFilter(targetName || targetId);
    setEvidenceControlFilter(undefined);
    switchTab("evidence");
  };

  const handleControlClick = (controlId: string) => {
    setEvidenceControlFilter(controlId);
    setEvidenceTargetFilter(undefined);
    switchTab("evidence");
  };

  useEffect(() => {
    if (!policyId) return;
    apiFetch(`/api/policies/${encodeURIComponent(policyId)}`)
      .then((r) => r.json())
      .then((d: { policy: PolicyInfo }) => setPolicy(d.policy))
      .catch(() => setPolicy({ policy_id: policyId, title: policyId }));
  }, [policyId]);

  if (!policyId) {
    navigate("posture");
    return null;
  }

  const switchTab = (t: TabId) => {
    activeTab.value = t;
    updateHash();
  };

  return (
    <section class="policy-detail-view" data-policy-id={policyId}>
      <nav class="breadcrumb" aria-label="Breadcrumb">
        <button class="breadcrumb-link" onClick={() => navigate("posture")}>Posture</button>
        <span class="breadcrumb-sep" aria-hidden="true">&rsaquo;</span>
        <span class="breadcrumb-current">{policy?.title || policyId}</span>
      </nav>

      <div class="tab-bar" role="tablist">
        {TABS.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={tab === t.id}
            class={`tab-btn ${tab === t.id ? "active" : ""}`}
            onClick={() => switchTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div class="tab-content" role="tabpanel">
        {tab === "requirements" && <RequirementMatrixView policyIdOverride={policyId} />}
        {tab === "inventory" && (
          <InventoryView
            policyIdOverride={policyId}
            onTargetClick={handleTargetClick}
            onControlClick={handleControlClick}
          />
        )}
        {tab === "evidence" && (
          <EvidenceView
            policyIdOverride={policyId}
            initialTargetFilter={evidenceTargetFilter}
            initialControlFilter={evidenceControlFilter}
          />
        )}
        {tab === "history" && <AuditHistoryView policyIdOverride={policyId} />}
      </div>
    </section>
  );
}
