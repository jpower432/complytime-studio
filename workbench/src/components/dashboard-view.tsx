// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useCallback } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import {
  navigate,
  navigateToPolicy,
  navigateToProgram,
  invalidateInbox,
  viewInvalidation,
} from "../app";
import { cardKeyHandler } from "../lib/a11y";
import { fmtDateTime } from "../lib/format";

interface Program {
  id: string;
  name: string;
  framework: string;
  status: string;
  health?: string | null;
  policy_ids: string[];
}

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
  status: string;
  created_at: string;
}

function parsePayload(raw: string): Record<string, unknown> | null {
  try {
    return JSON.parse(raw) as Record<string, unknown>;
  } catch {
    return null;
  }
}

function notificationMessage(n: Notification): string {
  const p = parsePayload(n.payload);
  const msg = p?.message;
  if (typeof msg === "string" && msg.trim()) return msg;
  if (n.type === "posture_change") return "Posture change detected";
  if (n.type === "evidence_arrival") return "New evidence arrived";
  return n.type.replace(/_/g, " ") || "Notification";
}

function notificationSeverity(n: Notification): string {
  const p = parsePayload(n.payload);
  const s = p?.severity;
  if (typeof s === "string" && s.trim()) return s.toLowerCase();
  if (n.type === "posture_change") return "warning";
  return "info";
}

function severityBadgeClass(sev: string): string {
  if (sev === "error" || sev === "critical") return "dashboard-severity-error";
  if (sev === "warning") return "dashboard-severity-warning";
  return "dashboard-severity-info";
}

function programHealthDot(health: string | null | undefined): string {
  const h = (health ?? "").toLowerCase();
  if (h === "red") return "health-red";
  if (h === "yellow") return "health-yellow";
  if (h === "green") return "health-green";
  return "health-yellow";
}

