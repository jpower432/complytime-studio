// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useCallback } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import { navigateToPolicy, navigateToAudit, invalidateInbox } from "../app";
import { cardKeyHandler } from "../lib/a11y";

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

type InboxItem = { kind: "draft"; data: DraftAuditLog } | { kind: "notification"; data: Notification };

export function InboxView() {
  const [drafts, setDrafts] = useState<DraftAuditLog[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchAll = useCallback(() => {
    setLoading(true);
    Promise.all([
      apiFetch("/api/draft-audit-logs?status=pending_review").then((r) => r.json()).catch(() => []),
      apiFetch("/api/notifications").then((r) => r.json()).catch(() => []),
    ]).then(([d, n]) => {
      setDrafts(d);
      setNotifications(n.filter((notif: Notification) => !notif.read));
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

  const markRead = (notifId: string) => {
    apiFetch(
      `/api/notifications/${encodeURIComponent(notifId)}/read`,
      { method: "PATCH" },
    ).catch(() => {});
    setNotifications((prev) =>
      prev.map((n) =>
        n.notification_id === notifId ? { ...n, read: true } : n
      )
    );
    invalidateInbox();
  };

  const dismissNotification = (notifId: string) => {
    apiFetch(
      `/api/notifications/${encodeURIComponent(notifId)}/read`,
      { method: "PATCH" },
    ).catch(() => {});
    setNotifications((prev) =>
      prev.filter((n) => n.notification_id !== notifId)
    );
    invalidateInbox();
  };

  const parseSummary = (s: string) => { try { return JSON.parse(s); } catch { return null; } };

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
                <article
                  key={draft.draft_id}
                  class={`inbox-card inbox-card-draft ${draft.status === "pending_review" ? "unread" : ""}`}
                  onClick={() => navigateToAudit(draft.draft_id)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={cardKeyHandler(() => navigateToAudit(draft.draft_id))}
                  aria-label={`Review draft for ${draft.policy_id}`}
                >
                  <div class="inbox-card-type-row">
                    <span class="inbox-card-type">Draft Audit Log</span>
                    <span class={`inbox-status-badge status-${draft.status}`}>
                      {draft.status === "pending_review" ? "Needs Review" : draft.status}
                    </span>
                  </div>
                  <div class="inbox-card-header">
                    <strong>{draft.policy_id}</strong>
                    <span class="inbox-card-date">{new Date(draft.created_at).toLocaleDateString()}</span>
                  </div>
                  {draft.framework && <div class="inbox-card-framework">{draft.framework}</div>}
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
                  {summary?.message && <p class="inbox-card-message">{summary.message}</p>}
                </article>
              );
            }
            const notif = item.data;
            const payload = parseSummary(notif.payload) || {};
            return (
              <article
                key={notif.notification_id}
                class={`inbox-card inbox-card-notification ${notif.read ? "read" : "unread"}`}
                onClick={() => { if (!notif.read) markRead(notif.notification_id); }}
                role="button"
                tabIndex={0}
                onKeyDown={cardKeyHandler(() => { if (!notif.read) markRead(notif.notification_id); })}
                aria-label={`${notif.type === "posture_change" ? "Posture change" : "Evidence arrival"} for ${notif.policy_id}`}
              >
                <div class="inbox-card-type">{notif.type === "posture_change" ? "Posture Change" : notif.type === "evidence_arrival" ? "Evidence Arrival" : notif.type}</div>
                <div class="inbox-card-header">
                  <strong>{notif.policy_id}</strong>
                  <span class="inbox-card-date">{new Date(notif.created_at).toLocaleDateString()}</span>
                </div>
                {payload.message && <p class="inbox-card-message">{payload.message}</p>}
                {payload.previous_rate !== undefined && payload.current_rate !== undefined && (
                  <div class="inbox-card-delta">
                    <span>{(payload.previous_rate * 100).toFixed(0)}%</span>
                    <span> → </span>
                    <span>{(payload.current_rate * 100).toFixed(0)}%</span>
                  </div>
                )}
                <div class="inbox-card-actions">
                  {notif.policy_id && (
                    <button
                      class="btn btn-sm btn-link"
                      onClick={(e) => { e.stopPropagation(); navigateToPolicy(notif.policy_id); }}
                    >
                      View Policy →
                    </button>
                  )}
                  <button
                    class="btn btn-sm btn-dismiss"
                    onClick={(e) => { e.stopPropagation(); dismissNotification(notif.notification_id); }}
                    title="Dismiss"
                  >
                    ✕
                  </button>
                </div>
              </article>
            );
          })}
        </div>
      )}

    </section>
  );
}
