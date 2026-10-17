export interface User {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string;
}
export interface Tokens {
  access_token: string;
  refresh_token: string;
  expires_at: string;
}
export interface Message {
  id: string;
  room_id: string;
  from_user: string;
  type: string;
  body: string;
  created_at: string;
  pending?: boolean;
}
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
const key = "messenger.session.v1";
export function readStored<T>(key: string, fallback: T): T {
  try {
    return JSON.parse(localStorage.getItem(key) || "null") ?? fallback;
  } catch {
    return fallback;
  }
}
export function store(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    return;
  }
}
let session: Tokens | null = readStored(key, null);
let refreshing: Promise<Tokens> | null = null;
export function hasSession() {
  return !!session;
}
export function setSession(value: Tokens | null) {
  session = value;
  store(key, value);
}
const base = (import.meta.env.VITE_API_BASE_URL || "/api").replace(/\/$/, "");
async function request<T>(
  path: string,
  init: RequestInit = {},
  token?: string,
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${base}${path}`, {
      ...init,
      signal: init.signal ?? AbortSignal.timeout(15000),
      headers: {
        ...(init.body ? { "Content-Type": "application/json" } : {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...init.headers,
      },
    });
  } catch (error) {
    if (init.signal?.aborted) throw error;
    throw new Error(
      "Cannot reach the server. Check your connection and try again.",
      { cause: error },
    );
  }
  const text = await response.text();
  let data: unknown;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = null;
  }
  if (!response.ok)
    throw new ApiError(
      response.status,
      (data as { error?: string })?.error ||
        `Request failed (${response.status}). Please try again.`,
    );
  return data as T;
}
export async function accessToken(): Promise<string> {
  if (!session) throw new ApiError(401, "Please sign in again.");
  if (Date.parse(session.expires_at) > Date.now() + 30000)
    return session.access_token;
  if (!refreshing) {
    const original = session;
    refreshing = request<Tokens>("/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refresh_token: original.refresh_token }),
    })
      .then((tokens) => {
        if (session !== original) throw new Error("Session changed.");
        setSession(tokens);
        return tokens;
      })
      .catch((error) => {
        if (
          error instanceof ApiError &&
          error.status === 401 &&
          session === original
        ) {
          setSession(null);
          window.dispatchEvent(new Event("session-expired"));
        }
        throw error;
      })
      .finally(() => {
        refreshing = null;
      });
  }
  return (await refreshing).access_token;
}
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = await accessToken();
  try {
    return await request<T>(path, init, token);
  } catch (error) {
    if (!(error instanceof ApiError) || error.status !== 401 || !session)
      throw error;
    session = { ...session, expires_at: new Date(0).toISOString() };
    return request<T>(path, init, await accessToken());
  }
}
export async function authenticate(
  mode: string,
  email: string,
  password: string,
) {
  setSession(
    await request<Tokens>(`/auth/${mode}`, {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  );
}
export function normalizeMessage(raw: Record<string, unknown>): Message {
  const event = raw.message as Record<string, unknown> | undefined;
  const message = event && typeof event === "object" ? event : raw;
  const payload = message.payload as Record<string, unknown> | undefined;
  return {
    id: String(message.message_id ?? ""),
    room_id: String(message.chat_id ?? ""),
    from_user: String(message.from_user ?? ""),
    type: String(message.type ?? "text"),
    body: typeof payload?.text === "string" ? payload.text : "",
    created_at: String(message.created_at ?? ""),
  };
}
export function socketURL(chat: string) {
  const url = new URL(import.meta.env.VITE_WS_URL || "/ws", location.href);
  url.protocol =
    url.protocol === "https:" || url.protocol === "wss:" ? "wss:" : "ws:";
  url.searchParams.set("chat_id", chat);
  return url.toString();
}
export const nameOf = (user: User) => user.display_name || user.email;
export const errorText = (error: unknown) =>
  error instanceof Error
    ? error.message
    : "Something went wrong. Please try again.";
