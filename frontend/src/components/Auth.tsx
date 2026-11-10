import { useState, type FormEvent } from "react";
import { authenticate } from "../api/auth";
import { errorText } from "../api/client";
import { ErrorText, Icon } from "./ui";

export function Auth({ onLogin }: { onLogin: () => void }) {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const password = String(data.get("password"));
    const bytes = new TextEncoder().encode(password).length;
    if (mode === "register" && (bytes < 8 || bytes > 72)) {
      setError("Use a password between 8 and 72 bytes.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await authenticate(mode, String(data.get("email")).trim(), password);
      onLogin();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="auth">
      <form className="auth-card" onSubmit={submit}>
        <span className="logo">
          <Icon name="chat" size={22} />
        </span>
        <h1>{mode === "login" ? "Sign in" : "Create account"}</h1>
        <label>
          Email
          <input name="email" type="email" autoComplete="email" required disabled={busy} />
        </label>
        <label>
          Password
          <input
            name="password"
            type="password"
            autoComplete={mode === "login" ? "current-password" : "new-password"}
            placeholder={mode === "register" ? "At least 8 characters" : undefined}
            required
            disabled={busy}
          />
        </label>
        {error && <ErrorText>{error}</ErrorText>}
        <button className="primary" disabled={busy}>
          {busy ? "Please wait…" : mode === "login" ? "Sign in" : "Create account"}
        </button>
        <p className="auth-switch">
          {mode === "login" ? "No account?" : "Already registered?"}{" "}
          <button
            type="button"
            className="link"
            disabled={busy}
            onClick={() => {
              setMode(mode === "login" ? "register" : "login");
              setError("");
            }}
          >
            {mode === "login" ? "Create one" : "Sign in"}
          </button>
        </p>
      </form>
    </main>
  );
}
