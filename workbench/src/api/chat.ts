// SPDX-License-Identifier: Apache-2.0

import { apiFetch } from "./fetch";

interface ChatHistoryResponse {
  messages: Array<{ role: string; text: string; artifact?: unknown; isContextIndicator?: boolean; contextText?: string }>;
  taskId: string | null;
}

export async function fetchChatHistory(): Promise<ChatHistoryResponse> {
  try {
    const res = await apiFetch("/api/chat/history");
    if (!res.ok) return { messages: [], taskId: null };
    return res.json();
  } catch {
    return { messages: [], taskId: null };
  }
}

export async function saveChatHistory(
  messages: Array<{ role: string; text: string; artifact?: unknown; isContextIndicator?: boolean; contextText?: string }>,
  taskId: string | null,
): Promise<void> {
  try {
    await apiFetch("/api/chat/history", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ messages: messages.slice(-50), taskId: taskId || "" }),
    });
  } catch {
    console.warn("failed to save chat history");
  }
}
