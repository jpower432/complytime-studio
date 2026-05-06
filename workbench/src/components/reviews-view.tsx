// SPDX-License-Identifier: Apache-2.0

import { useCallback, useEffect, useMemo, useRef, useState } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import { navigateToReview, viewInvalidation } from "../app";
import { cardKeyHandler } from "../lib/a11y";
import { displayName, fmtDate, fmtDateTime } from "../lib/format";

interface Policy {
  policy_id: string;
  title: string;
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
}

interface SummaryCounts {
  strengths?: number;
  findings?: number;
  gaps?: number;
}

interface EvidenceRow {
  resultId: string;
  type: string;
  collected: string;
  description: string;
  provenance: string;
}

interface ParsedDraftContent {
  scopeText: string;
  targetText: string;
  methodology: string;
  metadataDate: string;
  authorName: string;
  evidence: EvidenceRow[];
}

function parseSummaryJson(s: string): SummaryCounts | null {
  try {
    return JSON.parse(s) as SummaryCounts;
  } catch {
    return null;
  }
}

function extractResultsSection(content: string): string {
  const m = content.match(/^results:\s*$/m);
  if (!m || m.index === undefined) {
    return "";
  }
  return content.slice(m.index);
}

function parseEvidenceRows(resultsPart: string): EvidenceRow[] {
  const out: EvidenceRow[] = [];
  if (!resultsPart) {
    return out;
  }
  const blocks = resultsPart.split(/\n  - id:\s*/);
  for (let b = 1; b < blocks.length; b++) {
    const raw = blocks[b];
    const nl = raw.indexOf("\n");
    const resultId = (nl >= 0 ? raw.slice(0, nl) : raw).trim();
    const body = nl >= 0 ? raw.slice(nl + 1) : "";
    const lines = body.split("\n");
    const evIdx = lines.findIndex((l) => /^ {4}evidence:\s*$/.test(l));
    if (evIdx < 0) {
      continue;
    }
    for (let i = evIdx + 1; i < lines.length; i++) {
      const line = lines[i];
      if (/^ {4}\S/.test(line)) {
        break;
      }
      const typeM = line.match(/^ {6}- type:\s*(.+)$/);
      if (!typeM) {
        continue;
      }
      const row: EvidenceRow = {
        resultId,
        type: typeM[1].trim(),
        collected: "",
        description: "",
        provenance: "",
      };
      let j = i + 1;
      while (j < lines.length) {
        const L = lines[j];
        if (/^ {6}- /.test(L)) {
          break;
        }
        if (/^ {4}\S/.test(L)) {
          break;
        }
        const col = L.match(/^ {8}collected:\s*(.+)$/);
        if (col) {
          row.collected = col[1].trim().replace(/^["']|["']$/g, "");
        }
        const desc = L.match(/^ {8}description:\s*(.+)$/);
        if (desc) {
          row.description = desc[1].trim();
        }
        const refLine = L.match(/reference-id:\s*(.+)$/);
        if (refLine) {
          row.provenance = refLine[1].trim();
        }
        j++;
      }
      out.push(row);
      i = j - 1;
    }
  }
  return out;
}

function extractIndentedBlock(content: string, rootKey: string): string {
  const lines = content.split("\n");
  const re = new RegExp(`^${rootKey}:\\s*$`);
  const idx = lines.findIndex((l) => re.test(l));
  if (idx < 0) {
    return "";
  }
  const chunk: string[] = [];
  for (let j = idx + 1; j < lines.length; j++) {
    const line = lines[j];
    if (line.trim() === "") {
      chunk.push("");
      continue;
    }
    if (!line.startsWith("  ")) {
      break;
    }
    chunk.push(line);
  }
  return chunk.join("\n").trim();
}

function extractScalarRoot(content: string, key: string): string {
  const m = content.match(new RegExp(`^${key}:\\s*(.+)$`, "m"));
  if (!m) {
    return "";
  }
  return m[1].trim().replace(/^["']|["']$/g, "");
}

function extractMetadataDate(content: string): string {
  const lines = content.split("\n");
  let inMeta = false;
  for (const line of lines) {
    if (line.match(/^metadata:\s*$/)) {
      inMeta = true;
      continue;
    }
    if (inMeta && line.match(/^\S/) && !line.startsWith("  ")) {
      break;
    }
    if (inMeta) {
      const dm = line.match(/^ {2}date:\s*(.+)$/);
      if (dm) {
        return dm[1].trim().replace(/^["']|["']$/g, "");
      }
    }
  }
  return "";
}

function extractMetadataAuthorName(content: string): string {
  const lines = content.split("\n");
  let inMeta = false;
  let inAuthor = false;
  for (const line of lines) {
    if (line.match(/^metadata:\s*$/)) {
      inMeta = true;
      continue;
    }
    if (inMeta && line.match(/^\S/) && !line.startsWith("  ")) {
      break;
    }
    if (inMeta && line.match(/^  author:\s*$/)) {
      inAuthor = true;
      continue;
    }
    if (inAuthor && line.match(/^    name:\s*(.+)$/)) {
      return line.replace(/^    name:\s*/, "").trim().replace(/^["']|["']$/g, "");
    }
    if (inAuthor && line.match(/^  \S/) && !line.match(/^    /)) {
      inAuthor = false;
    }
  }
  return "";
}

function extractTargetSummary(content: string): string {
  const block = extractIndentedBlock(content, "target");
  if (!block) {
    return "";
  }
  const id = block.match(/^\s{2}id:\s*(.+)$/m);
  const name = block.match(/^\s{2}name:\s*(.+)$/m);
  const parts: string[] = [];
  if (name) {
    parts.push(name[1].trim().replace(/^["']|["']$/g, ""));
  }
  if (id) {
    parts.push(id[1].trim().replace(/^["']|["']$/g, ""));
  }
  return parts.filter(Boolean).join(" · ");
}

function parseDraftContent(content: string): ParsedDraftContent {
  const scopeText = extractIndentedBlock(content, "scope");
  const targetText = extractTargetSummary(content);
  let methodology = extractScalarRoot(content, "methodology");
  if (!methodology) {
    const inner = extractIndentedBlock(content, "metadata");
    const m = inner.match(/^\s{2}methodology:\s*(.+)$/m);
    if (m) {
      methodology = m[1].trim().replace(/^["']|["']$/g, "");
    }
  }
  return {
    scopeText,
    targetText,
    methodology,
    metadataDate: extractMetadataDate(content),
    authorName: extractMetadataAuthorName(content),
    evidence: parseEvidenceRows(extractResultsSection(content)),
  };
}

function statusLabel(status: string): string {
  return status.replace(/_/g, " ");
}

export function ReviewsView() {
  const [drafts, setDrafts] = useState<DraftAuditLog[]>([]);
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState("");
  const [policyFilter, setPolicyFilter] = useState("");
  const [sortOrder, setSortOrder] = useState<"newest" | "oldest">("newest");
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [expandedDraft, setExpandedDraft] = useState<DraftAuditLog | null>(null);
  const [expandedLoading, setExpandedLoading] = useState(false);
  const fetchTicket = useRef(0);

  const policyTitleById = useMemo(() => {
    const m = new Map<string, string>();
    for (const p of policies) {
      m.set(p.policy_id, p.title);
    }
    return m;
  }, [policies]);

  const loadPolicies = useCallback(() => {
    apiFetch("/api/policies")
      .then((r) => r.json())
      .then((rows: Policy[]) => setPolicies(Array.isArray(rows) ? rows : []))
      .catch(() => setPolicies([]));
  }, []);

  const loadDrafts = useCallback(() => {
    setLoading(true);
    const qs = new URLSearchParams();
    if (statusFilter) {
      qs.set("status", statusFilter);
    }
    qs.set("limit", "1000");
    const q = qs.toString();
    apiFetch(`/api/draft-audit-logs?${q}`)
      .then((r) => r.json())
      .then((rows: DraftAuditLog[]) => setDrafts(Array.isArray(rows) ? rows : []))
      .catch(() => setDrafts([]))
      .finally(() => setLoading(false));
  }, [statusFilter]);

  useEffect(() => {
    loadPolicies();
  }, [loadPolicies]);

  useEffect(() => {
    loadDrafts();
  }, [loadDrafts, viewInvalidation.value]);

  const filteredSorted = useMemo(() => {
    let rows = drafts;
    if (policyFilter) {
      rows = rows.filter((d) => d.policy_id === policyFilter);
    }
    const mul = sortOrder === "newest" ? -1 : 1;
    return [...rows].sort(
      (a, b) =>
        mul * (new Date(a.created_at).getTime() - new Date(b.created_at).getTime()),
    );
  }, [drafts, policyFilter, sortOrder]);

  const toggleExpand = (id: string) => {
    if (expandedId === id) {
      setExpandedId(null);
      setExpandedDraft(null);
      return;
    }
    fetchTicket.current += 1;
    const ticket = fetchTicket.current;
    setExpandedId(id);
    setExpandedDraft(null);
    setExpandedLoading(true);
    apiFetch(`/api/draft-audit-logs/${encodeURIComponent(id)}`)
      .then((r) => {
        if (!r.ok) {
          throw new Error("load failed");
        }
        return r.json();
      })
      .then((d: DraftAuditLog) => {
        if (ticket === fetchTicket.current) {
          setExpandedDraft(d);
        }
      })
      .catch(() => {
        if (ticket === fetchTicket.current) {
          setExpandedDraft(null);
        }
      })
      .finally(() => {
        if (ticket === fetchTicket.current) {
          setExpandedLoading(false);
        }
      });
  };

  const expandedParsed = expandedDraft
    ? parseDraftContent(expandedDraft.content ?? "")
    : null;

  return (
    <section class="reviews-view">
      <header class="reviews-header">
        <h2>Reviews</h2>
        <div class="reviews-filters">
          <select
            value={statusFilter}
            onChange={(e) =>
              setStatusFilter((e.target as HTMLSelectElement).value)
            }
            aria-label="Filter by status"
          >
            <option value="">All</option>
            <option value="pending_review">Pending Review</option>
            <option value="promoted">Promoted</option>
            <option value="expired">Expired</option>
          </select>
          <select
            value={policyFilter}
            onChange={(e) =>
              setPolicyFilter((e.target as HTMLSelectElement).value)
            }
            aria-label="Filter by policy"
          >
            <option value="">All policies</option>
            {policies.map((p) => (
              <option key={p.policy_id} value={p.policy_id}>
                {p.title || p.policy_id}
              </option>
            ))}
          </select>
          <select
            value={sortOrder}
            onChange={(e) =>
              setSortOrder(
                (e.target as HTMLSelectElement).value as "newest" | "oldest",
              )
            }
            aria-label="Sort order"
          >
            <option value="newest">Newest first</option>
            <option value="oldest">Oldest first</option>
          </select>
        </div>
      </header>

      {loading ? (
        <div class="view-loading">Loading reviews...</div>
      ) : drafts.length === 0 ? (
        <div class="empty-state">
          <p>
            No audit reviews pending. Reviews will appear here when the assistant
            completes an audit.
          </p>
        </div>
      ) : filteredSorted.length === 0 ? (
        <div class="empty-state">
          <p>No drafts match the selected filters.</p>
        </div>
      ) : (
        <div class="reviews-list">
          {filteredSorted.map((draft) => {
            const summary = parseSummaryJson(draft.summary);
            const title =
              policyTitleById.get(draft.policy_id) || draft.policy_id;
            const agentLabel = draft.model
              ? displayName(draft.model)
              : "Assistant";
            const isOpen = expandedId === draft.draft_id;
            return (
              <div
                key={draft.draft_id}
                class={`review-card-block${isOpen ? " is-open" : ""}`}
              >
                <article
                  class="review-card"
                  onClick={() => navigateToReview(draft.draft_id)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={cardKeyHandler(() =>
                    navigateToReview(draft.draft_id),
                  )}
                  aria-label={`Open review ${title}`}
                >
                  <div class="review-card-left">
                    <div class="review-card-title">{title}</div>
                    <div class="review-card-meta">
                      <span>{draft.framework || "—"}</span>
                      <span>
                        {fmtDate(draft.audit_start)} — {fmtDate(draft.audit_end)}
                      </span>
                      <span>Created by {agentLabel}</span>
                      <span>{fmtDateTime(draft.created_at)}</span>
                    </div>
                  </div>
                  <div class="review-card-right">
                    {summary && (
                      <div class="review-card-counts">
                        <span class="count-pass">
                          {summary.strengths ?? 0} pass
                        </span>
                        <span class="count-finding">
                          {summary.findings ?? 0} finding
                        </span>
                        <span class="count-gap">
                          {summary.gaps ?? 0} gap
                        </span>
                      </div>
                    )}
                    <span class={`draft-status status-${draft.status}`}>
                      {statusLabel(draft.status)}
                    </span>
                    <button
                      type="button"
                      class="btn btn-xs btn-secondary"
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleExpand(draft.draft_id);
                      }}
                      aria-expanded={isOpen}
                    >
                      {isOpen ? "Less" : "Details"}
                    </button>
                  </div>
                </article>
                {isOpen && (
                  <div
                    class="review-card-expanded"
                    onClick={(e) => e.stopPropagation()}
                  >
                    {expandedLoading ? (
                      <p class="reviews-expanded-loading">Loading details…</p>
                    ) : expandedDraft?.draft_id === draft.draft_id &&
                      expandedParsed ? (
                      <>
                        <h4 class="reviews-expanded-heading">Scope & methodology</h4>
                        <dl class="reviews-detail-dl">
                          <dt>Period</dt>
                          <dd>
                            {fmtDate(expandedDraft.audit_start)} —{" "}
                            {fmtDate(expandedDraft.audit_end)}
                            {expandedParsed.metadataDate && (
                              <span class="reviews-meta-date">
                                {" "}
                                (report {fmtDate(expandedParsed.metadataDate)})
                              </span>
                            )}
                          </dd>
                          {expandedParsed.scopeText && (
                            <>
                              <dt>Scope</dt>
                              <dd>
                                <pre class="reviews-pre-wrap">
                                  {expandedParsed.scopeText}
                                </pre>
                              </dd>
                            </>
                          )}
                          {expandedParsed.targetText && !expandedParsed.scopeText && (
                            <>
                              <dt>Target</dt>
                              <dd>{expandedParsed.targetText}</dd>
                            </>
                          )}
                          <dt>Methodology</dt>
                          <dd>
                            {expandedParsed.methodology || "—"}
                          </dd>
                          {(expandedParsed.authorName || expandedDraft.model) && (
                            <>
                              <dt>Author</dt>
                              <dd>
                                {expandedParsed.authorName ||
                                  displayName(expandedDraft.model)}
                              </dd>
                            </>
                          )}
                        </dl>
                        <h4 class="reviews-expanded-heading">Evidence provenance</h4>
                        {expandedParsed.evidence.length === 0 ? (
                          <p class="reviews-muted">No evidence entries in draft YAML.</p>
                        ) : (
                          <table class="reviews-evidence-table">
                            <thead>
                              <tr>
                                <th>Result</th>
                                <th>Type</th>
                                <th>Collected</th>
                                <th>Description</th>
                                <th>Provenance</th>
                              </tr>
                            </thead>
                            <tbody>
                              {expandedParsed.evidence.map((ev, idx) => (
                                <tr key={`${ev.resultId}-${idx}`}>
                                  <td class="mono">{ev.resultId}</td>
                                  <td>{ev.type}</td>
                                  <td>
                                    {ev.collected
                                      ? fmtDateTime(ev.collected)
                                      : "—"}
                                  </td>
                                  <td>{ev.description || "—"}</td>
                                  <td class="mono">{ev.provenance || "—"}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        )}
                      </>
                    ) : (
                      <p class="reviews-muted">Could not load draft details.</p>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}

