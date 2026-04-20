// SPDX-License-Identifier: Apache-2.0

import { useEffect, useRef, useState, useCallback } from "preact/hooks";
import {
  getJobAgent, updateJob, addArtifact, addMessage, cancelJob,
  updateLastAgentMessage, finalizeLastAgentMessage,
  addToolCallMessage, updateToolCall, jobsList, type Job, type Message,
} from "../store/jobs";
import { proposeArtifact } from "../store/editor";
import { getArtifactByName } from "../store/workspace";
import {
  streamMessage, streamReply, type StreamCallbacks, type ContextArtifact, type StreamReplyOptions,
} from "../api/a2a";
import { extractArtifacts, detectDefinition, stripFences } from "../lib/artifact-detect";
import { ChatPanel } from "./chat-panel";
import { StatusBadge } from "./status-badge";

const KAGENT_PREFIX = "kagent.dev/";
const PARTIAL_KEY = `${KAGENT_PREFIX}adk_partial`;
const TYPE_KEY = `${KAGENT_PREFIX}type`;
const LONG_RUNNING_KEY = `${KAGENT_PREFIX}is_long_running`;

function getMetaKey(meta: Record<string, unknown> | undefined, key: string): unknown {
  if (!meta) return undefined;
  return meta[key];
}

interface ChatDrawerProps {
  job: Job;
  onClose: () => void;
}

const MAX_POLL_RETRIES = 60;
const POLL_INTERVAL_MS = 5000;

const MAX_REPLAY_HISTORY_CHARS = 100_000;
const PRESERVE_LAST_MESSAGE_LINES = 4;

/** Drop oldest serialized lines until under the cap; keep the last four lines when trimming. */
function truncateHistory(serializedLines: string[]): { inner: string; truncated: boolean } {
  let lines = [...serializedLines];
  let truncatedFromCount = false;
  while (lines.join("\n\n").length > MAX_REPLAY_HISTORY_CHARS && lines.length > PRESERVE_LAST_MESSAGE_LINES) {
    lines.shift();
    truncatedFromCount = true;
  }
  let inner = lines.join("\n\n");
  const marker = "[Earlier conversation truncated]\n\n";
  if (inner.length > MAX_REPLAY_HISTORY_CHARS) {
    inner = marker + inner.slice(-(MAX_REPLAY_HISTORY_CHARS - marker.length));
    return { inner, truncated: true };
  }
  if (truncatedFromCount) {
    inner = marker + inner;
    return { inner, truncated: true };
  }
  return { inner, truncated: false };
}

function messageLineForReplay(msg: Message): string | null {
  if (msg.partial) return null;
  if (msg.toolCall) {
    return `[Agent]: [Tool: ${msg.toolCall.name}]`;
  }
  if (msg.role === "user") {
    return `[User]: ${msg.content}`;
  }
  return `[Agent]: ${msg.content}`;
}

function buildReplayHistoryBlock(messages: Message[]): string | undefined {
  const lines = messages.map(messageLineForReplay).filter((l): l is string => Boolean(l?.trim()));
  if (!lines.length) return undefined;
  const { inner } = truncateHistory(lines);
  if (!inner.trim()) return undefined;
  return `<conversation-history>\n${inner}\n</conversation-history>`;
}

function buildContextArtifactsForJob(job: Job): ContextArtifact[] | undefined {
  if (!job.contextArtifacts?.length) return undefined;
  const ctx = job.contextArtifacts
    .map((name) => {
      const a = getArtifactByName(name);
      return a ? { name: a.name, yaml: a.yaml } : null;
    })
    .filter((a): a is ContextArtifact => a !== null);
  return ctx.length ? ctx : undefined;
}

