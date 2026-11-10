import { json, request } from "./client";
import type { Message, ReadMarker, UnreadCount } from "./types";

const q = encodeURIComponent;

export function toMessage(raw: Record<string, unknown>): Message {
  const payload = raw.payload as { text?: unknown } | null | undefined;
  return {
    id: String(raw.message_id ?? ""),
    chatId: String(raw.chat_id ?? ""),
    fromUser: String(raw.from_user ?? ""),
    type: String(raw.type ?? "text"),
    text: typeof payload?.text === "string" ? payload.text : "",
    createdAt: String(raw.created_at ?? ""),
    editedAt: typeof raw.edited_at === "string" ? raw.edited_at : undefined,
    deletedAt: typeof raw.deleted_at === "string" ? raw.deleted_at : undefined,
  };
}

export const historyPageSize = 50;

export async function getHistory(chatId: string, before?: Message, signal?: AbortSignal): Promise<Message[]> {
  const params = new URLSearchParams({ limit: String(historyPageSize) });
  if (before) {
    params.set("before", before.createdAt);
    params.set("before_id", before.id);
  }
  const raw = await request<Record<string, unknown>[] | null>(`/messages/chats/${q(chatId)}/messages?${params}`, { signal });
  return (raw || []).map(toMessage);
}

export const editMessage = (chatId: string, id: string, text: string) =>
  request<Record<string, unknown>>(`/messages/chats/${q(chatId)}/messages/${q(id)}`, json("PATCH", { payload: { text } })).then(toMessage);

export const deleteMessage = (chatId: string, id: string) =>
  request(`/messages/chats/${q(chatId)}/messages/${q(id)}`, json("DELETE"));

export const markRead = (chatId: string, messageId: string) =>
  request<ReadMarker>(`/messages/chats/${q(chatId)}/read`, json("POST", { message_id: messageId }));

export const getReadMarkers = (chatId: string) =>
  request<ReadMarker[] | null>(`/messages/chats/${q(chatId)}/read`).then((markers) => markers || []);

export const getUnread = (chatId: string) => request<UnreadCount>(`/messages/chats/${q(chatId)}/unread`);
