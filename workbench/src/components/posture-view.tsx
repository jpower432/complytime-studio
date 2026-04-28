// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { navigate, navigateToPolicy, selectedPolicyId, selectedTimeRange, viewInvalidation } from "../app";
import { apiFetch } from "../api/fetch";
import { isStale, freshnessClass, relativeTime } from "../lib/freshness";
import { cardKeyHandler } from "../lib/a11y";

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

type PresetKey = "7d" | "30d" | "90d" | "all";

export function PostureView() {
  const [rows, setRows] = useState<PostureRow[]>([]);
  const [riskMap, setRiskMap] = useState<RiskSeverityMap>({});
  const [loading, setLoading] = useState(true);
  const [activePreset, setActivePreset] = useState<PresetKey>("all");

  const fetchPosture = () => {
    const params = new URLSearchParams();
    if (selectedTimeRange.value?.start) params.set("start", selectedTimeRange.value.start);
    if (selectedTimeRange.value?.end) params.set("end", selectedTimeRange.value.end);
    const qs = params.toString();
    apiFetch(`/api/posture${qs ? `?${qs}` : ""}`)
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

  const handlePreset = (key: PresetKey) => {
    setActivePreset(key);
    if (key === "all") {
      selectedTimeRange.value = null;
    } else {
      const days = key === "7d" ? 7 : key === "30d" ? 30 : 90;
      const end = new Date().toISOString().slice(0, 10);
      const start = new Date(Date.now() - days * 86_400_000).toISOString().slice(0, 10);
      selectedTimeRange.value = { start, end };
    }
    setLoading(true);
    fetchPosture();
  };

  return (
    <section class="posture-view">
      <h2>Compliance Posture</h2>
      <TimePresets active={activePreset} onSelect={handlePreset} />
      <PostureSummary rows={rows} />
      <div class="posture-grid">
        {rows.map((row) => (
          <PostureCard key={row.policy_id} row={row} riskSeverity={riskMap[row.policy_id]} />
        ))}
      </div>
    </section>
  );
}

function TimePresets({ active, onSelect }: { active: PresetKey; onSelect: (k: PresetKey) => void }) {
  const presets: { key: PresetKey; label: string }[] = [
    { key: "7d", label: "7d" },
    { key: "30d", label: "30d" },
    { key: "90d", label: "90d" },
    { key: "all", label: "All" },
  ];
  return (
    <div class="time-presets">
      {presets.map((p) => (
        <button
          key={p.key}
          class={`btn btn-sm ${active === p.key ? "btn-primary" : "btn-secondary"}`}
          onClick={() => onSelect(p.key)}
        >
          {p.label}
        </button>
      ))}
    </div>
  );
}

function PostureBar({ passed, failed, other, total }: { passed: number; failed: number; other: number; total: number }) {
  if (total === 0) return null;
  const pPct = (passed / total) * 100;
  const fPct = (failed / total) * 100;
  const oPct = (other / total) * 100;
  return (
    <div class="posture-bar" role="img" aria-label={`${passed} passed, ${failed} failed, ${other} other`}>
      {pPct > 0 && <div class="bar-pass" style={{ width: `${pPct}%` }} />}
      {fPct > 0 && <div class="bar-fail" style={{ width: `${fPct}%` }} />}
      {oPct > 0 && <div class="bar-other" style={{ width: `${oPct}%` }} />}
    </div>
  );
}

function PostureSummary({ rows }: { rows: PostureRow[] }) {
  if (rows.length === 0) return null;
  const totals = rows.reduce(
    (acc, r) => ({
      passed: acc.passed + r.passed_rows,
      failed: acc.failed + r.failed_rows,
      other: acc.other + r.other_rows,
      total: acc.total + r.total_rows,
      policies: acc.policies + 1,
      stale: acc.stale + (isStale(r.latest_evidence_at) ? 1 : 0),
    }),
    { passed: 0, failed: 0, other: 0, total: 0, policies: 0, stale: 0 }
  );
  const passRate = totals.total > 0 ? Math.round((totals.passed / totals.total) * 100) : 0;

  return (
    <div class="posture-summary">
      <span class="summary-stat">{totals.policies} policies</span>
      <span class="summary-stat">{passRate}% overall pass rate</span>
      <PostureBar passed={totals.passed} failed={totals.failed} other={totals.other} total={totals.total} />
      {totals.stale > 0 && (
        <span class="summary-stat summary-stale">{totals.stale} with stale evidence</span>
      )}
    </div>
  );
}

function readinessLevel(row: PostureRow, riskSeverity?: string): "green" | "yellow" | "red" {
  const severity = riskSeverity?.toLowerCase();
  if (severity === "critical" || severity === "high") return "red";
  if (!row.latest_evidence_at) return "red";
  const ageMs = Date.now() - new Date(row.latest_evidence_at).getTime();
  const days = ageMs / 86_400_000;
  if (days > 30) return "red";
  if (days > 7 || severity === "medium") return "yellow";
  return "green";
}

function PostureCard({ row, riskSeverity }: { row: PostureRow; riskSeverity?: string }) {
  const passRate = row.total_rows > 0
    ? Math.round((row.passed_rows / row.total_rows) * 100)
    : 0;
  const readiness = readinessLevel(row, riskSeverity);

  return (
    <article
      class={`posture-card ${freshnessClass(row.latest_evidence_at)}`}
      data-policy-id={row.policy_id}
      onClick={() => navigateToPolicy(row.policy_id)}
      role="button"
      tabIndex={0}
      onKeyDown={cardKeyHandler(() => navigateToPolicy(row.policy_id))}
      aria-label={`View details for ${row.title}`}
    >
      <div class="posture-card-header">
        <h3><span class={`readiness-dot readiness-${readiness}`} aria-label={`Readiness: ${readiness}`} />{row.title}</h3>
        <div class="posture-card-meta">
          {riskSeverity && (
            <span class={`risk-badge risk-${riskSeverity.toLowerCase()}`} data-severity={riskSeverity}>
              {riskSeverity}
            </span>
          )}
          <span class="posture-version">{row.version || "latest"}</span>
        </div>
      </div>

      {row.latest_evidence_at ? (
        <p class="posture-latest" title={row.latest_evidence_at}>
          Last evidence: {relativeTime(row.latest_evidence_at)}
        </p>
      ) : row.latest_at ? (
        <p class="posture-latest">Latest: {new Date(row.latest_at).toLocaleDateString()}</p>
      ) : (
        <p class="posture-latest posture-latest-missing">No evidence yet</p>
      )}

      {row.total_rows > 0 ? (
        <div class="posture-counts">
          <span class="count-pass">{row.passed_rows} passed</span>
          <span class="count-finding">{row.failed_rows} failed</span>
          <span class="count-gap">{row.other_rows} other</span>
          <span class="count-observation">{passRate}% pass rate</span>
        </div>
      ) : null}

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
    </article>
  );
}
