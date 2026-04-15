// SPDX-License-Identifier: Apache-2.0
import { useEffect, useRef } from "preact/hooks";
import type { Message } from "../store/missions";
import { renderMarkdown } from "../lib/markdown";

interface ChatPanelProps { messages: Message[]; status: string; onReply: (text: string) => void }

export function ChatPanel({ messages, status, onReply }: ChatPanelProps) {
  const messagesRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const isActive = status === "input-required";
  useEffect(() => { const el = messagesRef.current; if (el) el.scrollTop = el.scrollHeight; }, [messages.length]);
  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); const text = inputRef.current?.value.trim(); if (!text) return; inputRef.current!.value = ""; onReply(text); }
  }
  return (
    <div class="chat-panel">
      <div class="chat-messages" ref={messagesRef}>
        {messages.map((msg, i) => (
          <div key={i} class="chat-message">
            <div class={`chat-message-role ${msg.role}`}>{msg.role === "user" ? "You" : "Agent"}</div>
            <div dangerouslySetInnerHTML={{ __html: renderMarkdown(msg.content) }} />
          </div>
        ))}
        {["submitted", "working"].includes(status) && (<div class="chat-thinking"><span class="spinner" /> Agent is working...</div>)}
      </div>
      <div class="chat-input-area">
        <textarea ref={inputRef} placeholder="Reply to the agent..." rows={1} disabled={!isActive} onKeyDown={handleKeyDown} />
      </div>
    </div>
  );
}
