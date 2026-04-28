// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { selectedPolicyId, selectedTimeRange, viewInvalidation, updateHash, navigateToAudit } from "../app";
import { apiFetch } from "../api/fetch";
import { cardKeyHandler } from "../lib/a11y";

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

export function AuditHistoryView({ policyIdOverride }: { policyIdOverride?: string } = {}) {
  const embedded = !!policyIdOverride;
  const [policies, setPolicies] = useState<PolicyOption[]>([]);
  const [policyId, setPolicyId] = useState(policyIdOverride || selectedPolicyId.value || "");
  const [auditIdFilter, setAuditIdFilter] = useState("");
  const [startFilter, setStartFilter] = useState("");
  const [endFilter, setEndFilter] = useState("");
  const [logs, setLogs] = useState<AuditLog[]>([]);
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

  useEffect(() => {
    if (selectedTimeRange.value) {
      if (selectedTimeRange.value.start && !startFilter) setStartFilter(selectedTimeRange.value.start);
      if (selectedTimeRange.value.end && !endFilter) setEndFilter(selectedTimeRange.value.end);
    }
  }, []);

  const fetchLogs = () => {
    if (policyId) selectedPolicyId.value = policyId;
    selectedTimeRange.value = (startFilter || endFilter) ? { start: startFilter, end: endFilter } : null;
    updateHash();
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
  useEffect(() => { if (policyId) fetchLogs(); }, [viewInvalidation.value]);

  const parseSummary = (s: string) => {
    try { return JSON.parse(s); } catch { return null; }
  };

  const auditIds = logs.map((l) => l.audit_id);

  return (
    <section class="audit-history-view">
      {!embedded && <h2>Audit History</h2>}

      <div class="audit-filters">
        {!embedded && (
          <select value={policyId} onChange={(e) => setPolicyId((e.target as HTMLSelectElement).value)}>
            <option value="">Select a policy...</option>
            {policies.map((p) => (
              <option key={p.policy_id} value={p.policy_id}>{p.title}</option>
            ))}
          </select>
        )}
        {embedded && logs.length > 0 ? (
          <select
            value={auditIdFilter}
            onChange={(e) => setAuditIdFilter((e.target as HTMLSelectElement).value)}
          >
            <option value="">All Audits</option>
            {auditIds.map((id) => (
              <option key={id} value={id}>{id}</option>
            ))}
          </select>
        ) : (
          <input
            placeholder="Audit ID"
            value={auditIdFilter}
            onInput={(e) => setAuditIdFilter((e.target as HTMLInputElement).value)}
          />
        )}
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
        <div class="audit-card-list">
          {logs.map((log) => {
            const summary = parseSummary(log.summary);
            return (
              <article
                key={log.audit_id}
                class="audit-card"
                onClick={() => navigateToAudit(log.audit_id)}
                role="button"
                tabIndex={0}
                onKeyDown={cardKeyHandler(() => navigateToAudit(log.audit_id))}
                aria-label={`View audit for ${new Date(log.audit_start).toLocaleDateString()} — ${new Date(log.audit_end).toLocaleDateString()}`}
              >
                <div class="audit-card-header">
                  <span class="audit-period">
                    {new Date(log.audit_start).toLocaleDateString()} — {new Date(log.audit_end).toLocaleDateString()}
                  </span>
                  {log.framework && <span class="audit-framework">{log.framework}</span>}
                </div>
                {summary && (
                  <div class="posture-counts">
                    <span class="count-pass">{summary.strengths ?? 0} strengths</span>
                    <span class="count-finding">{summary.findings ?? 0} findings</span>
                    <span class="count-gap">{summary.gaps ?? 0} gaps</span>
                  </div>
                )}
                <div class="audit-card-meta">
                  <span>{log.created_by || "system"}</span>
                  <span>{new Date(log.created_at).toLocaleDateString()}</span>
                </div>
              </article>
            );
          })}
        </div>
      )}
    </section>
  );
}
