import type { Tokens } from "./types";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

const sessionKey = "messenger.session.v1";
const base = (import.meta.env.VITE_API_BASE_URL || "/api").replace(/\/$/, "");

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

let session: Tokens | null = readStored(sessionKey, null);
let refreshing: Promise<Tokens> | null = null;

export function currentSession() {
  return session;
}

export function setSession(value: Tokens | null) {
  session = value;
  store(sessionKey, value);
}

interface Response<T> {
  data: T;
  headers: Headers;
}

async function send<T>(path: string, init: RequestInit = {}, token?: string): Promise<Response<T>> {
  let response: globalThis.Response;
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
    throw new Error("Cannot reach the server. Check your connection and try again.", { cause: error });
  }

  const text = await response.text();
  let data: unknown;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = null;
  }

  if (!response.ok) {
    const message = (data as { error?: string } | null)?.error;
    throw new ApiError(response.status, message || `Request failed (${response.status}).`);
  }

  return { data: data as T, headers: response.headers };
}

export async function publicRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  return (await send<T>(path, init)).data;
}

export async function accessToken(): Promise<string> {
  if (!session) throw new ApiError(401, "Please sign in again.");
  if (Date.parse(session.expires_at) > Date.now() + 30000) return session.access_token;

  if (!refreshing) {
    const original = session;
    refreshing = publicRequest<Tokens>("/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refresh_token: original.refresh_token }),
    })
      .then((tokens) => {
        if (session !== original) throw new Error("Session changed.");
        setSession(tokens);
        return tokens;
      })
      .catch((error) => {
        if (error instanceof ApiError && error.status === 401 && session === original) {
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

export async function requestWithHeaders<T>(path: string, init: RequestInit = {}): Promise<Response<T>> {
  const token = await accessToken();
  try {
    return await send<T>(path, init, token);
  } catch (error) {
    if (!(error instanceof ApiError) || error.status !== 401 || !session) throw error;
    session = { ...session, expires_at: new Date(0).toISOString() };
    return send<T>(path, init, await accessToken());
  }
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  return (await requestWithHeaders<T>(path, init)).data;
}

export function json(method: string, body?: unknown): RequestInit {
  return body === undefined ? { method } : { method, body: JSON.stringify(body) };
}

export function errorText(error: unknown) {
  return error instanceof Error ? error.message : "Something went wrong. Please try again.";
}
