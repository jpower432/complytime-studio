// SPDX-License-Identifier: Apache-2.0
import { signal } from "@preact/signals";
import { useState, useEffect } from "preact/hooks";
import { navigate } from "../app";
import { missionsList, hasActiveMission, createMission, addMessage, timeAgo } from "../store/missions";
import { sendMessage } from "../api/a2a";
import { fetchAgents, type AgentCard } from "../api/agents";
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
        </div>
      ) : (
        missions.map((m) => (
          <div key={m.id} class="mission-card" onClick={() => navigate("workspace", m.id)}>
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

const DEFAULT_AGENT = "studio-threat-modeler";

function NewMissionDialog({ onClose }: { onClose: () => void }) {
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [agents, setAgents] = useState<AgentCard[]>([]);
  const [selectedAgent, setSelectedAgent] = useState(DEFAULT_AGENT);
  let textareaRef: HTMLTextAreaElement | null = null;

  useEffect(() => {
    fetchAgents()
      .then((cards) => {
        setAgents(cards);
        if (cards.length > 0) setSelectedAgent(cards[0].name);
      })
      .catch(() => {});
  }, []);

  async function handleSubmit() {
    const text = textareaRef?.value.trim();
    if (!text) { setError("Describe what you want the agent to do."); return; }
    setSubmitting(true);
    setError("");
    try {
      const result = await sendMessage(text, selectedAgent);
      const taskId = result?.result?.id || result?.id || crypto.randomUUID();
      const mission = createMission(taskId, text, selectedAgent);
      addMessage(taskId, "user", text);
      onClose();
      navigate("workspace", mission.id);
    } catch (e: unknown) { setError(`Failed: ${(e as Error).message}`); setSubmitting(false); }
  }

  return (
    <div class="dialog-overlay" onClick={onClose}>
      <div class="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>New Mission</h3>
        {agents.length > 0 && (
          <div class="agent-picker">
            <label class="dialog-label">Specialist</label>
            {agents.map((a) => (
              <div key={a.name} class={`agent-card ${selectedAgent === a.name ? "selected" : ""}`} onClick={() => setSelectedAgent(a.name)}>
                <span class="agent-card-name">{a.name}</span>
                <span class="agent-card-desc">{a.description}</span>
              </div>
            ))}
          </div>
        )}
        <textarea ref={(el) => { textareaRef = el; if (el && !agents.length) el.focus(); }} placeholder='Describe what you want to do, e.g. "Analyze threats for github.com/kyverno/kyverno"' onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); handleSubmit(); } }} />
        {error && <div class="dialog-error">{error}</div>}
        <div class="dialog-actions">
          <button class="btn btn-secondary" onClick={onClose}>Cancel</button>
          <button class="btn btn-primary" disabled={submitting} onClick={handleSubmit}>{submitting ? "Starting..." : "Start Mission"}</button>
        </div>
      </div>
    </div>
  );
}
