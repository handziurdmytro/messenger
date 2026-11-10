export interface User {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string;
  deleted: boolean;
}

export interface Tokens {
  access_token: string;
  refresh_token: string;
  expires_at: string;
}

export type ChatType = "private" | "group";

export interface Chat {
  id: string;
  type: ChatType;
  title?: string;
  owner_id?: string;
  participants: string[];
  created_at: string;
}

export interface Participant {
  user_id: string;
  role: "owner" | "member";
}

export interface Message {
  id: string;
  chatId: string;
  fromUser: string;
  type: string;
  text: string;
  createdAt: string;
  editedAt?: string;
  deletedAt?: string;
  pending?: boolean;
  failed?: string;
}

export interface ReadMarker {
  user_id: string;
  message_id: string;
  read_at: string;
}

export interface UnreadCount {
  count: number;
  capped: boolean;
}
