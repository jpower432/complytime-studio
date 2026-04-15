// SPDX-License-Identifier: Apache-2.0
import { signal } from "@preact/signals";
import { navigate } from "../app";
import { missionsList, hasActiveMission, createMission, addMessage, timeAgo } from "../store/missions";
import { sendMessage } from "../api/a2a";
import { StatusBadge } from "./status-badge";

const showNewDialog = signal(false);

export function MissionsView() {
  const missions = missionsList.value;
  const active = hasActiveMission();
  return (
    <div class="missions-view">
      <div class="missions-header">
        <h2>Missions</h2>
        <button class="btn btn-primary" disabled={active} title={active ? "Complete the active mission first" : undefined} onClick={() => (showNewDialog.value = true)}>+ New Mission</button>
      </div>
      {missions.length === 0 ? (
        <div class="empty-state">
          <h3>No missions yet</h3>
          <p>Start a mission to analyze threats, author controls, or manage Gemara artifacts.</p>
          <button class="btn btn-primary" onClick={() => (showNewDialog.value = true)}>+ New Mission</button>
        </div>
      ) : (
        missions.map((m) => (
          <div key={m.id} class="mission-card" onClick={() => navigate("detail", m.id)}>
            <div class="mission-card-header">
              <span class="mission-card-title">{m.title}</span>
              <StatusBadge status={m.status} />
            </div>
            <div class="mission-card-meta">
              <span>{m.artifacts.length} artifact{m.artifacts.length !== 1 ? "s" : ""}</span>
              <span>{timeAgo(m.createdAt)}</span>
            </div>
          </div>
        ))
      )}
      {showNewDialog.value && <NewMissionDialog onClose={() => (showNewDialog.value = false)} />}
    </div>
  );
}

function NewMissionDialog({ onClose }: { onClose: () => void }) {
  const error = signal("");
  const submitting = signal(false);
  let textareaRef: HTMLTextAreaElement | null = null;
  async function handleSubmit() {
    const text = textareaRef?.value.trim();
    if (!text) { error.value = "Describe what you want the agent to do."; return; }
    submitting.value = true;
    try {
      const result = await sendMessage(text);
      const taskId = result?.result?.id || result?.id || crypto.randomUUID();
      const mission = createMission(taskId, text);
      addMessage(taskId, "user", text);
      onClose();
      navigate("detail", mission.id);
    } catch (e: unknown) { error.value = `Failed: ${(e as Error).message}`; submitting.value = false; }
  }
  return (
    <div class="dialog-overlay" onClick={onClose}>
      <div class="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>New Mission</h3>
        <textarea ref={(el) => { textareaRef = el; el?.focus(); }} placeholder='Describe what you want to do, e.g. "Analyze threats for github.com/kyverno/kyverno"' onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); handleSubmit(); } }} />
        {error.value && <div class="dialog-error">{error.value}</div>}
        <div class="dialog-actions">
          <button class="btn btn-secondary" onClick={onClose}>Cancel</button>
          <button class="btn btn-primary" disabled={submitting.value} onClick={handleSubmit}>{submitting.value ? "Starting..." : "Start Mission"}</button>
        </div>
      </div>
    </div>
  );
}
