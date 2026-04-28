// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { selectedPolicyId, selectedTimeRange, selectedControlId, selectedRequirementId, viewInvalidation, updateHash } from "../app";
import { apiFetch } from "../api/fetch";
import {
  fetchRequirementMatrix,
  fetchRequirementEvidence,
  type RequirementRow,
  type RequirementEvidenceRow,
} from "../api/requirements";

interface PolicyOption {
  policy_id: string;
  title: string;
}

const CLASSIFICATIONS = ["", "Healthy", "Failing", "Wrong Source", "Wrong Method", "Unfit Evidence", "Stale", "No Evidence"];

function ClassificationBadge({ value }: { value: string | null | undefined }) {
  const label = value || "No Evidence";
  const cls = label.toLowerCase().replace(/\s+/g, "-");
  return <span class={`classification-badge ${cls}`} data-classification={label}>{label}</span>;
}

export function RequirementMatrixView({ policyIdOverride }: { policyIdOverride?: string } = {}) {
  const embedded = !!policyIdOverride;
  const [policies, setPolicies] = useState<PolicyOption[]>([]);
  const [policyId, setPolicyId] = useState(policyIdOverride || selectedPolicyId.value || "");
  const [startFilter, setStartFilter] = useState(selectedTimeRange.value?.start || "");
  const [endFilter, setEndFilter] = useState(selectedTimeRange.value?.end || "");
  const [classFilter, setClassFilter] = useState("");
  const [familyFilter, setFamilyFilter] = useState("");
  const [rows, setRows] = useState<RequirementRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [evidenceRows, setEvidenceRows] = useState<RequirementEvidenceRow[]>([]);
  const [evidenceLoading, setEvidenceLoading] = useState(false);
  const [evidenceError, setEvidenceError] = useState<string | null>(null);
  const [riskMap, setRiskMap] = useState<Record<string, string>>({});

  useEffect(() => {
    apiFetch("/api/policies")
      .then((r) => r.json())
      .then(setPolicies)
      .catch(() => setPolicies([]));
  }, []);

  useEffect(() => {
    if (selectedPolicyId.value && selectedPolicyId.value !== policyId) {
      setPolicyId(selectedPolicyId.value);
    }
  }, [selectedPolicyId.value]);

  useEffect(() => {
    if (selectedTimeRange.value) {
      if (selectedTimeRange.value.start) setStartFilter(selectedTimeRange.value.start);
      if (selectedTimeRange.value.end) setEndFilter(selectedTimeRange.value.end);
    }
  }, [selectedTimeRange.value]);

  const fetchMatrix = () => {
    if (!policyId) return;
    selectedTimeRange.value = (startFilter || endFilter) ? { start: startFilter, end: endFilter } : null;
    selectedControlId.value = familyFilter || null;
    updateHash();
    setLoading(true);
    setExpandedId(null);
    fetchRequirementMatrix({
      policy_id: policyId,
      audit_start: startFilter || undefined,
      audit_end: endFilter || undefined,
      classification: classFilter || undefined,
      control_family: familyFilter || undefined,
    })
      .then(setRows)
      .catch(() => setRows([]))
      .finally(() => setLoading(false));

    apiFetch(`/api/risks/severity?policy_id=${encodeURIComponent(policyId)}`)
      .then((r) => r.json())
      .then((rows: { control_id: string; max_severity: string }[]) => {
        const map: Record<string, string> = {};
        for (const r of rows) map[r.control_id] = r.max_severity;
        setRiskMap(map);
      })
      .catch(() => setRiskMap({}));
  };

  useEffect(fetchMatrix, [policyId]);
  useEffect(() => { if (policyId) fetchMatrix(); }, [viewInvalidation.value]);

  const toggleExpand = (reqId: string) => {
    if (expandedId === reqId) {
      setExpandedId(null);
      selectedRequirementId.value = null;
      return;
    }
    setExpandedId(reqId);
    selectedRequirementId.value = reqId;
    setEvidenceLoading(true);
    setEvidenceError(null);
    fetchRequirementEvidence(reqId, {
      policy_id: policyId,
      audit_start: startFilter || undefined,
      audit_end: endFilter || undefined,
    })
      .then(setEvidenceRows)
      .catch((e) => {
        setEvidenceError(e.message || "Failed to load evidence");
        setEvidenceRows([]);
      })
      .finally(() => setEvidenceLoading(false));
  };

  const families = [...new Set(rows.map((r) => r.control_id.split("-")[0]).filter(Boolean))];

  const triggerExport = async (format: string, pid: string, start: string, end: string) => {
    const qs = new URLSearchParams({ policy_id: pid });
    if (start) qs.set("audit_start", start);
    if (end) qs.set("audit_end", end);
    try {
      const res = await apiFetch(`/api/export/${format}?${qs}`);
      if (!res.ok) throw new Error(`Export failed: ${res.status}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = res.headers.get("content-disposition")?.match(/filename="(.+)"/)?.[1] || `export.${format}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      alert(`Export error: ${e}`);
    }
  };

  return (
    <section class="requirement-matrix-view">
      {!embedded && <h2>Requirement Matrix</h2>}

      <div class="audit-filters">
        {!embedded && (
          <select value={policyId} data-policy-id={policyId || ""} onChange={(e) => setPolicyId((e.target as HTMLSelectElement).value)}>
            <option value="">Select a policy...</option>
            {policies.map((p) => (
              <option key={p.policy_id} value={p.policy_id}>{p.title}</option>
            ))}
          </select>
        )}
        <select value={classFilter} onChange={(e) => setClassFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All classifications</option>
          {CLASSIFICATIONS.filter(Boolean).map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
        <select value={familyFilter} onChange={(e) => setFamilyFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All control families</option>
          {families.map((f) => (
            <option key={f} value={f}>{f}</option>
          ))}
        </select>
        <input type="date" value={startFilter} onInput={(e) => setStartFilter((e.target as HTMLInputElement).value)} title="Audit start" />
        <input type="date" value={endFilter} onInput={(e) => setEndFilter((e.target as HTMLInputElement).value)} title="Audit end" />
        <button class="btn btn-primary" onClick={fetchMatrix}>Search</button>
      </div>

      {policyId && rows.length > 0 && (
        <div class="export-toolbar">
          <button class="btn btn-xs" onClick={() => triggerExport("csv", policyId, startFilter, endFilter)}>Export CSV</button>
          <button class="btn btn-xs" disabled title="Coming soon">Export Excel</button>
          <button class="btn btn-xs" disabled title="Coming soon">Export PDF</button>
        </div>
      )}

      {loading ? (
        <div class="view-loading">Loading requirement matrix...</div>
      ) : !policyId ? (
        <div class="empty-state"><p>Select a policy to view requirement matrix.</p></div>
      ) : rows.length === 0 ? (
        <div class="empty-state"><p>No requirements found for this policy and filter combination.</p></div>
      ) : (
        <table class="data-table matrix-table">
          <thead>
            <tr>
              <th>Control</th>
              <th>Requirement</th>
              <th>Text</th>
              <th>Evidence</th>
              <th>Latest</th>
              <th>Classification</th>
              <th>Risk</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <>
                <tr
                  key={row.requirement_id}
                  class={`matrix-row ${expandedId === row.requirement_id ? "expanded" : ""}`}
                  data-requirement-id={row.requirement_id}
                  data-expanded={expandedId === row.requirement_id ? "true" : "false"}
                  aria-expanded={expandedId === row.requirement_id}
                  onClick={() => toggleExpand(row.requirement_id)}
                >
                  <td>
                    <span class="control-id">{row.control_id}</span>
                    {row.control_title && <span class="control-title">{row.control_title}</span>}
                  </td>
                  <td class="req-id">{row.requirement_id}</td>
                  <td class="req-text">{row.requirement_text}</td>
                  <td class="num">{row.evidence_count}</td>
                  <td class="date">{row.latest_evidence ? new Date(row.latest_evidence).toLocaleDateString() : "—"}</td>
                  <td><ClassificationBadge value={row.classification} /></td>
                  <td>{riskMap[row.control_id] ? (
                    <span class={`risk-badge risk-${riskMap[row.control_id].toLowerCase()}`}>{riskMap[row.control_id]}</span>
                  ) : "—"}</td>
                </tr>
                {expandedId === row.requirement_id && (
                  <tr class="evidence-expand-row">
                    <td colSpan={7}>
                      <EvidencePanel
                        rows={evidenceRows}
                        loading={evidenceLoading}
                        error={evidenceError}
                        onRetry={() => toggleExpand(row.requirement_id)}
                      />
                    </td>
                  </tr>
                )}
              </>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

function EvidencePanel({
  rows, loading, error, onRetry,
}: {
  rows: RequirementEvidenceRow[];
  loading: boolean;
  error: string | null;
  onRetry: () => void;
}) {
  if (loading) return <div class="view-loading">Loading evidence...</div>;
  if (error) {
    return (
      <div class="evidence-error">
        <span>{error}</span>
        <button class="btn btn-xs" onClick={onRetry}>Retry</button>
      </div>
    );
  }
  if (rows.length === 0) return <div class="empty-state"><p>No evidence found for this requirement.</p></div>;

  return (
    <table class="data-table evidence-sub-table">
      <thead>
        <tr>
          <th>Target</th>
          <th>Rule</th>
          <th>Result</th>
          <th>Collected</th>
          <th>Classification</th>
          <th>Registry</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((ev) => (
          <tr key={ev.evidence_id} data-evidence-id={ev.evidence_id}>
            <td>{ev.target_name || ev.target_id}</td>
            <td class="mono">{ev.rule_id}</td>
            <td><span class={`eval-result eval-${ev.eval_result.toLowerCase().replace(/\s+/g, "-")}`}>{ev.eval_result}</span></td>
            <td class="date">{new Date(ev.collected_at).toLocaleString()}</td>
            <td><ClassificationBadge value={ev.classification} /></td>
            <td>{ev.source_registry || "—"}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
