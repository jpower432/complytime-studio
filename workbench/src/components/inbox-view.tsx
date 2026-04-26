// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useRef, useCallback } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import { navigateToPolicy } from "../app";
import { downloadYaml, auditLogFilename } from "../lib/download";

interface Notification {
  notification_id: string;
  type: string;
  policy_id: string;
  payload: string;
  read: boolean;
  created_at: string;
}

interface DraftAuditLog {
  draft_id: string;
  policy_id: string;
  audit_start: string;
  audit_end: string;
  framework: string;
  created_at: string;
  status: string;
  summary: string;
  content?: string;
  agent_reasoning?: string;
  model?: string;
  prompt_version?: string;
  reviewed_by?: string;
  promoted_at?: string;
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

function parseYAMLContent(content: string): AuditResult[] | null {
  try {
    const resultsMatch = content.match(/^results:\s*$/m);
    if (!resultsMatch) return null;
    const results: AuditResult[] = [];
    const blocks = content.split(/^  - id:\s*/m).slice(1);
    for (const block of blocks) {
      const lines = block.split("\n");
      const id = lines[0]?.trim() || "";
      const title = lines.find((l) => l.includes("title:"))?.replace(/.*title:\s*/, "").trim() || "";
      const type = lines.find((l) => l.includes("type:") && !l.includes("evidence"))?.replace(/.*type:\s*/, "").trim() || "";
      const desc = lines.find((l) => l.includes("description:"))?.replace(/.*description:\s*/, "").trim() || "";
      const reasoning = lines.find((l) => l.includes("agent-reasoning:"))?.replace(/.*agent-reasoning:\s*>?-?\s*/, "").trim() || "";
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
    Strength: "var(--color-pass, #22c55e)",
    Finding: "var(--color-finding, #f59e0b)",
    Gap: "var(--color-gap, #ef4444)",
    Observation: "var(--color-observation, #6366f1)",
  };
  return (
    <span class="result-type-tag" style={{ background: colors[type] || "#94a3b8", color: "#fff", padding: "2px 8px", borderRadius: "4px", fontSize: "0.75rem", fontWeight: 600 }}>
      {type}
    </span>
  );
}

type InboxItem = { kind: "draft"; data: DraftAuditLog } | { kind: "notification"; data: Notification };

export function InboxView() {
  const [drafts, setDrafts] = useState<DraftAuditLog[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<DraftAuditLog | null>(null);
  const [results, setResults] = useState<AuditResult[] | null>(null);
  const [edits, setEdits] = useState<EditsMap>({});
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [promoting, setPromoting] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const fadeRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchAll = useCallback(() => {
    setLoading(true);
    Promise.all([
      apiFetch("/api/draft-audit-logs?status=pending_review").then((r) => r.json()).catch(() => []),
      apiFetch("/api/notifications").then((r) => r.json()).catch(() => []),
    ]).then(([d, n]) => {
      setDrafts(d);
      setNotifications(n);
    }).finally(() => setLoading(false));
  }, []);

  useEffect(fetchAll, [fetchAll]);

  const items: InboxItem[] = [
    ...drafts.map((d): InboxItem => ({ kind: "draft", data: d })),
    ...notifications.map((n): InboxItem => ({ kind: "notification", data: n })),
  ].sort((a, b) => {
    const ta = a.kind === "draft" ? a.data.created_at : a.data.created_at;
    const tb = b.kind === "draft" ? b.data.created_at : b.data.created_at;
    return new Date(tb).getTime() - new Date(ta).getTime();
  });

  const saveEdits = useCallback((draftId: string, editsMap: EditsMap) => {
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
    debounceRef.current = setTimeout(() => saveEdits(draftId, editsMap), 1000);
  }, [saveEdits]);

  const updateEdit = (resultId: string, field: "type_override" | "note", value: string) => {
    if (!selected) return;
    const next = { ...edits, [resultId]: { ...edits[resultId] || { type_override: "", note: "" }, [field]: value } };
    setEdits(next);
    scheduleAutoSave(selected.draft_id, next);
  };

  const openDraft = (draft: DraftAuditLog) => {
    apiFetch(`/api/draft-audit-logs/${encodeURIComponent(draft.draft_id)}`)
      .then((r) => r.json())
      .then((d: DraftAuditLog) => {
        setSelected(d);
        if (d.content) setResults(parseYAMLContent(d.content));
        setEdits(parseEdits(d.reviewer_edits));
        setSaveState("idle");
      })
      .catch(() => setSelected(draft));
  };

  const markRead = (notifId: string) => {
    apiFetch(`/api/notifications/${encodeURIComponent(notifId)}/read`, { method: "PATCH" }).catch(() => {});
    setNotifications((prev) => prev.map((n) => n.notification_id === notifId ? { ...n, read: true } : n));
  };

  const closeDetail = () => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    setSelected(null);
    setResults(null);
    setEdits({});
    setSaveState("idle");
  };

  const promote = () => {
    if (!selected) return;
    setPromoting(true);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    const finalize = () => {
      apiFetch("/api/audit-logs/promote", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ draft_id: selected.draft_id }),
      }).then((r) => {
        if (!r.ok) throw new Error("promote failed");
        closeDetail();
        fetchAll();
      }).catch(() => alert("Failed to promote draft"))
        .finally(() => setPromoting(false));
    };
    if (Object.keys(edits).length > 0) {
      apiFetch(`/api/draft-audit-logs/${encodeURIComponent(selected.draft_id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reviewer_edits: edits }),
      }).then(finalize).catch(finalize);
    } else {
      finalize();
    }
  };

  const parseSummary = (s: string) => { try { return JSON.parse(s); } catch { return null; } };
  const editFor = (id: string): EditEntry => edits[id] || { type_override: "", note: "" };

  return (
    <section class="inbox-view">
      <h2>Inbox</h2>
      <p class="view-subtitle">Agent-produced audit logs and compliance notifications.</p>

      {loading ? (
        <div class="view-loading">Loading inbox...</div>
      ) : items.length === 0 ? (
        <div class="empty-state"><p>No pending items. The agent will surface notifications here when evidence arrives or posture changes.</p></div>
      ) : (
        <div class="inbox-list">
          {items.map((item) => {
            if (item.kind === "draft") {
              const draft = item.data;
              const summary = parseSummary(draft.summary);
              return (
                <article key={draft.draft_id} class="inbox-card inbox-card-draft" onClick={() => openDraft(draft)}>
                  <div class="inbox-card-type">Draft Audit Log</div>
                  <div class="inbox-card-header">
                    <strong>{draft.policy_id}</strong>
                    <span class="inbox-card-date">{new Date(draft.created_at).toLocaleDateString()}</span>
                  </div>
                  <div class="inbox-card-period">
                    {new Date(draft.audit_start).toLocaleDateString()} — {new Date(draft.audit_end).toLocaleDateString()}
                  </div>
                  {summary && (
                    <div class="posture-counts">
                      <span class="count-pass">{summary.strengths ?? 0} strengths</span>
                      <span class="count-finding">{summary.findings ?? 0} findings</span>
                      <span class="count-gap">{summary.gaps ?? 0} gaps</span>
                    </div>
                  )}
                </article>
              );
            }
            const notif = item.data;
            const payload = parseSummary(notif.payload) || {};
            return (
              <article
                key={notif.notification_id}
                class={`inbox-card inbox-card-notification ${notif.read ? "read" : "unread"}`}
                onClick={() => {
                  if (!notif.read) markRead(notif.notification_id);
                  if (notif.policy_id) navigateToPolicy(notif.policy_id);
                }}
              >
                <div class="inbox-card-type">{notif.type === "posture_change" ? "Posture Change" : notif.type === "evidence_arrival" ? "Evidence Arrival" : notif.type}</div>
                <div class="inbox-card-header">
                  <strong>{notif.policy_id}</strong>
                  <span class="inbox-card-date">{new Date(notif.created_at).toLocaleDateString()}</span>
                </div>
                {payload.message && <p class="inbox-card-message">{payload.message}</p>}
              </article>
            );
          })}
        </div>
      )}

      {selected && (
        <div class="draft-detail">
          <div class="detail-header">
            <h3>Review Draft</h3>
            <div class="detail-actions">
              {saveState !== "idle" && (
                <span class={`save-indicator save-${saveState}`}>{saveState === "saving" ? "Saving..." : "Saved"}</span>
              )}
              {selected.content && (
                <button class="btn btn-sm btn-secondary" onClick={() => downloadYaml(selected.content!, auditLogFilename(selected.policy_id, selected.audit_start))}>
                  Download YAML
                </button>
              )}
              {selected.status === "pending_review" && (
                <button class="btn btn-primary" onClick={promote} disabled={promoting}>
                  {promoting ? "Saving..." : "Save to History"}
                </button>
              )}
              <button class="btn btn-sm" onClick={closeDetail}>Close</button>
            </div>
          </div>

          <div class="draft-meta-bar">
            <span>Policy: <strong>{selected.policy_id}</strong></span>
            <span>Created: {new Date(selected.created_at).toLocaleString()}</span>
            {selected.model && <span>Model: {selected.model}</span>}
          </div>

          {results ? (
            <div class="results-grid">
              {results.map((result) => {
                const edit = editFor(result.id);
                const displayType = edit.type_override || result.type;
                const overridden = edit.type_override !== "" && edit.type_override !== result.type;
                const editable = selected.status === "pending_review";
                return (
                  <article key={result.id} class={`result-card ${overridden ? "result-overridden" : ""}`}>
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
                    <p class="result-description">{result.description}</p>
                    {result["agent-reasoning"] && (
                      <div class="agent-reasoning"><strong>Agent Reasoning:</strong><p>{result["agent-reasoning"]}</p></div>
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
            <pre class="yaml-viewer">{selected.content || selected.summary}</pre>
          )}

          {selected.agent_reasoning && (
            <details class="reasoning-section">
              <summary>Full Agent Reasoning</summary>
              <pre class="yaml-viewer">{selected.agent_reasoning}</pre>
            </details>
          )}
        </div>
      )}
    </section>
  );
}
