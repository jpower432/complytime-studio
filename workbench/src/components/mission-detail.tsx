// SPDX-License-Identifier: Apache-2.0
import { useEffect, useRef, useState } from "preact/hooks";
import { navigate } from "../app";
import { getMission, updateMission, addArtifact, addMessage, missionsList, getMissionAgent } from "../store/missions";
import { streamTask, sendReply } from "../api/a2a";
import { extractArtifacts, detectDefinition } from "../lib/artifact-detect";
import { StatusBadge } from "./status-badge";
import { ChatPanel } from "./chat-panel";
import { ArtifactPanel } from "./artifact-panel";

export function MissionDetail({ missionId }: { missionId: string }) {
  const _trigger = missionsList.value;
  const mission = getMission(missionId);
  const streamRef = useRef<(() => void) | null>(null);
  const [, forceUpdate] = useState(0);
  useEffect(() => {
    if (!mission) return;
    if (["completed", "failed", "disconnected"].includes(mission.status)) return;
    const agentName = getMissionAgent(mission);
    const cleanup = streamTask(missionId, {
      onStatus(state) { updateMission(missionId, { status: state }); forceUpdate((n) => n + 1); },
      onMessage(message) {
        if (message?.parts) {
          for (const part of message.parts as Array<{ type: string; text: string }>) {
            if (part.type === "text") {
              const extracted = extractArtifacts(part.text);
              if (extracted.text.trim()) addMessage(missionId, "agent", extracted.text);
              for (const artifact of extracted.artifacts) addArtifact(missionId, artifact.name, artifact.yaml, artifact.definition);
              forceUpdate((n) => n + 1);
            }
          }
        }
      },
      onArtifact(artifact) {
        const parts = (artifact as { parts?: Array<{ type: string; text: string }>; name?: string }).parts;
        if (parts) { for (const part of parts) { if (part.type === "text") { const name = (artifact as { name?: string }).name || "artifact.yaml"; addArtifact(missionId, name, part.text, detectDefinition(part.text)); forceUpdate((n) => n + 1); } } }
      },
      onError() { updateMission(missionId, { status: "disconnected" }); forceUpdate((n) => n + 1); },
      onDone(state) { updateMission(missionId, { status: state }); forceUpdate((n) => n + 1); streamRef.current = null; },
    }, agentName);
    streamRef.current = cleanup;
    return () => { cleanup(); streamRef.current = null; };
  }, [missionId]);
  if (!mission) { navigate("missions"); return null; }
  const replyAgent = getMissionAgent(mission);
  async function handleReply(text: string) {
    addMessage(missionId, "user", text); forceUpdate((n) => n + 1);
    try { await sendReply(missionId, text, replyAgent); }
    catch (err) { addMessage(missionId, "system", `Error: ${(err as Error).message}`); forceUpdate((n) => n + 1); }
  }
  return (
    <div class="detail-view">
      <div class="detail-back">
        <button class="btn btn-secondary btn-sm" onClick={() => navigate("missions")}>&larr; Back</button>
        <span class="mission-title">{mission.title}</span>
        <StatusBadge status={mission.status} />
      </div>
      <div class="detail-panes">
        <ChatPanel messages={mission.messages} status={mission.status} onReply={handleReply} />
        <ArtifactPanel artifacts={mission.artifacts} missionId={missionId} />
      </div>
    </div>
  );
}
