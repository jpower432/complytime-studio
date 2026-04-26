// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { navigate, navigateToPolicy, selectedPolicyId, viewInvalidation } from "../app";
import { apiFetch } from "../api/fetch";

interface PostureRow {
  policy_id: string;
  title: string;
  version?: string;
  total_rows: number;
  passed_rows: number;
  failed_rows: number;
  other_rows: number;
  latest_at?: string;
  target_count?: number;
  control_count?: number;
  latest_evidence_at?: string;
  owner?: string;
}

interface RiskSeverityMap {
  [policyId: string]: string;
}

export function PostureView() {
  const [rows, setRows] = useState<PostureRow[]>([]);
  const [riskMap, setRiskMap] = useState<RiskSeverityMap>({});
  const [loading, setLoading] = useState(true);

  const fetchPosture = () => {
    apiFetch("/api/posture")
      .then((r) => r.json())
      .then((data: PostureRow[]) => {
        setRows(data);
        fetchRiskSeverity(data.map((r) => r.policy_id));
      })
      .catch(() => setRows([]))
      .finally(() => setLoading(false));
  };

  const fetchRiskSeverity = (policyIds: string[]) => {
    const promises = policyIds.map((pid) =>
      apiFetch(`/api/risks/severity?policy_id=${encodeURIComponent(pid)}`)
        .then((r) => r.json())
        .then((rows: { control_id: string; max_severity: string }[]) => {
          const severityOrder = ["Critical", "High", "Medium", "Low", "Informational"];
          let highest = "";
          for (const row of rows) {
            if (!highest || severityOrder.indexOf(row.max_severity) < severityOrder.indexOf(highest)) {
              highest = row.max_severity;
            }
          }
          return { pid, highest };
        })
        .catch(() => ({ pid, highest: "" }))
    );
    Promise.all(promises).then((results) => {
      const map: RiskSeverityMap = {};
      for (const { pid, highest } of results) {
        if (highest) map[pid] = highest;
      }
      setRiskMap(map);
    });
  };

  useEffect(fetchPosture, []);
  useEffect(fetchPosture, [viewInvalidation.value]);

  if (loading) return <div class="view-loading">Loading posture data...</div>;

  if (rows.length === 0) {
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
    <section class="posture-view">
      <h2>Compliance Posture</h2>
      <div class="posture-grid">
        {rows.map((row) => (
          <PostureCard key={row.policy_id} row={row} riskSeverity={riskMap[row.policy_id]} />
        ))}
      </div>
    </section>
  );
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 0) return "just now";
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

function PostureCard({ row, riskSeverity }: { row: PostureRow; riskSeverity?: string }) {
  const passRate = row.total_rows > 0
    ? Math.round((row.passed_rows / row.total_rows) * 100)
    : 0;

  return (
    <article class="posture-card" data-policy-id={row.policy_id}>
      <div class="posture-card-header">
        <h3>{row.title}</h3>
        <div class="posture-card-meta">
          {riskSeverity && (
            <span class={`risk-badge risk-${riskSeverity.toLowerCase()}`} data-severity={riskSeverity}>
              {riskSeverity}
            </span>
          )}
          <span class="posture-version">{row.version || "latest"}</span>
        </div>
      </div>

      <div class="posture-inventory">
        {(row.target_count ?? 0) > 0 && (
          <span class="inventory-stat" data-type="targets">{row.target_count} targets</span>
        )}
        {(row.control_count ?? 0) > 0 && (
          <span class="inventory-stat" data-type="controls">{row.control_count} controls</span>
        )}
        <span class="inventory-stat" data-type="owner">
          {row.owner ? `Owner: ${row.owner}` : "No owner"}
        </span>
      </div>

      {row.total_rows > 0 ? (
        <div class="posture-counts">
          <span class="count-pass">{row.passed_rows} passed</span>
          <span class="count-finding">{row.failed_rows} failed</span>
          <span class="count-gap">{row.other_rows} other</span>
          <span class="count-observation">{passRate}% pass rate</span>
        </div>
      ) : (
        <p class="posture-no-audit">No evidence yet</p>
      )}
      {row.latest_evidence_at ? (
        <p class="posture-latest" title={row.latest_evidence_at}>
          Last evidence: {relativeTime(row.latest_evidence_at)}
        </p>
      ) : row.latest_at ? (
        <p class="posture-latest">Latest: {new Date(row.latest_at).toLocaleDateString()}</p>
      ) : null}
      <button
        class="btn btn-sm btn-primary posture-drilldown-btn"
        onClick={() => navigateToPolicy(row.policy_id)}
      >
        View Details
      </button>
    </article>
  );
}
