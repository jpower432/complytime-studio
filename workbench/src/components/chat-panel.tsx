// SPDX-License-Identifier: Apache-2.0
import { useEffect, useRef, useState } from "preact/hooks";
import type { Message } from "../store/jobs";
import { acceptJob } from "../store/jobs";
import { renderMarkdown } from "../lib/markdown";

interface ChatPanelProps {
  messages: Message[];
  status: string;
  onReply: (text: string) => void;
  onCancel?: () => void;
  onApprove?: (toolCallId: string) => void;
  onReject?: (toolCallId: string) => void;
  jobId?: string;
}

export function ChatPanel({ messages, status, onReply, onCancel, onApprove, onReject, jobId }: ChatPanelProps) {
  const messagesRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const [showAccept, setShowAccept] = useState(false);
  const [acceptNote, setAcceptNote] = useState("");
  const isActive = status === "input-required";
  const canReply = isActive || status === "ready";
  const canCancel = ["submitted", "working", "input-required", "ready"].includes(status);
  const canAccept = status === "ready";

  useEffect(() => { const el = messagesRef.current; if (el) el.scrollTop = el.scrollHeight; }, [messages.length]);

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      const text = inputRef.current?.value.trim();
      if (!text) return;
      inputRef.current!.value = "";
      onReply(text);
    }
  }

  function handleAcceptSubmit() {
    if (jobId) acceptJob(jobId, acceptNote);
    setShowAccept(false);
    setAcceptNote("");
  }

  return (
    <div class="chat-panel">
      <div class="chat-messages" ref={messagesRef}>
        {messages.map((msg, i) => {
          if (msg.toolCall) {
            return <ToolCallBlock key={i} msg={msg} onApprove={onApprove} onReject={onReject} />;
          }
          return (
            <div key={i} class={`chat-message ${msg.partial ? "chat-message-streaming" : ""}`}>
              <div class={`chat-message-role ${msg.role}`}>{msg.role === "user" ? "You" : "Agent"}</div>
              <div dangerouslySetInnerHTML={{ __html: renderMarkdown(msg.content) }} />
              {msg.partial && <span class="typing-cursor" />}
            </div>
          );
        })}
        {["submitted", "working"].includes(status) && !messages.some((m) => m.partial) && (
          <div class="chat-thinking"><span class="spinner" /> Agent is working...</div>
        )}
      </div>

      <div class="chat-input-area">
        <textarea
          ref={inputRef}
          placeholder={canReply ? "Reply to the agent..." : "Waiting for agent..."}
          rows={1}
          disabled={!canReply}
          onKeyDown={handleKeyDown}
        />
      </div>

      {(canCancel || canAccept) && (
        <div class="chat-lifecycle-controls">
          {canCancel && <button class="btn btn-secondary btn-sm" onClick={onCancel}>Cancel Job</button>}
          {canAccept && <button class="btn btn-primary btn-sm" onClick={() => setShowAccept(true)}>Accept</button>}
        </div>
      )}

      {showAccept && (
        <div class="dialog-overlay" onClick={() => setShowAccept(false)}>
          <div class="dialog accept-dialog" onClick={(e) => e.stopPropagation()}>
            <h3>Accept Job</h3>
            <label class="dialog-label">Note (optional)</label>
            <textarea
              placeholder="e.g. Shipped to ghcr.io/..., Looks good, Partial — revisiting"
              value={acceptNote}
              onInput={(e) => setAcceptNote((e.target as HTMLTextAreaElement).value)}
              rows={3}
            />
            <div class="dialog-actions">
              <button class="btn btn-secondary" onClick={() => setShowAccept(false)}>Cancel</button>
              <button class="btn btn-primary" onClick={handleAcceptSubmit}>Accept</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function ToolCallBlock({ msg, onApprove, onReject }: {
  msg: Message;
  onApprove?: (id: string) => void;
  onReject?: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const tc = msg.toolCall!;
  const isPending = tc.status === "pending";
  const isExecuting = tc.status === "executing" || tc.status === "approved";
  const isCompleted = tc.status === "completed";
  const isRejected = tc.status === "rejected";

  return (
    <div class={`tool-call-block tool-call-${tc.status}`}>
      <div class="tool-call-header" onClick={() => isCompleted && setExpanded(!expanded)}>
        <span class="tool-call-icon">
          {isExecuting && <span class="spinner-sm" />}
          {isCompleted && "✓"}
          {isPending && "⏸"}
          {isRejected && "✗"}
        </span>
        <span class="tool-call-name">{tc.name}</span>
        {isCompleted && <span class="tool-call-chevron">{expanded ? "▾" : "▸"}</span>}
      </div>

      {isPending && (
        <div class="tool-call-approval">
          <span class="tool-call-approval-label">Waiting for your approval</span>
          <div class="tool-call-approval-actions">
            <button class="btn btn-primary btn-sm" onClick={() => onApprove?.(tc.id!)}>Approve</button>
            <button class="btn btn-secondary btn-sm" onClick={() => onReject?.(tc.id!)}>Reject</button>
          </div>
        </div>
      )}

      {isCompleted && expanded && tc.result && (
        <div class="tool-call-result">
          <pre>{tc.result}</pre>
        </div>
      )}
    </div>
  );
}
