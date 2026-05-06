// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useRef, useCallback } from "preact/hooks";
import { selectedAuditId, selectedPolicyId, selectedEvidenceTargetId, navigate, navigateToPolicy } from "../app";
import { apiFetch } from "../api/fetch";
import { downloadYaml, auditLogFilename } from "../lib/download";
import { fmtDate, fmtDateTime, displayName } from "../lib/format";
import { fetchRequirementMatrix, fetchRequirementEvidence, type RequirementRow, type RequirementEvidenceRow } from "../api/requirements";

interface AuditArtifact {
  id: string;
  policy_id: string;
  audit_start: string;
  audit_end: string;
  framework: string;
  created_at: string;
  created_by?: string;
  summary: string;
  content?: string;
  agent_reasoning?: string;
  model?: string;
  prompt_version?: string;
  status?: string;
  reviewed_by?: string;
  reviewer_edits?: string;
}

interface AuditResult {
  id: string;
  title: string;
  type: string;
  description: string;
  "agent-reasoning"?: string;
}

interface EditEntry {
  type_override: string;
  note: string;
}

type EditsMap = Record<string, EditEntry>;
type SaveState = "idle" | "saving" | "saved";
const RESULT_TYPES = ["Strength", "Finding", "Gap", "Observation"];

function extractBlockScalar(lines: string[], key: string): string {
  const idx = lines.findIndex((l) => l.includes(`${key}:`));
  if (idx < 0) return "";
  const firstLine = lines[idx].replace(new RegExp(`.*${key}:\\s*>?-?\\s*`), "").trim();
  const parts: string[] = firstLine ? [firstLine] : [];
  const baseIndent = lines[idx].search(/\S/);
  for (let i = idx + 1; i < lines.length; i++) {
    const line = lines[i];
    if (line.trim() === "") continue;
    if (line.search(/\S/) <= baseIndent) break;
    parts.push(line.trim());
  }
  return parts.join(" ");
}

function parseYAMLContent(content: string): AuditResult[] | null {
  try {
    if (!content.match(/^results:\s*$/m)) return null;
    const results: AuditResult[] = [];
    const blocks = content.split(/^  - id:\s*/m).slice(1);
    for (const block of blocks) {
      const lines = block.split("\n");
      const id = lines[0]?.trim() || "";
      const title = lines.find((l) => l.includes("title:"))?.replace(/.*title:\s*/, "").trim() || "";
      const type = lines.find((l) => l.includes("type:") && !l.includes("evidence"))?.replace(/.*type:\s*/, "").trim() || "";
      const desc = extractBlockScalar(lines, "description");
      const reasoning = extractBlockScalar(lines, "agent-reasoning");
      results.push({ id, title, type, description: desc, "agent-reasoning": reasoning });
    }
    return results;
  } catch {
    return null;
  }
}

function parseEdits(raw?: string): EditsMap {
  if (!raw || raw === "{}") return {};
  try { return JSON.parse(raw); } catch { return {}; }
}

function ResultTypeTag({ type }: { type: string }) {
  const colors: Record<string, string> = {
    Strength: "var(--color-pass)",
    Finding: "var(--color-finding)",
    Gap: "var(--color-gap)",
    Observation: "var(--color-observation)",
  };
  return (
    <span class="result-type-tag" style={{ background: colors[type] || "#94a3b8", color: "#fff", padding: "2px 8px", borderRadius: "4px", fontSize: "0.75rem", fontWeight: 600 }}>
      {type}
    </span>
  );
}

