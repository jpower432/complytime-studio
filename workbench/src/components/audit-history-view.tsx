// SPDX-License-Identifier: Apache-2.0

import { Fragment } from "preact";
import { useState, useEffect } from "preact/hooks";
import { selectedPolicyId, selectedTimeRange, viewInvalidation, updateHash } from "../app";
import { apiFetch } from "../api/fetch";
import { downloadYaml, auditLogFilename } from "../lib/download";

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

interface SummaryData {
  strengths: number;
  findings: number;
  gaps: number;
  observations: number;
}

function parseSummary(s: string): SummaryData | null {
  try {
    const parsed = JSON.parse(s);
    return {
      strengths: parsed.strengths ?? 0,
      findings: parsed.findings ?? 0,
      gaps: parsed.gaps ?? 0,
      observations: parsed.observations ?? 0,
    };
  } catch {
    return null;
  }
}

function deltaTooltip(delta: number): string {
  if (delta === 0) return "No change from prior audit";
  const abs = Math.abs(delta);
  return delta > 0 ? `${abs} more than prior audit` : `${abs} fewer than prior audit`;
}

function DeltaCell({ current, previous, warnOnIncrease }: {
  current: number;
  previous: number | null;
  warnOnIncrease: boolean;
}) {
  if (previous === null) return <>{current}</>;
  const delta = current - previous;
  const tip = deltaTooltip(delta);
  if (delta === 0) {
    return <>{current} <span class="delta-zero" title={tip}>(0)</span></>;
  }
  const isWarn = (delta > 0 && warnOnIncrease)
    || (delta < 0 && !warnOnIncrease);
  const cls = isWarn ? "delta-positive" : "delta-negative";
  const sign = delta > 0 ? "+" : "";
  return (
    <>{current} <span class={cls} title={tip}>({sign}{delta})</span></>
  );
}

export function AuditHistoryView({ policyIdOverride }: { policyIdOverride?: string } = {}) {
  const embedded = !!policyIdOverride;
  const [policies, setPolicies] = useState<PolicyOption[]>([]);
  const [policyId, setPolicyId] = useState(policyIdOverride || selectedPolicyId.value || "");
  const [auditIdFilter, setAuditIdFilter] = useState("");
  const [startFilter, setStartFilter] = useState("");
  const [endFilter, setEndFilter] = useState("");
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [expandedContent, setExpandedContent] = useState<string | null>(null);
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

  const toggleExpand = (log: AuditLog) => {
    if (expandedId === log.audit_id) {
      setExpandedId(null);
      setExpandedContent(null);
      return;
    }
    setExpandedId(log.audit_id);
    setExpandedContent(null);
    apiFetch(`/api/audit-logs/${encodeURIComponent(log.audit_id)}`)
      .then((r) => { if (!r.ok) throw new Error("not found"); return r.json(); })
      .then((full: AuditLog) => setExpandedContent(full.content || full.summary))
      .catch(() => setExpandedContent(log.summary));
  };

  const summaries = logs.map((l) => parseSummary(l.summary));
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
        <table class="history-table">
          <thead>
            <tr>
              <th>Period</th>
              <th>Framework</th>
              <th>Strengths</th>
              <th>Findings</th>
              <th>Gaps</th>
              <th>Author</th>
            </tr>
          </thead>
          <tbody>
            {logs.map((log, i) => {
              const summary = summaries[i];
              const prev = i + 1 < summaries.length ? summaries[i + 1] : null;
              const isExpanded = expandedId === log.audit_id;
              return (
                <Fragment key={log.audit_id}>
                  <tr
                    class={isExpanded ? "expanded" : ""}
                    onClick={() => toggleExpand(log)}
                    onKeyDown={(e: KeyboardEvent) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); toggleExpand(log); } }}
                    role="button"
                    tabIndex={0}
                    aria-expanded={isExpanded}
                    aria-label={`Audit ${new Date(log.audit_start).toLocaleDateString()} — ${new Date(log.audit_end).toLocaleDateString()}`}
                  >
                    <td>
                      {new Date(log.audit_start).toLocaleDateString()} — {new Date(log.audit_end).toLocaleDateString()}
                    </td>
                    <td>{log.framework || "—"}</td>
                    <td>
                      {summary ? (
                        <DeltaCell current={summary.strengths} previous={prev?.strengths ?? null} warnOnIncrease={false} />
                      ) : "—"}
                    </td>
                    <td>
                      {summary ? (
                        <DeltaCell current={summary.findings} previous={prev?.findings ?? null} warnOnIncrease={true} />
                      ) : "—"}
                    </td>
                    <td>
                      {summary ? (
                        <DeltaCell current={summary.gaps} previous={prev?.gaps ?? null} warnOnIncrease={true} />
                      ) : "—"}
                    </td>
                    <td>{log.created_by || "—"}</td>
                  </tr>
                  {isExpanded && (
                    <tr class="history-expand-row">
                      <td colSpan={6}>
                        <pre class="yaml-viewer">{expandedContent || "Loading..."}</pre>
                        {expandedContent && (
                          <button
                            class="btn btn-sm btn-secondary"
                            style={{ marginTop: "8px" }}
                            onClick={(e) => {
                              e.stopPropagation();
                              downloadYaml(expandedContent, auditLogFilename(log.policy_id, log.audit_start));
                            }}
                          >
                            Download YAML
                          </button>
                        )}
                      </td>
                    </tr>
                  )}
                </Fragment>
              );
            })}
          </tbody>
        </table>
      )}
    </section>
  );
}
