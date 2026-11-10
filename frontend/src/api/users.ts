import { json, request } from "./client";
import type { User } from "./types";

const q = encodeURIComponent;

export const getMe = () => request<User>("/users/me");

export const updateMe = (displayName: string, avatarURL: string) =>
  request<User>("/users/me", json("PATCH", { display_name: displayName, avatar_url: avatarURL }));

export async function getProfiles(ids: string[]): Promise<User[]> {
  const unique = [...new Set(ids.filter(Boolean))];
  const result: User[] = [];
  for (let i = 0; i < unique.length; i += 100) {
    const chunk = unique.slice(i, i + 100);
    result.push(...((await request<User[] | null>(`/users/profiles?ids=${chunk.map(q).join(",")}`)) || []));
  }
  return result;
}

export const searchUsers = (query: string, signal?: AbortSignal) =>
  request<User[] | null>(`/users/search?q=${q(query)}&limit=20`, { signal }).then((users) => users || []);

export const getContacts = () => request<User[] | null>("/users/me/contacts").then((users) => users || []);

export const addContact = (id: string) => request("/users/me/contacts", json("POST", { contact_id: id }));

export const removeContact = (id: string) => request(`/users/me/contacts/${q(id)}`, json("DELETE"));

export const getBlocked = () => request<User[] | null>("/users/me/blocks").then((users) => users || []);

export const blockUser = (id: string) => request("/users/me/blocks", json("POST", { user_id: id }));

export const unblockUser = (id: string) => request(`/users/me/blocks/${q(id)}`, json("DELETE"));
