// SPDX-License-Identifier: Apache-2.0

function a2aEndpoint(agentName?: string): string {
  if (agentName) return `/api/a2a/${agentName}`;
  return "/api/a2a/studio-threat-modeler";
}

interface A2AMessage {
  role: string;
  parts: Array<{ type: string; text: string }>;
}

interface A2AResponse {
  result?: { id?: string };
  id?: string;
}

export async function sendMessage(text: string, agentName?: string): Promise<A2AResponse> {
  const body = {
    jsonrpc: "2.0",
    id: crypto.randomUUID(),
    method: "message/send",
    params: {
      message: {
        role: "user",
        parts: [{ type: "text", text }],
      },
    },
  };

  const res = await fetch(a2aEndpoint(agentName), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  if (!res.ok) throw new Error(`A2A send failed: ${res.status}`);
  return res.json();
}

export async function sendReply(taskId: string, text: string, agentName?: string): Promise<A2AResponse> {
  const body = {
    jsonrpc: "2.0",
    id: crypto.randomUUID(),
    method: "message/send",
    params: {
      message: { role: "user", parts: [{ type: "text", text }] },
      taskId,
    },
  };
  const res = await fetch(a2aEndpoint(agentName), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`A2A reply failed: ${res.status}`);
  return res.json();
}

export interface StreamCallbacks {
  onStatus?: (state: string, status: Record<string, unknown>) => void;
  onMessage?: (message: A2AMessage) => void;
  onArtifact?: (artifact: Record<string, unknown>) => void;
  onError?: (error: Error) => void;
  onDone?: (state: string) => void;
}

export function streamTask(taskId: string, callbacks: StreamCallbacks, agentName?: string): () => void {
  const url = `${a2aEndpoint(agentName)}?taskId=${encodeURIComponent(taskId)}`;
  const es = new EventSource(url);
  let reconnectAttempts = 0;
  const maxReconnect = 5;

  es.onmessage = (event) => {
    reconnectAttempts = 0;
    try {
      const data = JSON.parse(event.data);
      if (data.status) callbacks.onStatus?.(data.status.state, data.status);
      if (data.artifacts) {
        for (const artifact of data.artifacts) callbacks.onArtifact?.(artifact);
      }
      if (data.message) callbacks.onMessage?.(data.message);
      if (data.status?.state === "completed" || data.status?.state === "failed") {
        es.close();
        callbacks.onDone?.(data.status.state);
      }
    } catch (e) {
      callbacks.onError?.(e as Error);
    }
  };

  es.onerror = () => {
    reconnectAttempts++;
    if (reconnectAttempts >= maxReconnect) {
      es.close();
      callbacks.onError?.(new Error("Disconnected"));
      callbacks.onDone?.("disconnected");
    }
  };

  return () => es.close();
}

export async function validate(yaml: string, definition: string, version = "latest"): Promise<{ valid: boolean; errors?: string[] }> {
  const res = await fetch("/api/validate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ yaml, definition, version }),
  });
  return res.json();
}

export async function publishBundle(input: { artifacts: string[]; target: string; tag?: string; sign?: boolean }): Promise<{ reference: string; digest: string; tag: string }> {
  const res = await fetch("/api/publish", {
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
