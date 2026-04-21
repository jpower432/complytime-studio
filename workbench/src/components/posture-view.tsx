// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { navigate, selectedPolicyId } from "../app";
import { apiFetch } from "../api/fetch";

interface PolicySummary {
  policy_id: string;
  title: string;
  version: string;
  imported_at: string;
}

interface AuditSummary {
  audit_id: string;
  policy_id: string;
  summary: string;
  created_at: string;
}

export function PostureView() {
  const [policies, setPolicies] = useState<PolicySummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch("/api/policies")
      .then((r) => r.json())
      .then(setPolicies)
      .catch(() => setPolicies([]))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div class="view-loading">Loading posture data...</div>;

  if (policies.length === 0) {
    return (
      <div class="empty-state">
        <h2>No Policies Imported</h2>
        <p>Import a policy from an OCI registry to get started.</p>
        <button class="btn btn-primary" onClick={() => navigate("policies")}>
          Go to Policies
        </button>
      </div>
    );
  }

  return (
    <div class="posture-view">
      <h2>Compliance Posture</h2>
      <div class="posture-grid">
        {policies.map((p) => (
          <PostureCard key={p.policy_id} policy={p} />
        ))}
      </div>
    </div>
  );
}

function PostureCard({ policy }: { policy: PolicySummary }) {
  const [audit, setAudit] = useState<AuditSummary | null>(null);

  useEffect(() => {
    apiFetch(`/api/audit-logs?policy_id=${policy.policy_id}`)
      .then((r) => r.json())
      .then((logs: AuditSummary[]) => {
        if (logs.length > 0) setAudit(logs[0]);
      })
      .catch(() => {});
  }, [policy.policy_id]);

  const summary = audit?.summary ? JSON.parse(audit.summary) : null;

  return (
    <div
      class="posture-card"
      onClick={() => {
        selectedPolicyId.value = policy.policy_id;
        navigate("audit-history");
      }}
    >
      <h3>{policy.title}</h3>
      <span class="posture-version">{policy.version || "latest"}</span>
      {summary ? (
        <div class="posture-counts">
          <span class="count-pass">{summary.strengths ?? 0} pass</span>
          <span class="count-finding">{summary.findings ?? 0} findings</span>
          <span class="count-gap">{summary.gaps ?? 0} gaps</span>
          <span class="count-observation">{summary.observations ?? 0} observations</span>
        </div>
      ) : (
        <p class="posture-no-audit">No audit data yet</p>
      )}
    </div>
  );
}