export function AuditWorkspaceView() {
  const auditId = selectedAuditId.value;
  const [artifact, setArtifact] = useState<AuditArtifact | null>(null);
  const [mode, setMode] = useState<"draft" | "readonly">("readonly");
  const [results, setResults] = useState<AuditResult[] | null>(null);
  const [edits, setEdits] = useState<EditsMap>({});
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [promoting, setPromoting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [reqTextMap, setReqTextMap] = useState<Map<string, string>>(new Map());
  const [expandedEvidence, setExpandedEvidence] = useState<string | null>(null);
  const [evidenceRows, setEvidenceRows] = useState<RequirementEvidenceRow[]>([]);
  const [evidenceLoading, setEvidenceLoading] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const fadeRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!auditId) return;
    setLoading(true);
    apiFetch(`/api/draft-audit-logs/${encodeURIComponent(auditId)}`)
      .then((r) => {
        if (!r.ok) throw new Error("not a draft");
        return r.json();
      })
      .then((d: AuditArtifact) => {
        setArtifact({ ...d, id: auditId });
        setMode(d.status === "pending_review" ? "draft" : "readonly");
        if (d.content) setResults(parseYAMLContent(d.content));
        setEdits(parseEdits(d.reviewer_edits));
      })
      .catch(() => {
        apiFetch(`/api/audit-logs/${encodeURIComponent(auditId)}`)
          .then((r) => { if (!r.ok) throw new Error("not found"); return r.json(); })
          .then((log: AuditArtifact) => {
            setArtifact({ ...log, id: auditId });
            setMode("readonly");
            if (log.content) setResults(parseYAMLContent(log.content));
          })
          .catch(() => setArtifact(null));
      })
      .finally(() => setLoading(false));
  }, [auditId]);

  useEffect(() => {
    if (!artifact?.policy_id) return;
    fetchRequirementMatrix({ policy_id: artifact.policy_id })
      .then((rows: RequirementRow[]) => {
        const m = new Map<string, string>();
        for (const r of rows) m.set(r.control_id, r.requirement_text);
        setReqTextMap(m);
      })
      .catch(() => setReqTextMap(new Map()));
  }, [artifact?.policy_id]);

  const toggleEvidence = (resultId: string) => {
    if (expandedEvidence === resultId) {
      setExpandedEvidence(null);
      return;
    }
    setExpandedEvidence(resultId);
    if (!artifact) return;
    setEvidenceLoading(true);
    fetchRequirementEvidence(resultId, { policy_id: artifact.policy_id })
      .then(setEvidenceRows)
      .catch(() => setEvidenceRows([]))
      .finally(() => setEvidenceLoading(false));
  };

  const saveEditsToServer = useCallback((draftId: string, editsMap: EditsMap) => {
    setSaveState("saving");
    apiFetch(`/api/draft-audit-logs/${encodeURIComponent(draftId)}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reviewer_edits: editsMap }),
    }).then((r) => {
      if (!r.ok) throw new Error("save failed");
      setSaveState("saved");
      if (fadeRef.current) clearTimeout(fadeRef.current);
      fadeRef.current = setTimeout(() => setSaveState("idle"), 2000);
    }).catch(() => setSaveState("idle"));
  }, []);

  const scheduleAutoSave = useCallback((draftId: string, editsMap: EditsMap) => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => saveEditsToServer(draftId, editsMap), 1000);
  }, [saveEditsToServer]);

  const updateEdit = (resultId: string, field: "type_override" | "note", value: string) => {
    if (!artifact) return;
    const next = { ...edits, [resultId]: { ...edits[resultId] || { type_override: "", note: "" }, [field]: value } };
    setEdits(next);
    scheduleAutoSave(artifact.id, next);
  };

  const promote = () => {
    if (!artifact) return;
    setPromoting(true);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    const finalize = () => {
      apiFetch("/api/audit-logs/promote", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ draft_id: artifact.id }),
      }).then((r) => {
        if (!r.ok) throw new Error("promote failed");
        setMode("readonly");
        setArtifact((prev) => prev ? { ...prev, status: "promoted" } : prev);
      }).catch(() => alert("Failed to promote draft"))
        .finally(() => setPromoting(false));
    };
    if (Object.keys(edits).length > 0) {
      apiFetch(`/api/draft-audit-logs/${encodeURIComponent(artifact.id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reviewer_edits: edits }),
      }).then(finalize).catch(finalize);
    } else {
      finalize();
    }
  };

  const editFor = (id: string): EditEntry => edits[id] || { type_override: "", note: "" };
  const editable = mode === "draft";

  const reasoningMap: Record<string, string> = (() => {
    if (!artifact?.agent_reasoning) return {};
    try {
      const parsed = JSON.parse(artifact.agent_reasoning);
      if (typeof parsed === "object" && !Array.isArray(parsed)) return parsed;
    } catch { /* fall through */ }
    const map: Record<string, string> = {};
    for (const line of artifact.agent_reasoning.split("\n")) {
      const idx = line.indexOf(": ");
      if (idx > 0) map[line.slice(0, idx).trim()] = line.slice(idx + 2).trim();
    }
    return map;
  })();

  if (!auditId) {
    navigate("reviews");
    return null;
  }

  if (loading) return <div class="view-loading">Loading audit workspace...</div>;
  if (!artifact) return <div class="empty-state"><p>Audit not found.</p></div>;

  const summaryData = (() => {
    try { return JSON.parse(artifact.summary); } catch { return null; }
  })();

  return (
    <section class="audit-workspace">
      <nav class="breadcrumb" aria-label="Breadcrumb">
        <button class="breadcrumb-link" onClick={() => navigate("reviews")}>Reviews</button>
        <span class="breadcrumb-sep" aria-hidden="true">&rsaquo;</span>
        <button class="breadcrumb-link" onClick={() => navigateToPolicy(artifact.policy_id)}>{artifact.policy_id}</button>
        <span class="breadcrumb-sep" aria-hidden="true">&rsaquo;</span>
        <span class="breadcrumb-current">Audit</span>
      </nav>

      <header class="workspace-toolbar">
        <div class="workspace-meta">
          <span><strong>{artifact.policy_id}</strong></span>
          <span>{fmtDate(artifact.audit_start)} — {fmtDate(artifact.audit_end)}</span>
          {artifact.framework && <span>{artifact.framework}</span>}
          {artifact.model && <span class="meta-dim">Model: {artifact.model}</span>}
          <span class={`workspace-mode-badge mode-${mode}`}>{mode === "draft" ? "Draft" : "Promoted"}</span>
        </div>
        <div class="workspace-actions">
          {saveState !== "idle" && (
            <span class={`save-indicator save-${saveState}`}>{saveState === "saving" ? "Saving..." : "Saved"}</span>
          )}
          {artifact.content && (
            <button class="btn btn-sm btn-secondary" onClick={() => downloadYaml(artifact.content!, auditLogFilename(artifact.policy_id, artifact.audit_start))}>
              Download YAML
            </button>
          )}
          {editable && (
            <button class="btn btn-primary" onClick={promote} disabled={promoting}>
              {promoting ? "Saving..." : "Save to History"}
            </button>
          )}
        </div>
      </header>

      <div class="workspace-panels">
        <div class="workspace-left">
          <h3>Results</h3>
          {results ? (
            <div class="results-grid">
              {results.map((result) => {
                const edit = editFor(result.id);
                const displayType = edit.type_override || result.type;
                const overridden = edit.type_override !== "" && edit.type_override !== result.type;
                return (
                  <article key={result.id} class={`result-card type-${displayType} ${overridden ? "result-overridden" : ""}`}>
                    <div class="result-card-header">
                      <span class="result-id">{result.id}</span>
                      {editable ? (
                        <select class="result-type-select" value={displayType} onChange={(e) => updateEdit(result.id, "type_override", (e.target as HTMLSelectElement).value)}>
                          {RESULT_TYPES.map((t) => <option key={t} value={t}>{t}{t === result.type ? " (agent)" : ""}</option>)}
                        </select>
                      ) : (
                        <ResultTypeTag type={displayType} />
                      )}
                    </div>
                    <h4>{result.title}</h4>
                    {reqTextMap.get(result.id) && (
                      <p class="result-requirement-text"><strong>Requirement:</strong> {reqTextMap.get(result.id)}</p>
                    )}
                    <p class="result-description">{result.description}</p>
                    {(reasoningMap[result.id] || result["agent-reasoning"]) && (
                      <div class="agent-reasoning"><strong>Agent Reasoning:</strong><p>{reasoningMap[result.id] || result["agent-reasoning"]}</p></div>
                    )}
                    <button
                      type="button"
                      class="btn btn-xs btn-secondary"
                      onClick={(e) => { e.stopPropagation(); toggleEvidence(result.id); }}
                    >
                      {expandedEvidence === result.id ? "Hide Evidence" : "Show Evidence"}
                    </button>
                    {expandedEvidence === result.id && (
                      <div class="result-evidence-panel">
                        {evidenceLoading ? (
                          <p class="evidence-detail-muted">Loading evidence...</p>
                        ) : evidenceRows.length === 0 ? (
                          <p class="evidence-detail-muted">No evidence found for this requirement.</p>
                        ) : (
                          <table class="data-table evidence-sub-table">
                            <thead>
                              <tr><th>Target</th><th>Rule</th><th>Result</th><th>Collected</th></tr>
                            </thead>
                            <tbody>
                              {evidenceRows.map((ev) => (
                                <tr key={ev.evidence_id}>
                                  <td>{ev.target_name || ev.target_id}</td>
                                  <td class="mono">{ev.rule_id}</td>
                                  <td><span class={`eval-result eval-${ev.eval_result.toLowerCase().replace(/\s+/g, "-")}`}>{ev.eval_result}</span></td>
                                  <td class="date">{fmtDateTime(ev.collected_at)}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        )}
                      </div>
                    )}
                    {editable && (
                      <div class="result-controls">
                        {overridden && <span class="override-badge">Overridden</span>}
                        <textarea
                          class="result-note"
                          placeholder="Add reviewer note..."
                          value={edit.note}
                          onInput={(e) => updateEdit(result.id, "note", (e.target as HTMLTextAreaElement).value)}
                          rows={2}
                        />
                      </div>
                    )}
                  </article>
                );
              })}
            </div>
          ) : (
            <pre class="yaml-viewer">{artifact.content || artifact.summary}</pre>
          )}

          {artifact.agent_reasoning && (
            <details class="reasoning-section">
              <summary>Full Agent Reasoning</summary>
              <pre class="yaml-viewer">{artifact.agent_reasoning}</pre>
            </details>
          )}
        </div>

        <aside class="workspace-right">
          <h3>Audit Summary</h3>
          {summaryData && (
            <div class="workspace-summary-stats">
              <div class="summary-stat-card stat-pass"><span class="stat-value">{summaryData.strengths ?? 0}</span><span class="stat-label">Strengths</span></div>
              <div class="summary-stat-card stat-finding"><span class="stat-value">{summaryData.findings ?? 0}</span><span class="stat-label">Findings</span></div>
              <div class="summary-stat-card stat-gap"><span class="stat-value">{summaryData.gaps ?? 0}</span><span class="stat-label">Gaps</span></div>
            </div>
          )}
          <dl class="workspace-detail-list">
            <dt>Period</dt>
            <dd>{fmtDate(artifact.audit_start)} — {fmtDate(artifact.audit_end)}</dd>
            {artifact.framework && <><dt>Framework</dt><dd>{artifact.framework}</dd></>}
            <dt>Created by</dt>
            <dd>{displayName(artifact.created_by || artifact.reviewed_by)}</dd>
            <dt>Created</dt>
            <dd>{fmtDateTime(artifact.created_at)}</dd>
            {artifact.model && <><dt>Model</dt><dd>{artifact.model}</dd></>}
            {artifact.prompt_version && <><dt>Prompt version</dt><dd>{artifact.prompt_version}</dd></>}
          </dl>
          <nav class="workspace-links">
            <button class="btn btn-sm btn-secondary" onClick={() => navigateToPolicy(artifact.policy_id, "requirements")}>
              Requirements Matrix
            </button>
            <button class="btn btn-sm btn-secondary" onClick={() => {
              selectedPolicyId.value = artifact.policy_id;
              selectedEvidenceTargetId.value = null;
              navigate("evidence");
            }}>
              Evidence Library
            </button>
            <button class="btn btn-sm btn-secondary" onClick={() => navigateToPolicy(artifact.policy_id, "history")}>
              Full Audit History
            </button>
          </nav>
        </aside>
      </div>
    </section>
  );
}
