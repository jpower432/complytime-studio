// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useRef, useCallback } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import { downloadYaml, auditLogFilename } from "../lib/download";

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
  evidence?: { type: string; collected: string; description: string }[];
  recommendations?: { text: string }[];
}

interface ParsedContent {
  results: AuditResult[];
  summary?: string;
}

interface EditEntry {
  type_override: string;
  note: string;
}

type EditsMap = Record<string, EditEntry>;

function parseYAMLContent(content: string): ParsedContent | null {
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
    return { results };
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
    <span
      class="result-type-tag"
      style={{ background: colors[type] || "#94a3b8", color: "#fff", padding: "2px 8px", borderRadius: "4px", fontSize: "0.75rem", fontWeight: 600 }}
    >
      {type}
    </span>
  );
}

const RESULT_TYPES = ["Strength", "Finding", "Gap", "Observation"];

interface ResultCardProps {
  result: AuditResult;
  editable: boolean;
  edit: EditEntry;
  onTypeChange: (type: string) => void;
  onNoteChange: (note: string) => void;
}

function ResultCard({ result, editable, edit, onTypeChange, onNoteChange }: ResultCardProps) {
  const [showNote, setShowNote] = useState(!!edit.note);
  const displayType = edit.type_override || result.type;
  const overridden = edit.type_override !== "" && edit.type_override !== result.type;

  return (
    <article class={`result-card ${overridden ? "result-overridden" : ""}`}>
      <div class="result-card-header">
        <span class="result-id">{result.id}</span>
        {editable ? (
          <select
            class="result-type-select"
            value={displayType}
            onChange={(e) => onTypeChange((e.target as HTMLSelectElement).value)}
          >
            {RESULT_TYPES.map((t) => (
              <option key={t} value={t}>{t}{t === result.type ? " (agent)" : ""}</option>
            ))}
          </select>
        ) : (
          <ResultTypeTag type={displayType} />
        )}
      </div>
      <h4>{result.title}</h4>
      <p class="result-description">{result.description}</p>
      {result["agent-reasoning"] && (
        <div class="agent-reasoning">
          <strong>Agent Reasoning:</strong>
          <p>{result["agent-reasoning"]}</p>
        </div>
      )}
      {editable && (
        <div class="result-controls">
          {overridden && <span class="override-badge">Overridden</span>}
          <button class="btn btn-xs" onClick={() => setShowNote(!showNote)}>
            {showNote ? "Hide Note" : "Add Note"}
          </button>
          {showNote && (
            <textarea
              class="result-note"
              placeholder="Add reviewer note..."
              value={edit.note}
              onInput={(e) => onNoteChange((e.target as HTMLTextAreaElement).value)}
              rows={2}
            />
          )}
        </div>
      )}
    </article>
  );
}

type SaveState = "idle" | "saving" | "saved";

