// SPDX-License-Identifier: Apache-2.0

import { useState, useRef, useEffect } from "preact/hooks";
import {
  currentView, selectedPolicyId, selectedTimeRange, currentUser,
  selectedControlId, selectedRequirementId, selectedEvalResult,
  selectedPolicyDetail, activeTab,
  invalidateViews,
} from "../app";
import { streamMessage, streamReply, type StreamCallbacks } from "../api/a2a";
import { apiFetch } from "../api/fetch";
import { fetchConfig } from "../api/config";
import { fetchChatHistory, saveChatHistory } from "../api/chat";
import { renderMarkdown } from "../lib/markdown";
import { renderMermaidBlocks } from "../lib/mermaid";

interface ChatMessage {
  role: "user" | "agent";
  text: string;
  artifact?: { content: string; name: string; mimeType?: string; model?: string; promptVersion?: string; autoSaved?: boolean };
  isContextIndicator?: boolean;
  contextText?: string;
}

const STICKY_NOTES_KEY = "studio-sticky-notes";
const AGENT_NAME = "studio-assistant";
const MAX_STICKY_NOTES = 10;
const MAX_STICKY_NOTE_CHARS = 200;

interface StickyNote {
  id: string;
  text: string;
  createdAt: string;
}

function loadStickyNotes(): StickyNote[] {
  try {
    return JSON.parse(localStorage.getItem(STICKY_NOTES_KEY) || "[]");
  } catch {
    return [];
  }
}

function saveStickyNotes(notes: StickyNote[]) {
  localStorage.setItem(STICKY_NOTES_KEY, JSON.stringify(notes));
}

function buildInjectedContext(
  dashboardCtx: Record<string, string>,
  stickyNotes: StickyNote[],
): string {
  const parts: string[] = [];
  parts.push(`[Dashboard context: ${JSON.stringify(dashboardCtx)}]`);

  if (stickyNotes.length > 0) {
    const noteLines = stickyNotes.map((n) => `- ${n.text}`).join("\n");
    parts.push(`<sticky-notes>\n${noteLines}\n</sticky-notes>`);
  }

  return parts.join("\n\n");
}

function StickyNotesPanel({
  notes,
  onAdd,
  onDelete,
}: {
  notes: StickyNote[];
  onAdd: (text: string) => void;
  onDelete: (id: string) => void;
}) {
  const [draft, setDraft] = useState("");
  const atLimit = notes.length >= MAX_STICKY_NOTES;

  return (
    <div class="sticky-notes-panel">
      <div class="sticky-notes-list">
        {notes.length === 0 && <div class="sticky-notes-empty">No sticky notes yet</div>}
        {notes.map((n) => (
          <div key={n.id} class="sticky-note-item">
            <span class="sticky-note-text">{n.text}</span>
            <button class="sticky-note-delete" onClick={() => onDelete(n.id)} title="Delete">&times;</button>
          </div>
        ))}
      </div>
      <div class="sticky-notes-add">
        <input
          type="text"
          value={draft}
          maxLength={MAX_STICKY_NOTE_CHARS}
          onInput={(e) => setDraft((e.target as HTMLInputElement).value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && draft.trim() && !atLimit) {
              onAdd(draft.trim());
              setDraft("");
            }
          }}
          placeholder={atLimit ? "Limit reached (10)" : "Add a note\u2026"}
          disabled={atLimit}
        />
        <span class="sticky-notes-charcount">{draft.length}/{MAX_STICKY_NOTE_CHARS}</span>
        <button
          class="btn btn-sm btn-primary"
          disabled={!draft.trim() || atLimit}
          onClick={() => { onAdd(draft.trim()); setDraft(""); }}
        >Add</button>
      </div>
    </div>
  );
}

