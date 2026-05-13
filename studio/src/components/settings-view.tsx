// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useCallback } from "preact/hooks";
import { currentUser, navigate } from "../app";
import { apiFetch } from "../api/fetch";
import { fetchMe } from "../api/auth";
import { fmtDateTime, displayName, registerNames } from "../lib/format";

interface UserRow {
  email: string;
  name: string;
  avatar_url: string;
  role: string;
  created_at: string;
}

interface RoleChangeRow {
  changed_by: string;
  target_email: string;
  old_role: string;
  new_role: string;
  changed_at: string;
}

type SettingsTab = "users" | "audit-log";

export function SettingsView() {
  const [users, setUsers] = useState<UserRow[]>([]);
  const [changes, setChanges] = useState<RoleChangeRow[]>([]);
  const [tab, setTab] = useState<SettingsTab>("users");
  const [updating, setUpdating] = useState<string | null>(null);
  const me = currentUser.value;

  if (me?.role !== "admin") {
    navigate("posture");
    return null;
  }

  const fetchData = useCallback(() => {
    apiFetch("/api/users").then((r) => r.json()).then((data: UserRow[]) => {
      setUsers(data);
      registerNames(data.map((u) => ({ email: u.email, name: u.name })));
    }).catch(() => {});
    apiFetch("/api/role-changes").then((r) => r.json()).then(setChanges).catch(() => {});
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const toggleRole = async (email: string, currentRole: string) => {
    const newRole = currentRole === "admin" ? "reviewer" : "admin";
    setUpdating(email);
    try {
      const res = await apiFetch(`/api/users/${encodeURIComponent(email)}/role`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role: newRole }),
      });
      if (res.ok) {
        fetchData();
        const me = await fetchMe();
        currentUser.value = me;
      }
    } finally {
      setUpdating(null);
    }
  };

  const admins = users.filter((u) => u.role === "admin");
  const reviewers = users.filter((u) => u.role === "reviewer");

  const nameByEmail = new Map(users.map((u) => [u.email, u.name]));
  const resolveName = (email: string) => nameByEmail.get(email) || displayName(email);

  return (
    <section class="settings-view">
      <header class="settings-header">
        <h2>Settings</h2>
      </header>

      <div class="settings-layout">
        <nav class="settings-nav">
          <button class={`settings-nav-item ${tab === "users" ? "active" : ""}`} onClick={() => setTab("users")}>
            Users
          </button>
          <button class={`settings-nav-item ${tab === "audit-log" ? "active" : ""}`} onClick={() => setTab("audit-log")}>
            Audit Log
          </button>
        </nav>

        <div class="settings-content">
          {tab === "users" && (
            <div class="settings-panel">
              <h3 class="settings-section-title">Admins ({admins.length})</h3>
              <p class="settings-section-desc">
                Admins can manage users, change roles, and access all settings.
              </p>
              {admins.length === 0 ? (
                <div class="settings-empty">No admins configured.</div>
              ) : (
                <ul class="member-list">
                  {admins.map((u) => (
                    <li key={u.email}>
                      <div class="member-identity">
                        <div class="member-name">
                          {u.avatar_url && <img class="user-avatar" src={u.avatar_url} alt="" width="24" height="24" referrerpolicy="no-referrer" />}
                          {u.name || u.email.split("@")[0]}
                          <span class="role-badge role-admin">admin</span>
                          {u.email === me?.email && <span class="member-you-tag">you</span>}
                        </div>
                        <div class="member-email">{u.email}</div>
                      </div>
                      {u.email !== me?.email && (
                        <button
                          class="revoke-btn"
                          disabled={updating === u.email}
                          onClick={() => toggleRole(u.email, u.role)}
                        >
                          {updating === u.email ? "..." : "Demote"}
                        </button>
                      )}
                    </li>
                  ))}
                </ul>
              )}

              <h3 class="settings-section-title">Reviewers ({reviewers.length})</h3>
              <p class="settings-section-desc">
                Reviewers have read-only access to audits, evidence, and policies.
              </p>
              {reviewers.length === 0 ? (
                <div class="settings-empty">No reviewers yet.</div>
              ) : (
                <ul class="member-list">
                  {reviewers.map((u) => (
                    <li key={u.email}>
                      <div class="member-identity">
                        <div class="member-name">
                          {u.avatar_url && <img class="user-avatar" src={u.avatar_url} alt="" width="24" height="24" referrerpolicy="no-referrer" />}
                          {u.name || u.email.split("@")[0]}
                          <span class="role-badge role-reviewer">reviewer</span>
                        </div>
                        <div class="member-email">{u.email}</div>
                      </div>
                      <button
                        class="promote-btn"
                        disabled={updating === u.email}
                        onClick={() => toggleRole(u.email, u.role)}
                      >
                        {updating === u.email ? "..." : "Promote"}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {tab === "audit-log" && (
            <div class="settings-panel">
              <h3 class="settings-section-title">Role Changes ({changes.length})</h3>
              <p class="settings-section-desc">
                Immutable log of every role change in the system.
              </p>
              {changes.length === 0 ? (
                <div class="settings-empty">No role changes recorded.</div>
              ) : (
                <ul class="member-list">
                  {changes.map((c, i) => (
                    <li key={i}>
                      <div class="member-identity">
                        <div class="member-name">{resolveName(c.target_email)}</div>
                        <div class="member-email">
                          {resolveName(c.changed_by)} changed role on {fmtDateTime(c.changed_at)}
                        </div>
                      </div>
                      <div class="role-change-badges">
                        <span class={`role-badge role-${c.old_role}`}>{c.old_role}</span>
                        <span class="role-arrow">&rarr;</span>
                        <span class={`role-badge role-${c.new_role}`}>{c.new_role}</span>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