export function DraftReviewView() {
  const [drafts, setDrafts] = useState<DraftAuditLog[]>([]);
  const [selected, setSelected] = useState<DraftAuditLog | null>(null);
  const [parsed, setParsed] = useState<ParsedContent | null>(null);
  const [loading, setLoading] = useState(true);
  const [promoting, setPromoting] = useState(false);
  const [statusFilter, setStatusFilter] = useState("pending_review");
  const [edits, setEdits] = useState<EditsMap>({});
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const fadeRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const saveEdits = useCallback((draftId: string, editsMap: EditsMap) => {
    setSaveState("saving");
    apiFetch(`/api/draft-audit-logs/${encodeURIComponent(draftId)}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reviewer_edits: editsMap }),
    })
      .then((r) => {
        if (!r.ok) throw new Error("save failed");
        setSaveState("saved");
        if (fadeRef.current) clearTimeout(fadeRef.current);
        fadeRef.current = setTimeout(() => setSaveState("idle"), 2000);
      })
      .catch(() => setSaveState("idle"));
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

  const fetchDrafts = () => {
    setLoading(true);
    const params = statusFilter ? `?status=${statusFilter}` : "";
    apiFetch(`/api/draft-audit-logs${params}`)
      .then((r) => r.json())
      .then(setDrafts)
      .catch(() => setDrafts([]))
      .finally(() => setLoading(false));
  };

  useEffect(fetchDrafts, [statusFilter]);

  const loadDetail = (draft: DraftAuditLog) => {
    apiFetch(`/api/draft-audit-logs/${encodeURIComponent(draft.draft_id)}`)
      .then((r) => r.json())
      .then((d: DraftAuditLog) => {
        setSelected(d);
        if (d.content) setParsed(parseYAMLContent(d.content));
        setEdits(parseEdits(d.reviewer_edits));
        setSaveState("idle");
      })
      .catch(() => setSelected(draft));
  };

  const closeDetail = () => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    setSelected(null);
    setParsed(null);
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
      })
        .then((r) => {
          if (!r.ok) throw new Error("Save failed");
          closeDetail();
          fetchDrafts();
        })
        .catch(() => alert("Failed to promote draft"))
        .finally(() => setPromoting(false));
    };
    const hasUnsaved = Object.keys(edits).length > 0;
    if (hasUnsaved) {
      apiFetch(`/api/draft-audit-logs/${encodeURIComponent(selected.draft_id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reviewer_edits: edits }),
      }).then(finalize).catch(finalize);
    } else {
      finalize();
    }
  };

  const parseSummary = (s: string) => {
    try { return JSON.parse(s); } catch { return null; }
  };

  const editFor = (id: string): EditEntry => edits[id] || { type_override: "", note: "" };

  return (
    <section class="draft-review-view">
      <h2>Review</h2>
      <p class="view-subtitle">Agent-produced audit logs awaiting review before promotion to official record.</p>

      <div class="audit-filters">
        <select value={statusFilter} onChange={(e) => setStatusFilter((e.target as HTMLSelectElement).value)}>
          <option value="pending_review">Pending Review</option>
          <option value="promoted">Saved</option>
          <option value="">All</option>
        </select>
        <button class="btn btn-primary" onClick={fetchDrafts}>Refresh</button>
      </div>

      {loading ? (
        <div class="view-loading">Loading drafts...</div>
      ) : drafts.length === 0 ? (
        <div class="empty-state">
          <p>No draft audit logs {statusFilter ? `with status "${statusFilter}"` : ""}.</p>
        </div>
      ) : (
        <div class="audit-timeline">
          {drafts.map((draft) => {
            const summary = parseSummary(draft.summary);
            return (
              <article key={draft.draft_id} class={`audit-card ${draft.status === "promoted" ? "promoted" : ""}`} onClick={() => loadDetail(draft)}>
                <div class="audit-card-header">
                  <span class="audit-period">
                    {new Date(draft.audit_start).toLocaleDateString()} — {new Date(draft.audit_end).toLocaleDateString()}
                  </span>
                  <span class={`draft-status status-${draft.status}`}>{draft.status.replace("_", " ")}</span>
                </div>
                <div class="audit-card-meta">
                  <span>Policy: {draft.policy_id}</span>
                  {draft.model && <span>Model: {draft.model}</span>}
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
          })}
        </div>
      )}

      {selected && (
        <div class="draft-detail">
          <div class="detail-header">
            <h3>Review</h3>
            <div class="detail-actions">
              {saveState !== "idle" && (
                <span class={`save-indicator save-${saveState}`}>
                  {saveState === "saving" ? "Saving..." : "Saved"}
                </span>
              )}
              {selected.content && (
                <button
                  class="btn btn-sm btn-secondary"
                  onClick={() => downloadYaml(selected.content!, auditLogFilename(selected.policy_id, selected.audit_start))}
                >
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
            {selected.reviewed_by && <span>Reviewed by: {selected.reviewed_by}</span>}
          </div>

          {parsed?.results ? (
            <div class="results-grid">
              {parsed.results.map((result) => (
                <ResultCard
                  key={result.id}
                  result={result}
                  editable={selected.status === "pending_review"}
                  edit={editFor(result.id)}
                  onTypeChange={(v) => updateEdit(result.id, "type_override", v)}
                  onNoteChange={(v) => updateEdit(result.id, "note", v)}
                />
              ))}
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