export function ChatAssistant({ open, onClose }: { open: boolean; onClose?: () => void }) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [streamBuffer, setStreamBuffer] = useState("");
  const [modelLabel, setModelLabel] = useState("");
  const [autoPersist, setAutoPersist] = useState(false);
  const [stickyNotes, setStickyNotes] = useState<StickyNote[]>(loadStickyNotes);
  const [showNotes, setShowNotes] = useState(false);
  const messagesEnd = useRef<HTMLDivElement>(null);
  const messagesContainer = useRef<HTMLDivElement>(null);
  const abortRef = useRef<(() => void) | null>(null);
  const taskIdRef = useRef<string | null>(null);
  const autoPersistRef = useRef(false);

  useEffect(() => {
    fetchChatHistory()
      .then((data) => {
        if (data.messages.length > 0) {
          setMessages(data.messages as ChatMessage[]);
        }
        if (data.taskId) {
          taskIdRef.current = data.taskId;
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    messagesEnd.current?.scrollIntoView({ behavior: "smooth" });
    if (messagesContainer.current) {
      renderMermaidBlocks(messagesContainer.current);
    }
  }, [messages, streamBuffer]);

  useEffect(() => {
    saveStickyNotes(stickyNotes);
  }, [stickyNotes]);

  useEffect(() => {
    fetchConfig().then((cfg) => {
      if (cfg.model_name) setModelLabel(cfg.model_name);
      const ap = cfg.auto_persist_artifacts === "true";
      setAutoPersist(ap);
      autoPersistRef.current = ap;
    });
  }, []);

  const isAdmin = (): boolean => currentUser.value?.role === "admin";

  const buildDashboardContext = (): Record<string, string> => {
    const ctx: Record<string, string> = { view: currentView.value };
    const policyId = selectedPolicyDetail.value || selectedPolicyId.value;
    if (policyId) ctx.policy_id = policyId;
    if (
      policyId &&
      (selectedPolicyDetail.value || currentView.value === "policies")
    ) {
      ctx.active_tab = activeTab.value;
    }
    if (selectedTimeRange.value) {
      ctx.start = selectedTimeRange.value.start;
      ctx.end = selectedTimeRange.value.end;
    }
    if (selectedControlId.value) ctx.control_id = selectedControlId.value;
    if (selectedRequirementId.value) ctx.requirement_id = selectedRequirementId.value;
    if (selectedEvalResult.value) ctx.eval_result = selectedEvalResult.value;
    return ctx;
  };

  const handleNewSession = () => {
    setMessages([]);
    taskIdRef.current = null;
    saveChatHistory([], null);
  };

  const addStickyNote = (text: string) => {
    setStickyNotes((prev) => {
      if (prev.length >= MAX_STICKY_NOTES) return prev;
      return [...prev, { id: crypto.randomUUID(), text, createdAt: new Date().toISOString() }];
    });
  };

  const deleteStickyNote = (id: string) => {
    setStickyNotes((prev) => prev.filter((n) => n.id !== id));
  };

  const finalize = (_state: string) => {
    setStreaming(false);
    setStreamBuffer((buf) => {
      if (buf) {
        setMessages((prev) => {
          const updated = [...prev, { role: "agent" as const, text: buf }];
          saveChatHistory(updated, taskIdRef.current);
          return updated;
        });
      } else {
        setMessages((prev) => {
          saveChatHistory(prev, taskIdRef.current);
          return prev;
        });
      }
      return "";
    });
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
      let shouldInvalidate = false;
      for (const part of parts) {
        if (part.metadata?.mimeType === "application/yaml") {
          const name = part.metadata?.name || "artifact.yaml";
          const content = part.text || "";
          const model = part.metadata?.model;
          const promptVersion = part.metadata?.promptVersion;
          setMessages((prev) => [
            ...prev,
            { role: "agent", text: `Artifact produced: ${name}`, artifact: { content, name, mimeType: "application/yaml", model, promptVersion, autoSaved: autoPersistRef.current } },
          ]);
          const lname = name.toLowerCase();
          if (lname.includes("auditlog") || lname.includes("assessment") || lname.includes("evidence") || lname.includes("posture")) {
            shouldInvalidate = true;
          }
        }
      }
      if (shouldInvalidate) {
        setTimeout(() => invalidateViews(), 500);
      }
    },
    onError: (err) => {
      setMessages((prev) => [...prev, { role: "agent", text: `Error: ${err.message}` }]);
      setStreaming(false);
    },
    onDone: () => finalize("done"),
  };

  // New tasks use streamMessage with context prefixed to the user text. Follow-up turns use
  // streamReply: dashboard + sticky notes are embedded once in `history` (first text part),
  // and `text` is only the new user utterance — no second buildInjectedContext on the message
  // tail, so context is not double-injected. See streamReply in api/a2a.ts.
  const send = () => {
    if (!input.trim() || streaming) return;
    const text = input.trim();
    setInput("");
    setMessages((prev) => [...prev, { role: "user", text }]);
    setStreaming(true);
    setStreamBuffer("");

    if (taskIdRef.current) {
      const recentMsgs = messages.filter((m) => !m.isContextIndicator).slice(-20);
      const historyLines = recentMsgs.map(
        (m) => `[${m.role === "user" ? "USER" : "ASSISTANT"}]\n${m.text}`
      );
      const ctx = buildDashboardContext();
      const contextPrefix = ctx ? `[DASHBOARD CONTEXT]\n${buildInjectedContext(ctx, stickyNotes)}\n\n` : "";
      const history = contextPrefix + "--- Conversation so far ---\n" + historyLines.join("\n\n");
      abortRef.current = streamReply(taskIdRef.current, text, callbacks, AGENT_NAME, { history });
    } else {
      const injected = buildInjectedContext(buildDashboardContext(), stickyNotes);
      const hasMemoryContext = stickyNotes.length > 0;

      if (hasMemoryContext) {
        setMessages((prev) => [
          ...prev,
          { role: "agent", text: "Memory context sent to agent", isContextIndicator: true, contextText: injected },
        ]);
      }

      abortRef.current = streamMessage(injected + "\n\n" + text, callbacks, AGENT_NAME);
    }
  };

  const saveAuditLog = async (artifact: { content: string; name: string; model?: string; promptVersion?: string }) => {
    try {
      const resp = await apiFetch("/api/audit-logs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          policy_id: selectedPolicyId.value || "unknown",
          content: artifact.content,
          ...(artifact.model && { model: artifact.model }),
          ...(artifact.promptVersion && { prompt_version: artifact.promptVersion }),
        }),
      });
      if (!resp.ok) {
        const errText = await resp.text();
        setMessages((prev) => [...prev, { role: "agent", text: `Failed to save: ${errText}` }]);
        return;
      }
      setMessages((prev) => [...prev, { role: "agent", text: `Saved ${artifact.name} to Audit History.` }]);
    } catch (e) {
      setMessages((prev) => [...prev, { role: "agent", text: `Failed to save: ${e}` }]);
    }
  };

  if (!open) return null;

  return (
    <>
        <div class="chat-overlay">
          <div class="chat-overlay-header">
            <div>
              <h3>Studio Assistant</h3>
              {modelLabel && <span class="chat-model-label">{modelLabel}</span>}
            </div>
            <div class="chat-header-controls">
              <button
                class={`btn btn-sm${showNotes ? " btn-primary" : " btn-secondary"}`}
                onClick={() => setShowNotes(!showNotes)}
                title="Sticky Notes"
              ><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M5 3m0 2a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v14a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2z"/><path d="M9 7l6 0"/><path d="M9 11l6 0"/><path d="M9 15l4 0"/></svg></button>
              <button
                class="btn btn-sm"
                onClick={handleNewSession}
                disabled={streaming}
              >New Session</button>
              {onClose && (
                <button class="btn btn-sm btn-secondary" onClick={onClose} title="Close" aria-label="Close chat">
                  &times;
                </button>
              )}
            </div>
          </div>

          {showNotes && (
            <StickyNotesPanel notes={stickyNotes} onAdd={addStickyNote} onDelete={deleteStickyNote} />
          )}

          <div class="chat-overlay-messages" ref={messagesContainer} role="log" aria-live="polite">
            {loading && <div class="view-loading">Loading chat...</div>}
            {messages.map((msg, i) => {
              if (msg.isContextIndicator) {
                return (
                  <details key={i} class="chat-context-indicator">
                    <summary>Memory context sent to agent</summary>
                    <pre class="chat-context-preview">{msg.contextText}</pre>
                  </details>
                );
              }

              return (
                <div key={i} class={`chat-msg chat-msg-${msg.role}`}>
                  <span class="chat-msg-role">{msg.role === "user" ? "You" : "Agent"}</span>
                  {msg.artifact ? (
                    <div class="chat-artifact-card">
                      <div class="chat-artifact-name">
                        {msg.artifact.name}
                        {msg.artifact.autoSaved && <span class="chat-artifact-autosaved">Auto-saved</span>}
                      </div>
                      <pre class="chat-artifact-preview">{msg.artifact.content.slice(0, 500)}{msg.artifact.content.length > 500 ? "..." : ""}</pre>
                      {isAdmin() && (
                        <button class="btn btn-primary btn-sm" onClick={() => saveAuditLog(msg.artifact!)}>
                          Save to Audit History
                        </button>
                      )}
                    </div>
                  ) : (
                    <div class="chat-msg-text" dangerouslySetInnerHTML={{ __html: renderMarkdown(msg.text) }} />
                  )}
                </div>
              );
            })}
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

          {!streaming && messages.length === 0 && (
            <div class="canned-queries">
              {[
                { label: "Run posture check", text: "Run a posture check for the current policy. Summarize pass/fail counts and highlight gaps." },
                { label: "Generate AuditLog", text: "Generate an AuditLog artifact for the current policy and audit window." },
                { label: "Summarize gaps", text: "Summarize the compliance gaps: which requirements have no evidence or failing evidence?" },
              ].map((q) => (
                <button
                  key={q.label}
                  class="btn btn-xs canned-btn"
                  onClick={() => { setInput(q.text); }}
                >
                  {q.label}
                </button>
              ))}
            </div>
          )}

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
    </>
  );
}
