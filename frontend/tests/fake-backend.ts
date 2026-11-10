import type { Page, Route, WebSocketRoute } from "@playwright/test";

export interface FakeUser {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string;
  deleted: boolean;
}

export interface FakeChat {
  id: string;
  type: "private" | "group";
  title?: string;
  owner_id?: string;
  participants: string[];
  created_at: string;
}

export interface FakeMessage {
  message_id: string;
  chat_id: string;
  from_user: string;
  type: string;
  payload: { text: string } | null;
  created_at: string;
  edited_at?: string;
  deleted_at?: string;
}

export interface Call {
  method: string;
  path: string;
  body: unknown;
}

export const tokens = {
  access_token: "access",
  refresh_token: "refresh",
  expires_at: "2099-01-01T00:00:00Z",
};

export function person(id: string, name: string): FakeUser {
  return { id, email: `${id}@example.com`, display_name: name, avatar_url: "", deleted: false };
}

export function message(id: string, chat: string, from: string, text: string, minute: number): FakeMessage {
  return {
    message_id: id,
    chat_id: chat,
    from_user: from,
    type: "text",
    payload: { text },
    created_at: `2026-11-10T10:${String(minute).padStart(2, "0")}:00Z`,
  };
}

export class FakeBackend {
  me = person("me", "Alex Morgan");
  users: FakeUser[] = [this.me, person("sam", "Sam Rivera"), person("kim", "Kim Lee"), person("lee", "Lee Park")];
  contacts = ["sam", "kim"];
  blocked: string[] = [];
  chats: FakeChat[] = [];
  messages: FakeMessage[] = [];
  unread: Record<string, number> = {};
  markers: Record<string, { user_id: string; message_id: string; read_at: string }[]> = {};
  calls: Call[] = [];
  socket: WebSocketRoute | null = null;
  sent: Record<string, unknown>[] = [];
  autoConfirm = true;
  overrides = new Map<string, (route: Route) => Promise<void> | void>();

  user(id: string) {
    return this.users.find((user) => user.id === id)!;
  }

  privateChat(id: string, other: string) {
    const chat: FakeChat = { id, type: "private", participants: ["me", other], created_at: "2026-11-10T09:00:00Z" };
    this.chats.push(chat);
    return chat;
  }

  groupChat(id: string, title: string, owner: string, members: string[]) {
    const chat: FakeChat = { id, type: "group", title, owner_id: owner, participants: [owner, ...members], created_at: "2026-11-10T09:00:00Z" };
    this.chats.push(chat);
    return chat;
  }

  push(event: Record<string, unknown>) {
    this.socket?.send(JSON.stringify(event));
  }

  callsTo(method: string, path: string) {
    return this.calls.filter((call) => call.method === method && call.path === path);
  }

