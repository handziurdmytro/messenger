import { currentSession, json, publicRequest, request, setSession } from "./client";
import type { Tokens } from "./types";

export async function authenticate(mode: "login" | "register", email: string, password: string) {
  setSession(await publicRequest<Tokens>(`/auth/${mode}`, json("POST", { email, password })));
}

export async function logout() {
  const session = currentSession();
  setSession(null);
  if (!session) return;
  try {
    await publicRequest("/auth/logout", json("POST", { refresh_token: session.refresh_token }));
  } catch {
    return;
  }
}

export async function logoutEverywhere() {
  await request("/auth/logout-all", json("POST"));
  setSession(null);
}

export async function changePassword(email: string, currentPassword: string, newPassword: string) {
  await request("/auth/password", json("PATCH", { current_password: currentPassword, new_password: newPassword }));
  await authenticate("login", email, newPassword);
}

export async function deleteAccount(password: string) {
  await request("/auth/account", json("DELETE", { password }));
  setSession(null);
}
