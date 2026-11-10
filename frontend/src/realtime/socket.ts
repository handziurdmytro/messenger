import { accessToken } from "../api/client";
import { toMessage } from "../api/messages";
import type { Message, ReadMarker } from "../api/types";

export type RealtimeEvent =
  | { type: "message.created"; message: Message }
  | { type: "message.edited"; message: Message }
  | { type: "message.deleted"; chatId: string; messageId: string; deletedAt: string }
  | { type: "message.read"; chatId: string; marker: ReadMarker }
  | { type: "message.accepted"; messageId: string }
  | { type: "error"; messageId?: string; error: string };

export type ConnectionStatus = "connecting" | "online" | "offline";

type Listener = (event: RealtimeEvent) => void;
type StatusListener = (status: ConnectionStatus) => void;

export function parseEvent(raw: Record<string, unknown>): RealtimeEvent | null {
  const body = (raw.message ?? {}) as Record<string, unknown>;
  switch (raw.type) {
    case "message.created":
    case "message.edited":
      return { type: raw.type, message: toMessage(body) };
    case "message.deleted":
      return {
        type: "message.deleted",
        chatId: String(body.chat_id ?? ""),
        messageId: String(body.message_id ?? ""),
        deletedAt: String(body.deleted_at ?? ""),
      };
    case "message.read":
      return {
        type: "message.read",
        chatId: String(body.chat_id ?? ""),
        marker: {
          user_id: String(body.user_id ?? ""),
          message_id: String(body.message_id ?? ""),
          read_at: String(body.read_at ?? ""),
        },
      };
    case "message.accepted":
      return { type: "message.accepted", messageId: String(raw.message_id ?? "") };
    case "error":
      return {
        type: "error",
        messageId: raw.message_id ? String(raw.message_id) : undefined,
        error: String(raw.error ?? "The server rejected this message."),
      };
    default:
      return null;
  }
}

export function socketURL() {
  const url = new URL(import.meta.env.VITE_WS_URL || "/ws", location.href);
  url.protocol = url.protocol === "https:" || url.protocol === "wss:" ? "wss:" : "ws:";
  return url.toString();
}

export class Realtime {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private statusListeners = new Set<StatusListener>();
  private retryTimer: ReturnType<typeof setTimeout> | undefined;
  private attempt = 0;
  private stopped = false;
  status: ConnectionStatus = "connecting";

  start() {
    this.stopped = false;
    void this.connect();
  }

  stop() {
    this.stopped = true;
    clearTimeout(this.retryTimer);
    if (this.socket) {
      this.socket.onclose = null;
      this.socket.close();
    }
    this.socket = null;
  }

  reconnect() {
    clearTimeout(this.retryTimer);
    this.attempt = 0;
    if (this.socket) {
      this.socket.onclose = null;
      this.socket.close();
    }
    void this.connect();
  }

  subscribe(listener: Listener) {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  onStatus(listener: StatusListener) {
    this.statusListeners.add(listener);
    listener(this.status);
    return () => {
      this.statusListeners.delete(listener);
    };
  }

  send(payload: unknown): boolean {
    if (this.socket?.readyState !== WebSocket.OPEN) return false;
    this.socket.send(JSON.stringify(payload));
    return true;
  }

  private setStatus(status: ConnectionStatus) {
    this.status = status;
    this.statusListeners.forEach((listener) => listener(status));
  }

  private async connect() {
    this.setStatus("connecting");
    let token: string;
    try {
      token = await accessToken();
    } catch {
      this.scheduleRetry();
      return;
    }
    if (this.stopped) return;

    const socket = new WebSocket(socketURL(), token);
    this.socket = socket;
    socket.onopen = () => {
      this.attempt = 0;
      this.setStatus("online");
    };
    socket.onclose = () => this.scheduleRetry();
    socket.onmessage = (message) => {
      try {
        const event = parseEvent(JSON.parse(message.data) as Record<string, unknown>);
        if (event) this.listeners.forEach((listener) => listener(event));
      } catch {
        return;
      }
    };
  }

  private scheduleRetry() {
    if (this.stopped) return;
    this.setStatus("offline");
    const delay = Math.min(30000, 1000 * 2 ** this.attempt);
    this.attempt++;
    this.retryTimer = setTimeout(() => void this.connect(), delay);
  }
}