  async install(page: Page, signedIn = true) {
    if (signedIn) {
      await page.addInitScript((value) => localStorage.setItem("messenger.session.v1", JSON.stringify(value)), tokens);
    }

    await page.routeWebSocket("**/ws", (ws) => {
      this.socket = ws;
      ws.onMessage((raw) => {
        const data = JSON.parse(String(raw)) as Record<string, unknown>;
        this.sent.push(data);
        if (!this.autoConfirm) return;
        ws.send(JSON.stringify({ type: "message.accepted", message_id: data.message_id }));
        const created = {
          message_id: String(data.message_id),
          chat_id: String(data.chat_id),
          from_user: "me",
          type: "text",
          payload: data.payload as { text: string },
          created_at: new Date().toISOString(),
        };
        this.messages.push(created);
        ws.send(JSON.stringify({ type: "message.created", message: created }));
      });
    });

    await page.route((url) => url.pathname.startsWith("/api/"), async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      const path = url.pathname.replace(/^\/api/, "");
      let body: unknown;
      try {
        body = request.postDataJSON();
      } catch {
        body = request.postData();
      }
      this.calls.push({ method: request.method(), path, body });
      const override = this.overrides.get(`${request.method()} ${path}`);
      if (override) return override(route);
      const result = this.handle(request.method(), path, url.searchParams, body as Record<string, unknown>);
      if (result === undefined) return route.fulfill({ status: 404, json: { error: `Unexpected ${request.method()} ${path}` } });
      if (result instanceof Response) return route.fulfill({ status: result.status, json: result.json });
      if (result === null) return route.fulfill({ status: 204 });
      return route.fulfill({ status: 200, json: result, headers: this.headers });
    });
  }

  private headers: Record<string, string> = {};

  private handle(method: string, path: string, params: URLSearchParams, body: Record<string, unknown>): unknown {
    this.headers = {};
    const parts = path.split("/").filter(Boolean);

    if (path.startsWith("/auth/")) {
      if (["/auth/login", "/auth/register", "/auth/refresh"].includes(path)) return tokens;
      return null;
    }

    if (path === "/users/me") {
      if (method === "PATCH") Object.assign(this.me, { display_name: body.display_name, avatar_url: body.avatar_url });
      return this.me;
    }
    if (path === "/users/profiles") {
      const ids = (params.get("ids") || "").split(",");
      return this.users.filter((user) => ids.includes(user.id));
    }
    if (path === "/users/search") {
      const term = (params.get("q") || "").toLowerCase();
      return this.users.filter((user) => user.id !== "me" && `${user.display_name} ${user.email}`.toLowerCase().includes(term));
    }
    if (path === "/users/me/contacts") {
      if (method === "POST") {
        this.contacts.push(String(body.contact_id));
        return { status: "success" };
      }
      return this.contacts.map((id) => this.user(id));
    }
    if (parts[1] === "me" && parts[2] === "contacts" && method === "DELETE") {
      this.contacts = this.contacts.filter((id) => id !== parts[3]);
      return null;
    }
    if (path === "/users/me/blocks") {
      if (method === "POST") {
        this.blocked.push(String(body.user_id));
        this.contacts = this.contacts.filter((id) => id !== body.user_id);
        return null;
      }
      return this.blocked.map((id) => this.user(id));
    }
    if (parts[1] === "me" && parts[2] === "blocks" && method === "DELETE") {
      this.blocked = this.blocked.filter((id) => id !== parts[3]);
      return null;
    }

    if (path === "/chat/") {
      if (method === "POST") {
        const existing = this.chats.find((chat) => chat.type === "private" && chat.participants.includes(String(body.contact_id)));
        return existing ?? this.privateChat(`chat-${body.contact_id}`, String(body.contact_id));
      }
      const sorted = [...this.chats];
      const limit = Number(params.get("limit") || 50);
      const start = params.get("before") ? sorted.findIndex((chat) => chat.id === params.get("before")) + 1 : 0;
      const page = sorted.slice(start, start + limit);
      if (start + limit < sorted.length) this.headers["X-Next-Cursor"] = page.at(-1)!.id;
      return page;
    }
    if (path === "/chats/groups") {
      return this.groupChat(`group-${this.chats.length}`, String(body.title), "me", body.member_ids as string[]);
    }
    if (parts[0] === "chats" && parts.length >= 2) {
      const chat = this.chats.find((item) => item.id === parts[1]);
      if (!chat) return new Response(404, { error: "chat not found" });
      if (parts.length === 2 && method === "GET") return chat;
      if (parts.length === 2 && method === "PATCH") {
        chat.title = String(body.title);
        return chat;
      }
      if (parts.length === 2 && method === "DELETE") {
        this.chats = this.chats.filter((item) => item.id !== chat.id);
        return null;
      }
      if (parts[2] === "participants" && method === "POST") {
        chat.participants.push(...(body.user_ids as string[]));
        return chat;
      }
      if (parts[2] === "participants" && method === "DELETE") {
        chat.participants = chat.participants.filter((id) => id !== parts[3]);
        return chat;
      }
    }

    if (parts[0] === "messages" && parts[1] === "chats") {
      const chatId = parts[2];
      if (parts[3] === "messages" && parts.length === 4) {
        const limit = Number(params.get("limit") || 50);
        const before = params.get("before");
        return this.messages
          .filter((item) => item.chat_id === chatId && (!before || item.created_at < before))
          .sort((a, b) => b.created_at.localeCompare(a.created_at))
          .slice(0, limit);
      }
      if (parts[3] === "messages" && parts.length === 5) {
        const item = this.messages.find((candidate) => candidate.message_id === parts[4]);
        if (!item) return new Response(404, { error: "message not found" });
        if (method === "PATCH") {
          item.payload = (body as { payload: { text: string } }).payload;
          item.edited_at = new Date().toISOString();
          return item;
        }
        if (method === "DELETE") {
          item.payload = null;
          item.deleted_at = new Date().toISOString();
          return null;
        }
      }
      if (parts[3] === "read") {
        if (method === "POST") {
          const item = this.messages.find((candidate) => candidate.message_id === body.message_id)!;
          return { user_id: "me", message_id: item.message_id, read_at: item.created_at };
        }
        return this.markers[chatId] ?? [];
      }
      if (parts[3] === "unread") return { count: this.unread[chatId] ?? 0, capped: false };
    }

    return undefined;
  }
}

class Response {
  constructor(
    public status: number,
    public json: unknown,
  ) {}
}