export function DashboardView() {
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [metrics, setMetrics] = useState({
    programs: 0,
    policies: 0,
    pendingReviews: 0,
    unread: 0,
  });
  const [programs, setPrograms] = useState<Program[]>([]);
  const [actionableUnread, setActionableUnread] = useState<Notification[]>([]);
  const [feedNotifs, setFeedNotifs] = useState<Notification[]>([]);

  const fetchDashboard = useCallback(() => {
    const failures: string[] = [];

    const programsP = apiFetch("/api/programs")
      .then((r) => {
        if (!r.ok) {
          failures.push("programs");
          return [] as Program[];
        }
        return r.json() as Promise<Program[]>;
      })
      .catch(() => {
        failures.push("programs");
        return [] as Program[];
      });

    const policiesP = apiFetch("/api/policies")
      .then((r) => {
        if (!r.ok) {
          failures.push("policies");
          return [];
        }
        return r.json() as Promise<unknown[]>;
      })
      .catch(() => {
        failures.push("policies");
        return [];
      });

    const draftsP = apiFetch("/api/draft-audit-logs?status=pending_review")
      .then((r) => {
        if (!r.ok) {
          failures.push("reviews");
          return [] as DraftAuditLog[];
        }
        return r.json() as Promise<DraftAuditLog[]>;
      })
      .catch(() => {
        failures.push("reviews");
        return [] as DraftAuditLog[];
      });

    const unreadP = apiFetch("/api/notifications/unread-count")
      .then((r) => {
        if (!r.ok) {
          failures.push("notifications");
          return { count: 0 };
        }
        return r.json() as Promise<{ count: number }>;
      })
      .catch(() => {
        failures.push("notifications");
        return { count: 0 };
      });

    const feedP = apiFetch("/api/notifications?limit=5")
      .then((r) => {
        if (!r.ok) {
          failures.push("notification feed");
          return [] as Notification[];
        }
        return r.json() as Promise<Notification[]>;
      })
      .catch(() => {
        failures.push("notification feed");
        return [] as Notification[];
      });

    const unreadListP = apiFetch("/api/notifications?unread=true")
      .then((r) => {
        if (!r.ok) {
          failures.push("notifications");
          return [] as Notification[];
        }
        return r.json() as Promise<Notification[]>;
      })
      .catch(() => {
        failures.push("notifications");
        return [] as Notification[];
      });

    Promise.all([
      programsP,
      policiesP,
      draftsP,
      unreadP,
      feedP,
      unreadListP,
    ]).then(([progRows, polRows, drafts, unreadRow, feed, unreadRaw]) => {
      const unreadNotifs = unreadRaw.filter((n) => !n.read).slice(0, 8);
      setPrograms(progRows);
      setMetrics({
        programs: progRows.length,
        policies: polRows.length,
        pendingReviews: drafts.length,
        unread: unreadRow.count,
      });
      setActionableUnread(unreadNotifs);
      setFeedNotifs(feed);
      setFetchError(
        failures.length
          ? `Some data failed to load (${[...new Set(failures)].join(", ")}).`
          : null,
      );
      setLoading(false);
    });
  }, []);

  useEffect(() => {
    fetchDashboard();
  }, [fetchDashboard, viewInvalidation.value]);

  const applyMarkRead = (id: string) => {
    apiFetch(`/api/notifications/${encodeURIComponent(id)}/read`, {
      method: "PATCH",
    }).catch(() => {});
    setFeedNotifs((prev) =>
      prev.map((n) =>
        n.notification_id === id ? { ...n, read: true } : n,
      ),
    );
    setActionableUnread((prev) =>
      prev.filter((n) => n.notification_id !== id),
    );
    setMetrics((m) => ({ ...m, unread: Math.max(0, m.unread - 1) }));
    invalidateInbox();
  };

  const onFeedItemActivate = (n: Notification) => {
    if (!n.read) applyMarkRead(n.notification_id);
    if (n.policy_id) navigateToPolicy(n.policy_id);
  };

  const dismissActionable = (e: Event, id: string) => {
    e.stopPropagation();
    applyMarkRead(id);
  };

  if (loading) {
    return <div class="view-loading">Loading dashboard…</div>;
  }

  return (
    <section class="dashboard-view">
      <h2>Dashboard</h2>
      <p class="view-subtitle">
        Programs, policies, reviews, and notifications at a glance.
      </p>

      {fetchError ? (
        <div class="dashboard-fetch-warning" role="status">
          {fetchError}
        </div>
      ) : null}

      <div class="dashboard-metrics">
        <div class="metric-card">
          <div class="metric-value">{metrics.programs}</div>
          <div class="metric-label">Programs</div>
        </div>
        <div class="metric-card">
          <div class="metric-value">{metrics.policies}</div>
          <div class="metric-label">Policies</div>
        </div>
        <div class="metric-card">
          <div class="metric-value">{metrics.pendingReviews}</div>
          <div class="metric-label">Pending reviews</div>
        </div>
        <div class="metric-card">
          <div class="metric-value">{metrics.unread}</div>
          <div class="metric-label">Unread notifications</div>
        </div>
      </div>

      <div class="dashboard-section">
        <h3>Program health</h3>
        {programs.length === 0 ? (
          <div class="empty-state dashboard-programs-empty">
            <p>
              No programs configured. Create one from the Programs view.
            </p>
          </div>
        ) : (
          <div class="program-health-grid">
            {programs.map((p) => {
              const dot = programHealthDot(p.health ?? null);
              const policyCount = p.policy_ids?.length ?? 0;
              return (
                <article
                  key={p.id}
                  class="program-health-card"
                  onClick={() => navigateToProgram(p.id)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={cardKeyHandler(() => navigateToProgram(p.id))}
                  aria-label={`Open program ${p.name}`}
                >
                  <h4>{p.name}</h4>
                  <div class="program-meta">
                    <span>{p.framework}</span>
                    <span class="program-status-pill">{p.status}</span>
                    <span>
                      <span
                        class={`health-dot ${dot}`}
                        aria-hidden
                      />
                      {p.health ?? "—"}
                    </span>
                    <span>
                      {policyCount} polic{policyCount === 1 ? "y" : "ies"}
                    </span>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </div>

      <div class="dashboard-section">
        <h3>Actionable items</h3>
        <div class="actionable-list">
          {metrics.pendingReviews > 0 ? (
            <div
              class="actionable-item"
              onClick={() => navigate("reviews")}
              role="button"
              tabIndex={0}
              onKeyDown={cardKeyHandler(() => navigate("reviews"))}
            >
              <span>
                {metrics.pendingReviews} audit review
                {metrics.pendingReviews === 1 ? "" : "s"} pending
              </span>
              <span class="actionable-chevron">Reviews →</span>
            </div>
          ) : null}
          {actionableUnread.map((n) => (
            <div
              key={n.notification_id}
              class="actionable-item actionable-item-split"
              onClick={() => {
                applyMarkRead(n.notification_id);
                if (n.policy_id) navigateToPolicy(n.policy_id);
              }}
              role="button"
              tabIndex={0}
              onKeyDown={cardKeyHandler(() => {
                applyMarkRead(n.notification_id);
                if (n.policy_id) navigateToPolicy(n.policy_id);
              })}
            >
              <span class="actionable-item-text">
                {notificationMessage(n)}
                {n.policy_id ? (
                  <span class="actionable-policy-id">{n.policy_id}</span>
                ) : null}
              </span>
              <div class="actionable-item-actions">
                <button
                  type="button"
                  class="btn btn-sm btn-dismiss"
                  title="Dismiss"
                  aria-label="Dismiss notification"
                  onClick={(e) => dismissActionable(e, n.notification_id)}
                >
                  ✕
                </button>
              </div>
            </div>
          ))}
          {metrics.pendingReviews === 0 && actionableUnread.length === 0 ? (
            <p class="dashboard-muted-hint">No pending items right now.</p>
          ) : null}
        </div>
      </div>

      <div class="dashboard-section">
        <h3>Recent notifications</h3>
        {feedNotifs.length === 0 ? (
          <p class="dashboard-muted-hint">No notifications yet.</p>
        ) : (
          <div class="dashboard-notification-feed">
            {feedNotifs.map((n) => {
              const sev = notificationSeverity(n);
              return (
                <div
                  key={n.notification_id}
                  class={`dashboard-notif-feed-row ${n.read ? "" : "unread"}`}
                  onClick={() => onFeedItemActivate(n)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={cardKeyHandler(() => onFeedItemActivate(n))}
                >
                  <div class="dashboard-notif-feed-main">
                    <p class="dashboard-notif-feed-message">
                      {notificationMessage(n)}
                    </p>
                    <span class="dashboard-notif-feed-meta">
                      {n.policy_id ? `${n.policy_id} · ` : ""}
                      {fmtDateTime(n.created_at)}
                    </span>
                  </div>
                  <span
                    class={`dashboard-severity-badge ${severityBadgeClass(sev)}`}
                  >
                    {sev}
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}
