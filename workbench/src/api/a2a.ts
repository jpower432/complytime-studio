// SPDX-License-Identifier: Apache-2.0

import { apiFetch } from "./fetch";

function a2aEndpoint(agentName?: string): string {
  if (agentName) return `/api/a2a/${agentName}`;
  return "/api/a2a/studio-threat-modeler";
}

interface A2AMessage {
  role: string;
  parts: Array<{ kind?: string; type?: string; text?: string; data?: Record<string, unknown>; metadata?: Record<string, unknown> }>;
}

export interface StreamCallbacks {
  onTaskId?: (taskId: string) => void;
  onStatus?: (state: string, status: Record<string, unknown>) => void;
  onMessage?: (message: A2AMessage) => void;
  onArtifact?: (artifact: Record<string, unknown>) => void;
  onError?: (error: Error) => void;
  onDone?: (state: string) => void;
}

/**
 * Send a new message via streaming A2A (message/stream).
 * Returns a cleanup function to abort the stream.
 */
export interface ContextArtifact {
  name: string;
  yaml: string;
}

export function streamMessage(text: string, callbacks: StreamCallbacks, agentName?: string, context?: ContextArtifact[]): () => void {
  const parts: Array<{ kind: string; text: string }> = [{ kind: "text", text }];
  if (context?.length) {
    for (const a of context) {
      parts.push({ kind: "text", text: `--- Context: ${a.name} ---\n${a.yaml}` });
    }
  }
  const body = {
    jsonrpc: "2.0",
    id: crypto.randomUUID(),
    method: "message/stream",
    params: {
      message: {
        messageId: crypto.randomUUID(),
        role: "user",
        parts,
      },
    },
  };
  return doStreamFetch(a2aEndpoint(agentName), body, callbacks);
}

/**
 * Send a reply on an existing task via streaming A2A.
 * Returns a cleanup function to abort the stream.
 */
export function streamReply(taskId: string, text: string, callbacks: StreamCallbacks, agentName?: string): () => void {
  const body = {
    jsonrpc: "2.0",
    id: crypto.randomUUID(),
    method: "message/stream",
    params: {
      message: {
        messageId: crypto.randomUUID(),
        role: "user",
        parts: [{ kind: "text", text }],
      },
      taskId,
    },
  };
  return doStreamFetch(a2aEndpoint(agentName), body, callbacks);
}

function doStreamFetch(url: string, body: object, callbacks: StreamCallbacks): () => void {
  const controller = new AbortController();

  (async () => {
    try {
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json", "Accept": "text/event-stream" },
        body: JSON.stringify(body),
        signal: controller.signal,
        credentials: "same-origin",
      });

      if (res.status === 401) {
        window.location.href = "/auth/login";
        return;
      }

      if (!res.ok) {
        const text = await res.text().catch(() => "");
        callbacks.onError?.(new Error(`A2A stream failed: ${res.status} — ${text.slice(0, 200)}`));
        callbacks.onDone?.("failed");
        return;
      }

      const contentType = res.headers.get("content-type") || "";
      if (contentType.includes("application/json")) {
        const json = await res.json();
        handleSyncResponse(json, callbacks);
        return;
      }

      const reader = res.body!.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      let eventCount = 0;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        const chunk = decoder.decode(value, { stream: true });
        buffer += chunk;

        const result = parseSSEBuffer(buffer);
        buffer = result.remaining;

        for (const eventData of result.events) {
          eventCount++;
          try {
            const parsed = JSON.parse(eventData);
            processEvent(parsed, callbacks);
          } catch {
            // skip malformed SSE data
          }
        }
      }

      buffer += decoder.decode();
      if (buffer.trim()) {
        const result = parseSSEBuffer(buffer + "\n\n");
        for (const eventData of result.events) {
          eventCount++;
          try {
            const parsed = JSON.parse(eventData);
            processEvent(parsed, callbacks);
          } catch {
            // skip
          }
        }
      }
    } catch (err) {
      if ((err as Error).name === "AbortError") return;
      callbacks.onError?.(err as Error);
      callbacks.onDone?.("failed");
    }
  })();

  return () => controller.abort();
}

const TERMINAL_STATES = new Set(["completed", "failed", "canceled", "rejected"]);

/**
 * Fallback: if the server returns JSON instead of SSE (message/send behavior),
 * extract the response and deliver it through callbacks.
 * Only fires onDone for terminal states — non-terminal JSON responses (e.g. a
 * "working" ack) leave the stream open so the caller can retry or poll.
 */
function handleSyncResponse(json: Record<string, unknown>, callbacks: StreamCallbacks) {
  const result = (json.result || json) as Record<string, unknown>;

  if (result.id) callbacks.onTaskId?.(result.id as string);

  if (result.status) {
    const status = result.status as Record<string, unknown>;
    callbacks.onStatus?.(status.state as string, status);
    if (status.message) callbacks.onMessage?.(status.message as A2AMessage);
  }

  if (result.artifacts) {
    for (const artifact of result.artifacts as Array<Record<string, unknown>>) {
      callbacks.onArtifact?.(artifact);
    }
  }

  const state = ((result.status as Record<string, unknown>)?.state as string) || "completed";
  if (TERMINAL_STATES.has(state)) {
    callbacks.onDone?.(state);
  }
}

function parseSSEBuffer(buffer: string): { events: string[]; remaining: string } {
  const events: string[] = [];
  const normalized = buffer.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  const blocks = normalized.split("\n\n");
  const remaining = blocks.pop() || "";

  for (const block of blocks) {
    if (!block.trim()) continue;
    let data = "";
    for (const line of block.split("\n")) {
      if (line.startsWith("data: ")) {
        data += line.slice(6);
      } else if (line.startsWith("data:")) {
        data += line.slice(5);
      }
    }
    if (data) events.push(data);
  }

  return { events, remaining };
}

function processEvent(data: Record<string, unknown>, callbacks: StreamCallbacks) {
  const result = (data.result || data) as Record<string, unknown>;

  if (result.id) callbacks.onTaskId?.(result.id as string);

  let statusMessageHandled = false;
  if (result.status) {
    const status = result.status as Record<string, unknown>;
    const state = status.state as string;
    callbacks.onStatus?.(state, status);

    if (status.message) {
      callbacks.onMessage?.(status.message as A2AMessage);
      statusMessageHandled = true;
    }

    if (state === "completed" || state === "failed") {
      callbacks.onDone?.(state);
    }
  }

  if (result.message && !statusMessageHandled) {
    callbacks.onMessage?.(result.message as A2AMessage);
  }

  if (result.artifact) {
    callbacks.onArtifact?.(result.artifact as Record<string, unknown>);
  }
  if (result.artifacts) {
    for (const artifact of result.artifacts as Array<Record<string, unknown>>) {
      callbacks.onArtifact?.(artifact);
    }
  }
}

export async function validate(yaml: string, definition: string, version = "latest"): Promise<{ valid: boolean; errors?: string[] }> {
  const res = await apiFetch("/api/validate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ yaml, definition, version }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `Validation failed: ${res.status}`);
  }
  return res.json();
}

export async function publishBundle(input: { artifacts: string[]; target: string; tag?: string }): Promise<{ reference: string; digest: string; tag: string }> {
  const res = await apiFetch("/api/publish", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `Publish failed: ${res.status}`);
  }
  return res.json();
}