export function ChatDrawer({ job, onClose }: ChatDrawerProps) {
  const _trigger = jobsList.value;
  const streamRef = useRef<(() => void) | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollCountRef = useRef(0);
  const streamingBuffer = useRef("");
  const [, forceUpdate] = useState(0);

  function mapState(state: string): string {
    return state === "completed" ? "ready" : state;
  }

  function finalizeStreamedContent() {
    if (!streamingBuffer.current) return;
    const content = streamingBuffer.current;
    streamingBuffer.current = "";
    finalizeLastAgentMessage(job.id);

    const extracted = extractArtifacts(content);
    for (const artifact of extracted.artifacts) {
      addArtifact(job.id, artifact.name, artifact.yaml, artifact.definition ?? undefined);
      proposeArtifact(artifact.name, artifact.yaml, artifact.definition ?? undefined);
    }
  }

  function buildCallbacks(): StreamCallbacks {
    return {
      onTaskId(taskId) {
        updateJob(job.id, { taskId, status: "working" });
        forceUpdate((n) => n + 1);
      },
      onStatus(state) {
        const mapped = mapState(state);
        updateJob(job.id, { status: mapped });
        if (mapped === "ready") finalizeStreamedContent();
        forceUpdate((n) => n + 1);
      },
      onMessage(message) {
        if (!message?.parts) return;
        if (message.role === "user") return;
        for (const part of message.parts as Array<{
          kind?: string; type?: string; text?: string;
          data?: Record<string, unknown>;
          metadata?: Record<string, unknown>;
        }>) {
          const meta = part.metadata;
          const partType = getMetaKey(meta, TYPE_KEY) as string | undefined;
          const isPartial = getMetaKey(meta, PARTIAL_KEY) === true;

          if (partType === "function_call" && part.data) {
            finalizeStreamedContent();
            const toolId = (part.data.id as string) || crypto.randomUUID();
            const isLongRunning = getMetaKey(meta, LONG_RUNNING_KEY) === true;
            addToolCallMessage(job.id, {
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
            updateToolCall(job.id, toolId, { status: "completed", result: resultText });
            forceUpdate((n) => n + 1);
            continue;
          }

          if (part.text && (!part.kind && !part.type || (part.kind || part.type) === "text")) {
            if (isPartial) {
              streamingBuffer.current += part.text;
              updateLastAgentMessage(job.id, streamingBuffer.current);
              forceUpdate((n) => n + 1);
            } else {
              if (streamingBuffer.current) {
                streamingBuffer.current += part.text;
                updateLastAgentMessage(job.id, streamingBuffer.current);
                finalizeStreamedContent();
              } else {
                const extracted = extractArtifacts(part.text);
                if (extracted.text.trim()) addMessage(job.id, "agent", extracted.text);
                for (const artifact of extracted.artifacts) {
                  addArtifact(job.id, artifact.name, artifact.yaml, artifact.definition ?? undefined);
                  proposeArtifact(artifact.name, artifact.yaml, artifact.definition ?? undefined);
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
            if (part.text && (!part.kind && !part.type || (part.kind || part.type) === "text")) {
              const extracted = extractArtifacts(part.text);
              if (extracted.artifacts.length > 0) {
                for (const a of extracted.artifacts) {
                  addArtifact(job.id, a.name, a.yaml, a.definition ?? undefined);
                  proposeArtifact(a.name, a.yaml, a.definition ?? undefined);
                }
              } else {
                const name = (artifact as { name?: string }).name || "artifact.yaml";
                const definition = detectDefinition(part.text) ?? undefined;
                const yaml = stripFences(part.text);
                addArtifact(job.id, name, yaml, definition);
                proposeArtifact(name, yaml, definition);
              }
              forceUpdate((n) => n + 1);
            }
          }
        }
      },
      onError() {
        updateJob(job.id, { status: "disconnected" });
        if (pollTimerRef.current) { clearTimeout(pollTimerRef.current); pollTimerRef.current = null; }
        forceUpdate((n) => n + 1);
      },
      onDone(state) {
        const mapped = mapState(state);
        finalizeStreamedContent();
        updateJob(job.id, { status: mapped });
        if (pollTimerRef.current) { clearTimeout(pollTimerRef.current); pollTimerRef.current = null; }
        forceUpdate((n) => n + 1);
        streamRef.current = null;
      },
    };
  }

  function startPollRetry() {
    if (pollTimerRef.current) return;
    pollCountRef.current = 0;

    const tick = () => {
      const latest = jobsList.value.find((j) => j.id === job.id);
      if (!latest || !["submitted", "working"].includes(latest.status)) {
        pollTimerRef.current = null;
        return;
      }
      if (pollCountRef.current >= MAX_POLL_RETRIES) {
        console.warn("[chat] poll retry limit reached");
        updateJob(job.id, { status: "disconnected" });
        forceUpdate((n) => n + 1);
        pollTimerRef.current = null;
        return;
      }
      pollCountRef.current++;
      console.log("[chat] poll retry", pollCountRef.current, "for task", latest.taskId);

      if (streamRef.current) { streamRef.current(); streamRef.current = null; }
      const agentName = getJobAgent(latest);
      const taskId = latest.taskId || latest.id;
      const cleanup = streamReply(taskId, "", buildCallbacks(), agentName);
      streamRef.current = cleanup;
      pollTimerRef.current = setTimeout(tick, POLL_INTERVAL_MS);
    };

    pollTimerRef.current = setTimeout(tick, POLL_INTERVAL_MS);
  }

  useEffect(() => {
    if (["ready", "accepted", "cancelled", "failed", "disconnected"].includes(job.status)) return;
    if (streamRef.current) return;

    const agentName = getJobAgent(job);
    streamingBuffer.current = "";
    pollCountRef.current = 0;

    const currentJob = jobsList.value.find((j) => j.id === job.id) ?? job;

    const callbacks: StreamCallbacks = {
      ...buildCallbacks(),
      onDone(state) {
        buildCallbacks().onDone?.(state);
      },
    };

    if (!currentJob.taskId) {
      const userMsg = currentJob.messages.find((m) => m.role === "user");
      if (!userMsg) return;
      const context = buildContextArtifactsForJob(currentJob);
      const cleanup = streamMessage(userMsg.content, callbacks, agentName, context);
      streamRef.current = cleanup;
      startPollRetry();
      return () => {
        cleanup(); streamRef.current = null;
        if (pollTimerRef.current) { clearTimeout(pollTimerRef.current); pollTimerRef.current = null; }
      };
    }

    const cleanup = streamReply(currentJob.taskId, "", callbacks, agentName);
    streamRef.current = cleanup;
    startPollRetry();
    return () => {
      cleanup(); streamRef.current = null;
      if (pollTimerRef.current) { clearTimeout(pollTimerRef.current); pollTimerRef.current = null; }
    };
  }, [job.id]);

  const drawerRef = useRef<HTMLDivElement>(null);
  const [drawerWidth, setDrawerWidth] = useState<number | null>(null);

  const handleResizeStart = useCallback((e: MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = drawerRef.current?.offsetWidth ?? 400;

    const onMove = (ev: MouseEvent) => {
      const delta = startX - ev.clientX;
      const newWidth = Math.max(280, Math.min(startWidth + delta, window.innerWidth * 0.8));
      setDrawerWidth(newWidth);
    };
    const onUp = () => {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  }, []);

  const currentJob = jobsList.value.find((j) => j.id === job.id) ?? job;
  const replyAgent = getJobAgent(currentJob);

  function handleReply(text: string) {
    addMessage(job.id, "user", text);
    updateJob(job.id, { status: "working" });
    forceUpdate((n) => n + 1);

    if (streamRef.current) { streamRef.current(); streamRef.current = null; }
    streamingBuffer.current = "";

    const after = jobsList.value.find((j) => j.id === job.id) ?? currentJob;
    const prior = after.messages.slice(0, -1);
    const history = buildReplayHistoryBlock(prior);
    const context = buildContextArtifactsForJob(after);
    const replyOpts: StreamReplyOptions = {};
    if (history) replyOpts.history = history;
    if (context) replyOpts.context = context;

    const taskId = after.taskId || job.id;
    const cleanup = streamReply(taskId, text, buildCallbacks(), replyAgent,
      Object.keys(replyOpts).length ? replyOpts : undefined);
    streamRef.current = cleanup;
  }

  function handleCancel() {
    if (pollTimerRef.current) { clearTimeout(pollTimerRef.current); pollTimerRef.current = null; }
    if (streamRef.current) { streamRef.current(); streamRef.current = null; }
    cancelJob(job.id);
    forceUpdate((n) => n + 1);
  }

  function handleApprove(toolCallId: string) {
    updateToolCall(job.id, toolCallId, { status: "approved" });
    forceUpdate((n) => n + 1);

    if (streamRef.current) { streamRef.current(); streamRef.current = null; }
    streamingBuffer.current = "";

    const after = jobsList.value.find((j) => j.id === job.id) ?? currentJob;
    const history = buildReplayHistoryBlock(after.messages);
    const context = buildContextArtifactsForJob(after);
    const replyOpts: StreamReplyOptions = {};
    if (history) replyOpts.history = history;
    if (context) replyOpts.context = context;

    const taskId = after.taskId || job.id;
    const cleanup = streamReply(taskId, "", buildCallbacks(), replyAgent,
      Object.keys(replyOpts).length ? replyOpts : undefined);
    streamRef.current = cleanup;
  }

  function handleReject(toolCallId: string) {
    updateToolCall(job.id, toolCallId, { status: "rejected" });
    forceUpdate((n) => n + 1);

    if (streamRef.current) { streamRef.current(); streamRef.current = null; }
    streamingBuffer.current = "";

    const after = jobsList.value.find((j) => j.id === job.id) ?? currentJob;
    const history = buildReplayHistoryBlock(after.messages);
    const context = buildContextArtifactsForJob(after);
    const replyOpts: StreamReplyOptions = {};
    if (history) replyOpts.history = history;
    if (context) replyOpts.context = context;

    const taskId = after.taskId || job.id;
    const cleanup = streamReply(taskId, "rejected", buildCallbacks(), replyAgent,
      Object.keys(replyOpts).length ? replyOpts : undefined);
    streamRef.current = cleanup;
  }

  return (
    <div class="chat-drawer" ref={drawerRef} style={drawerWidth ? { width: `${drawerWidth}px` } : undefined}>
      <div class="chat-drawer-resize" onMouseDown={handleResizeStart} />
      <div class="chat-drawer-header">
        <span class="chat-drawer-title">{currentJob.title}</span>
        <StatusBadge status={currentJob.status} />
        <button class="btn btn-secondary btn-sm chat-drawer-close" onClick={onClose}>&times;</button>
      </div>
      <ChatPanel
        messages={currentJob.messages}
        status={currentJob.status}
        onReply={handleReply}
        onCancel={handleCancel}
        onApprove={handleApprove}
        onReject={handleReject}
        jobId={currentJob.id}
      />
    </div>
  );
}
