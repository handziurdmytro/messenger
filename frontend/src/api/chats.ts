import { json, request, requestWithHeaders } from "./client";
import type { Chat, Participant } from "./types";

const q = encodeURIComponent;

export interface ChatPage {
  chats: Chat[];
  next: string | null;
}

export async function listChats(before?: string | null): Promise<ChatPage> {
  const params = new URLSearchParams({ limit: "50" });
  if (before) params.set("before", before);
  const { data, headers } = await requestWithHeaders<Chat[] | null>(`/chat/?${params}`);
  return { chats: data || [], next: headers.get("X-Next-Cursor") };
}

export const getChat = (id: string) => request<Chat>(`/chats/${q(id)}`);

export const createPrivateChat = (contactId: string) => request<Chat>("/chat/", json("POST", { contact_id: contactId }));

export const createGroup = (title: string, memberIds: string[]) =>
  request<Chat>("/chats/groups", json("POST", { title, member_ids: memberIds }));

export const renameGroup = (id: string, title: string) => request<Chat>(`/chats/${q(id)}`, json("PATCH", { title }));

export const addParticipants = (id: string, userIds: string[]) =>
  request<Chat>(`/chats/${q(id)}/participants`, json("POST", { user_ids: userIds }));

export const removeParticipant = (id: string, userId: string) =>
  request<Chat | null>(`/chats/${q(id)}/participants/${q(userId)}`, json("DELETE"));

export const getParticipants = (id: string) => request<Participant[]>(`/chats/${q(id)}/participants`);

export const deleteChat = (id: string) => request(`/chats/${q(id)}`, json("DELETE"));
