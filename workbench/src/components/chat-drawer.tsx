// SPDX-License-Identifier: Apache-2.0

import { useEffect, useRef, useState } from "preact/hooks";
import { getMissionAgent, updateMission, addArtifact, addMessage, missionsList, type Mission } from "../store/missions";
import { setEditorArtifact } from "../store/editor";
import { streamTask, sendReply } from "../api/a2a";
import { extractArtifacts, detectDefinition } from "../lib/artifact-detect";
import { ChatPanel } from "./chat-panel";
import { StatusBadge } from "./status-badge";

interface ChatDrawerProps {
  mission: Mission;
  onClose: () => void;
}

export function ChatDrawer({ mission, onClose }: ChatDrawerProps) {
  const _trigger = missionsList.value;
  const streamRef = useRef<(() => void) | null>(null);
  const [, forceUpdate] = useState(0);

  useEffect(() => {
    if (["completed", "failed", "disconnected"].includes(mission.status)) return;
    if (streamRef.current) return;

    const agentName = getMissionAgent(mission);
    const cleanup = streamTask(mission.id, {
      onStatus(state) { updateMission(mission.id, { status: state }); forceUpdate((n) => n + 1); },
      onMessage(message) {
        if (message?.parts) {
          for (const part of message.parts as Array<{ kind?: string; type?: string; text: string }>) {
            if ((part.kind || part.type) === "text") {
              const extracted = extractArtifacts(part.text);
              if (extracted.text.trim()) addMessage(mission.id, "agent", extracted.text);
              for (const artifact of extracted.artifacts) {
                addArtifact(mission.id, artifact.name, artifact.yaml, artifact.definition);
                setEditorArtifact(artifact.name, artifact.yaml, artifact.definition);
              }
              forceUpdate((n) => n + 1);
            }
          }
        }
      },
      onArtifact(artifact) {
        const parts = (artifact as { parts?: Array<{ kind?: string; type?: string; text: string }>; name?: string }).parts;
        if (parts) {
          for (const part of parts) {
            if ((part.kind || part.type) === "text") {
              const name = (artifact as { name?: string }).name || "artifact.yaml";
              const definition = detectDefinition(part.text);
              addArtifact(mission.id, name, part.text, definition);
              setEditorArtifact(name, part.text, definition);
              forceUpdate((n) => n + 1);
            }
          }
        }
      },
      onError() { updateMission(mission.id, { status: "disconnected" }); forceUpdate((n) => n + 1); },
      onDone(state) { updateMission(mission.id, { status: state }); forceUpdate((n) => n + 1); streamRef.current = null; },
    }, agentName);
    streamRef.current = cleanup;
    return () => { cleanup(); streamRef.current = null; };
  }, [mission.id]);

  const replyAgent = getMissionAgent(mission);
  async function handleReply(text: string) {
    addMessage(mission.id, "user", text);
    forceUpdate((n) => n + 1);
    try { await sendReply(mission.id, text, replyAgent); }
    catch (err) { addMessage(mission.id, "system", `Error: ${(err as Error).message}`); forceUpdate((n) => n + 1); }
  }

  const currentMission = missionsList.value.find((m) => m.id === mission.id) ?? mission;

  return (
    <div class="chat-drawer">
      <div class="chat-drawer-header">
        <span class="chat-drawer-title">{currentMission.title}</span>
        <StatusBadge status={currentMission.status} />
        <button class="btn btn-secondary btn-sm chat-drawer-close" onClick={onClose}>&times;</button>
      </div>
      <ChatPanel messages={currentMission.messages} status={currentMission.status} onReply={handleReply} />
    </div>
  );
}
