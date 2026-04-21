// SPDX-License-Identifier: Apache-2.0

import { useState, useRef, useEffect } from "preact/hooks";
import { currentView, selectedPolicyId, selectedTimeRange } from "../app";
import { streamMessage, streamReply, type StreamCallbacks, type ContextArtifact } from "../api/a2a";
import { apiFetch } from "../api/fetch";
import { renderMarkdown } from "../lib/markdown";

interface ChatMessage {
  role: "user" | "agent";
  text: string;
  artifact?: { content: string; name: string; mimeType?: string };
}

const STORAGE_KEY = "studio-chat-history";
const AGENT_NAME = "studio-assistant";

function loadHistory(): ChatMessage[] {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]");
  } catch {
    return [];
  }
}

function saveHistory(msgs: ChatMessage[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(msgs.slice(-50)));
}

export function ChatAssistant() {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>(loadHistory);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [streamBuffer, setStreamBuffer] = useState("");
  const messagesEnd = useRef<HTMLDivElement>(null);
  const abortRef = useRef<(() => void) | null>(null);
  const taskIdRef = useRef<string | null>(null);

  useEffect(() => {
    messagesEnd.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streamBuffer]);

  useEffect(() => {
    saveHistory(messages);
  }, [messages]);

  const buildContext = (): Record<string, string> => {
    const ctx: Record<string, string> = { view: currentView.value };
    if (selectedPolicyId.value) ctx.policy_id = selectedPolicyId.value;
    if (selectedTimeRange.value) {
      ctx.start = selectedTimeRange.value.start;
      ctx.end = selectedTimeRange.value.end;
    }
    return ctx;
  };

  const callbacks: StreamCallbacks = {
    onTaskId: (id) => { taskIdRef.current = id; },
    onStatus: (state) => {
      if (state === "completed" || state === "failed") {
        finalize(state);
      }
    },
    onMessage: (msg) => {
      if (msg.role === "user") return;
      const text = msg.parts?.map((p: any) => p.text || "").join("") || "";
      if (text) setStreamBuffer((prev) => prev + text);
    },
    onArtifact: (artifact: any) => {
      const parts = artifact?.parts || [];
      for (const part of parts) {
        if (part.metadata?.mimeType === "application/yaml") {
          const name = part.metadata?.name || "artifact.yaml";
          const content = part.text || "";
          setMessages((prev) => [
            ...prev,
            { role: "agent", text: `Artifact produced: ${name}`, artifact: { content, name, mimeType: "application/yaml" } },
          ]);
        }
      }
    },
    onError: (err) => {
      setMessages((prev) => [...prev, { role: "agent", text: `Error: ${err.message}` }]);
      setStreaming(false);
    },
    onDone: () => finalize("done"),
  };

  const finalize = (_state: string) => {
    setStreaming(false);
    setStreamBuffer((buf) => {
      if (buf) {
        setMessages((prev) => [...prev, { role: "agent", text: buf }]);
      }
      return "";
    });
  };

  const send = () => {
    if (!input.trim() || streaming) return;
    const text = input.trim();
    setInput("");
    setMessages((prev) => [...prev, { role: "user", text }]);
    setStreaming(true);
    setStreamBuffer("");

    const ctxMeta = buildContext();
    const contextPrefix = `[Dashboard context: ${JSON.stringify(ctxMeta)}]\n\n`;

    if (taskIdRef.current) {
      abortRef.current = streamReply(taskIdRef.current, text, callbacks, AGENT_NAME);
    } else {
      abortRef.current = streamMessage(contextPrefix + text, callbacks, AGENT_NAME);
    }
  };

  const saveAuditLog = async (artifact: { content: string; name: string }) => {
    try {
      await apiFetch("/api/audit-logs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          policy_id: selectedPolicyId.value || "unknown",
          content: artifact.content,
          audit_start: new Date().toISOString(),
          audit_end: new Date().toISOString(),
          summary: JSON.stringify({ saved_from: "chat" }),
        }),
      });
      setMessages((prev) => [...prev, { role: "agent", text: `Saved ${artifact.name} to Audit History.` }]);
    } catch (e) {
      setMessages((prev) => [...prev, { role: "agent", text: `Failed to save: ${e}` }]);
    }
  };

  return (
    <>
      <button class="chat-fab" onClick={() => setOpen(!open)} title="Chat with Studio Assistant">
        {open ? "\u2715" : "\u{1F4AC}"}
      </button>

      {open && (
        <div class="chat-overlay">
          <div class="chat-overlay-header">
            <h3>Studio Assistant</h3>
            <button class="btn btn-sm" onClick={() => { setMessages([]); taskIdRef.current = null; }}>Clear</button>
          </div>

          <div class="chat-overlay-messages">
            {messages.map((msg, i) => (
              <div key={i} class={`chat-msg chat-msg-${msg.role}`}>
                <span class="chat-msg-role">{msg.role === "user" ? "You" : "Agent"}</span>
                {msg.artifact ? (
                  <div class="chat-artifact-card">
                    <div class="chat-artifact-name">{msg.artifact.name}</div>
                    <pre class="chat-artifact-preview">{msg.artifact.content.slice(0, 500)}{msg.artifact.content.length > 500 ? "..." : ""}</pre>
                    <button class="btn btn-primary btn-sm" onClick={() => saveAuditLog(msg.artifact!)}>
                      Save to Audit History
                    </button>
                  </div>
                ) : (
                  <div class="chat-msg-text" dangerouslySetInnerHTML={{ __html: renderMarkdown(msg.text) }} />
                )}
              </div>
            ))}
            {streaming && (
              <div class="chat-msg chat-msg-agent">
                <span class="chat-msg-role">Agent</span>
                {streamBuffer ? (
                  <div class="chat-msg-text" dangerouslySetInnerHTML={{ __html: renderMarkdown(streamBuffer) }} />
                ) : (
                  <div class="chat-msg-text chat-thinking">Thinking<span class="dot-pulse" /></div>
                )}
              </div>
            )}
            <div ref={messagesEnd} />
          </div>

          <div class="chat-overlay-input">
            <input
              type="text"
              value={input}
              onInput={(e) => setInput((e.target as HTMLInputElement).value)}
              onKeyDown={(e) => e.key === "Enter" && send()}
              placeholder="Ask about audit gaps, evidence, or compliance..."
              disabled={streaming}
            />
            <button class="btn btn-primary" onClick={send} disabled={streaming || !input.trim()}>
              {streaming ? "..." : "Send"}
            </button>
          </div>
        </div>
      )}
    </>
  );
}
