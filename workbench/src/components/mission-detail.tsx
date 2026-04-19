// SPDX-License-Identifier: Apache-2.0
import { useEffect, useRef, useState } from "preact/hooks";
import { navigate } from "../app";
import { getJob, updateJob, addArtifact, addMessage, jobsList, getJobAgent,
  updateLastAgentMessage, finalizeLastAgentMessage,
  addToolCallMessage, updateToolCall } from "../store/jobs";
import { streamMessage, streamReply, type StreamCallbacks } from "../api/a2a";
import { extractArtifacts, detectDefinition } from "../lib/artifact-detect";
import { StatusBadge } from "./status-badge";
import { ChatPanel } from "./chat-panel";
import { ArtifactPanel } from "./artifact-panel";

const KAGENT_PREFIX = "kagent.dev/";
const PARTIAL_KEY = `${KAGENT_PREFIX}adk_partial`;
const TYPE_KEY = `${KAGENT_PREFIX}type`;
const LONG_RUNNING_KEY = `${KAGENT_PREFIX}is_long_running`;

function getMetaKey(meta: Record<string, unknown> | undefined, key: string): unknown {
  if (!meta) return undefined;
  return meta[key];
}

export function JobDetail({ jobId }: { jobId: string }) {
  const _trigger = jobsList.value;
  const job = getJob(jobId);
  const streamRef = useRef<(() => void) | null>(null);
  const streamingBuffer = useRef("");
  const [, forceUpdate] = useState(0);

  function mapState(state: string): string {
    return state === "completed" ? "ready" : state;
  }

  function buildCallbacks(): StreamCallbacks {
    return {
      onTaskId(taskId) {
        updateJob(jobId, { taskId, status: "working" });
        forceUpdate((n) => n + 1);
      },
      onStatus(state) {
        const mapped = mapState(state);
        updateJob(jobId, { status: mapped });
        if (mapped === "ready" && streamingBuffer.current) {
          finalizeLastAgentMessage(jobId);
          streamingBuffer.current = "";
        }
        forceUpdate((n) => n + 1);
      },
      onMessage(message) {
        if (!message?.parts) return;
        for (const part of message.parts as Array<{
          kind?: string; type?: string; text?: string;
          data?: Record<string, unknown>;
          metadata?: Record<string, unknown>;
        }>) {
          const meta = part.metadata;
          const partType = getMetaKey(meta, TYPE_KEY) as string | undefined;
          const isPartial = getMetaKey(meta, PARTIAL_KEY) === true;

          if (partType === "function_call" && part.data) {
            if (streamingBuffer.current) {
              finalizeLastAgentMessage(jobId);
              streamingBuffer.current = "";
            }
            const toolId = (part.data.id as string) || crypto.randomUUID();
            const isLongRunning = getMetaKey(meta, LONG_RUNNING_KEY) === true;
            addToolCallMessage(jobId, {
              name: (part.data.name as string) || "unknown",
              id: toolId,
              args: part.data.args as Record<string, unknown> | undefined,
              status: isLongRunning ? "pending" : "executing",
            });
            forceUpdate((n) => n + 1);
            continue;
          }

          if (partType === "function_response" && part.data) {
            const toolId = (part.data.id as string) || "";
            const resultText = typeof part.data.response === "string"
              ? part.data.response
              : JSON.stringify(part.data.response ?? "").slice(0, 200);
            updateToolCall(jobId, toolId, { status: "completed", result: resultText });
            forceUpdate((n) => n + 1);
            continue;
          }

          if ((part.kind || part.type) === "text" && part.text) {
            if (isPartial) {
              streamingBuffer.current += part.text;
              updateLastAgentMessage(jobId, streamingBuffer.current);
              forceUpdate((n) => n + 1);
            } else {
              if (streamingBuffer.current) {
                streamingBuffer.current += part.text;
                updateLastAgentMessage(jobId, streamingBuffer.current);
                finalizeLastAgentMessage(jobId);
                streamingBuffer.current = "";
              } else {
                const extracted = extractArtifacts(part.text);
                if (extracted.text.trim()) addMessage(jobId, "agent", extracted.text);
                for (const artifact of extracted.artifacts) {
                  addArtifact(jobId, artifact.name, artifact.yaml, artifact.definition ?? undefined);
                }
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
              addArtifact(jobId, name, part.text, detectDefinition(part.text) ?? undefined);
              forceUpdate((n) => n + 1);
            }
          }
        }
      },
      onError() { updateJob(jobId, { status: "disconnected" }); forceUpdate((n) => n + 1); },
      onDone(state) {
        const mapped = mapState(state);
        if (streamingBuffer.current) {
          finalizeLastAgentMessage(jobId);
          streamingBuffer.current = "";
        }
        updateJob(jobId, { status: mapped });
        forceUpdate((n) => n + 1);
        streamRef.current = null;
      },
    };
  }

  useEffect(() => {
    if (!job) return;
    if (["ready", "accepted", "cancelled", "failed", "disconnected"].includes(job.status)) return;
    if (streamRef.current) return;

    const agentName = getJobAgent(job);
    streamingBuffer.current = "";

    if (!job.taskId) {
      const userMsg = job.messages.find((m) => m.role === "user");
      if (!userMsg) return;
      const cleanup = streamMessage(userMsg.content, buildCallbacks(), agentName);
      streamRef.current = cleanup;
      return () => { cleanup(); streamRef.current = null; };
    }

    const cleanup = streamReply(job.taskId, "", buildCallbacks(), agentName);
    streamRef.current = cleanup;
    return () => { cleanup(); streamRef.current = null; };
  }, [jobId]);

  if (!job) { navigate("jobs"); return null; }

  const replyAgent = getJobAgent(job);

  function handleReply(text: string) {
    addMessage(jobId, "user", text);
    updateJob(jobId, { status: "working" });
    forceUpdate((n) => n + 1);

    if (streamRef.current) { streamRef.current(); streamRef.current = null; }
    streamingBuffer.current = "";

    const taskId = job!.taskId || jobId;
    const cleanup = streamReply(taskId, text, buildCallbacks(), replyAgent);
    streamRef.current = cleanup;
  }

  return (
    <div class="detail-view">
      <div class="detail-back">
        <button class="btn btn-secondary btn-sm" onClick={() => navigate("jobs")}>&larr; Back</button>
        <span class="job-title">{job.title}</span>
        <StatusBadge status={job.status} />
      </div>
      <div class="detail-panes">
        <ChatPanel messages={job.messages} status={job.status} onReply={handleReply} />
        <ArtifactPanel artifacts={job.artifacts} jobId={jobId} />
      </div>
    </div>
  );
}
