// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { selectedPolicyId } from "../app";
import { apiFetch } from "../api/fetch";

interface AuditLog {
  audit_id: string;
  policy_id: string;
  audit_start: string;
  audit_end: string;
  framework: string;
  created_at: string;
  created_by: string;
  summary: string;
  content?: string;
}

interface PolicyOption {
  policy_id: string;
  title: string;
}

export function AuditHistoryView() {
  const [policies, setPolicies] = useState<PolicyOption[]>([]);
  const [policyId, setPolicyId] = useState(selectedPolicyId.value || "");
  const [auditIdFilter, setAuditIdFilter] = useState("");
  const [startFilter, setStartFilter] = useState("");
  const [endFilter, setEndFilter] = useState("");
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  const [compareA, setCompareA] = useState<string | null>(null);
  const [compareB, setCompareB] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

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

  const fetchLogs = () => {
    if (auditIdFilter.trim()) {
      setLoading(true);
      apiFetch(`/api/audit-logs/${encodeURIComponent(auditIdFilter.trim())}`)
        .then((r) => { if (!r.ok) throw new Error("not found"); return r.json(); })
        .then((log: AuditLog) => setLogs([log]))
        .catch(() => setLogs([]))
        .finally(() => setLoading(false));
      return;
    }
    if (!policyId) return;
    setLoading(true);
    const params = new URLSearchParams({ policy_id: policyId });
    if (startFilter) params.set("start", startFilter);
    if (endFilter) params.set("end", endFilter);
    apiFetch(`/api/audit-logs?${params}`)
      .then((r) => r.json())
      .then(setLogs)
      .catch(() => setLogs([]))
      .finally(() => setLoading(false));
  };

  useEffect(fetchLogs, [policyId]);

  const parseSummary = (s: string) => {
    try { return JSON.parse(s); } catch { return null; }
  };

  const logA = compareA ? logs.find((l) => l.audit_id === compareA) : null;
  const logB = compareB ? logs.find((l) => l.audit_id === compareB) : null;

  return (
    <div class="audit-history-view">
      <h2>Audit History</h2>

      <div class="audit-filters">
        <select value={policyId} onChange={(e) => setPolicyId((e.target as HTMLSelectElement).value)}>
          <option value="">Select a policy...</option>
          {policies.map((p) => (
            <option key={p.policy_id} value={p.policy_id}>{p.title}</option>
          ))}
        </select>
        <input placeholder="Audit ID" value={auditIdFilter} onInput={(e) => setAuditIdFilter((e.target as HTMLInputElement).value)} />
        <input type="date" value={startFilter} onInput={(e) => setStartFilter((e.target as HTMLInputElement).value)} title="Start date" />
        <input type="date" value={endFilter} onInput={(e) => setEndFilter((e.target as HTMLInputElement).value)} title="End date" />
        <button class="btn btn-primary" onClick={fetchLogs}>Search</button>
      </div>

      {loading ? (
        <div class="view-loading">Loading audit history...</div>
      ) : logs.length === 0 ? (
        <div class="empty-state">
          <p>{policyId ? "No audit logs for this policy." : "Select a policy to view audit history."}</p>
        </div>
      ) : (
        <>
          <div class="audit-timeline">
            {logs.map((log) => {
              const summary = parseSummary(log.summary);
              return (
                <div key={log.audit_id} class="audit-card" onClick={() => setSelectedLog(log)}>
                  <div class="audit-card-header">
                    <span class="audit-period">
                      {new Date(log.audit_start).toLocaleDateString()} — {new Date(log.audit_end).toLocaleDateString()}
                    </span>
                    {log.framework && <span class="audit-framework">{log.framework}</span>}
                  </div>
                  {summary && (
                    <div class="posture-counts">
                      <span class="count-pass">{summary.strengths ?? 0}</span>
                      <span class="count-finding">{summary.findings ?? 0}</span>
                      <span class="count-gap">{summary.gaps ?? 0}</span>
                    </div>
                  )}
                  <div class="audit-card-actions">
                    <label>
                      <input type="radio" name="compareA" onChange={() => setCompareA(log.audit_id)} checked={compareA === log.audit_id} /> A
                    </label>
                    <label>
                      <input type="radio" name="compareB" onChange={() => setCompareB(log.audit_id)} checked={compareB === log.audit_id} /> B
                    </label>
                  </div>
                </div>
              );
            })}
          </div>

          {logA && logB && (
            <ComparisonView a={logA} b={logB} parseSummary={parseSummary} />
          )}

          {selectedLog && (
            <div class="audit-detail">
              <div class="detail-header">
                <h3>Audit Detail</h3>
                <button class="btn btn-sm" onClick={() => setSelectedLog(null)}>Close</button>
              </div>
              <pre class="yaml-viewer">{selectedLog.content || selectedLog.summary}</pre>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function ComparisonView({
  a, b, parseSummary,
}: {
  a: AuditLog;
  b: AuditLog;
  parseSummary: (s: string) => any;
}) {
  const sa = parseSummary(a.summary);
  const sb = parseSummary(b.summary);
  if (!sa || !sb) return null;

  const fields = ["strengths", "findings", "gaps", "observations"];

  return (
    <div class="comparison-view">
      <h3>Comparison</h3>
      <table class="data-table">
        <thead>
          <tr>
            <th>Metric</th>
            <th>Period A</th>
            <th>Period B</th>
            <th>Delta</th>
          </tr>
        </thead>
        <tbody>
          {fields.map((f) => {
            const va = sa[f] ?? 0;
            const vb = sb[f] ?? 0;
            const delta = vb - va;
            return (
              <tr key={f}>
                <td>{f}</td>
                <td>{va}</td>
                <td>{vb}</td>
                <td class={delta > 0 ? "delta-up" : delta < 0 ? "delta-down" : ""}>
                  {delta > 0 ? `+${delta}` : delta}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
