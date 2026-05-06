// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import {
  selectedPolicyId,
  activeTab,
  navigate,
  updateHash,
  selectedEvidenceTargetId,
} from "../app";
import { apiFetch } from "../api/fetch";
import { RequirementMatrixView } from "./requirement-matrix-view";
import { AuditHistoryView } from "./audit-history-view";
import { fmtDate } from "../lib/format";

interface PolicyInfo {
  policy_id: string;
  title: string;
  version?: string;
}

interface MappingDocument {
  mapping_id: string;
  source_catalog_id: string;
  target_catalog_id: string;
  framework: string;
  imported_at: string;
}

type TabId = "requirements" | "mappings" | "history";

const TABS: { id: TabId; label: string }[] = [
  { id: "requirements", label: "Requirements" },
  { id: "mappings", label: "Mappings" },
  { id: "history", label: "History" },
];

export function PolicyDetailView() {
  const policyId = selectedPolicyId.value;
  const tab = (activeTab.value as TabId) || "requirements";
  const [policy, setPolicy] = useState<PolicyInfo | null>(null);
  const [mappings, setMappings] = useState<MappingDocument[]>([]);

  useEffect(() => {
    if (!policyId) return;
    apiFetch(`/api/policies/${encodeURIComponent(policyId)}`)
      .then((r) => r.json())
      .then((d: { policy: PolicyInfo; mappings?: MappingDocument[] }) => {
        setPolicy(d.policy);
        setMappings(d.mappings || []);
      })
      .catch(() => {
        setPolicy({ policy_id: policyId, title: policyId });
        setMappings([]);
      });
  }, [policyId]);

  if (!policyId) {
    navigate("policies");
    return null;
  }

  const switchTab = (t: TabId) => {
    activeTab.value = t;
    updateHash();
  };

  const goBack = () => {
    activeTab.value = "requirements";
    navigate("policies");
  };

  const goToInventory = () => {
    selectedPolicyId.value = policyId;
    navigate("inventory");
  };

  const goToEvidence = () => {
    selectedPolicyId.value = policyId;
    selectedEvidenceTargetId.value = null;
    navigate("evidence");
  };

  return (
    <section class="policy-detail-view" data-policy-id={policyId}>
      <nav class="breadcrumb" aria-label="Breadcrumb">
        <button class="breadcrumb-link" onClick={goBack}>Policies</button>
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
        <span class="tab-bar-spacer" />
        <button class="btn btn-sm btn-secondary" onClick={goToInventory}>
          Inventory &rsaquo;
        </button>
        <button class="btn btn-sm btn-secondary" onClick={goToEvidence}>
          Evidence &rsaquo;
        </button>
      </div>

      <div class="tab-content" role="tabpanel">
        {tab === "requirements" && <RequirementMatrixView policyIdOverride={policyId} mode="adherence" />}
        {tab === "mappings" && <MappingsPanel mappings={mappings} />}
        {tab === "history" && <AuditHistoryView policyIdOverride={policyId} />}
      </div>
    </section>
  );
}

function MappingsPanel({ mappings }: { mappings: MappingDocument[] }) {
  if (mappings.length === 0) {
    return (
      <div class="empty-state">
        <p>No mapping documents loaded for this policy.</p>
      </div>
    );
  }

  return (
    <table class="data-table">
      <thead>
        <tr>
          <th>Framework</th>
          <th>Source Catalog</th>
          <th>Target Catalog</th>
          <th>Imported</th>
        </tr>
      </thead>
      <tbody>
        {mappings.map((m) => (
          <tr key={m.mapping_id}>
            <td>{m.framework || "—"}</td>
            <td class="mono">{m.source_catalog_id}</td>
            <td class="mono">{m.target_catalog_id}</td>
            <td>{fmtDate(m.imported_at)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
