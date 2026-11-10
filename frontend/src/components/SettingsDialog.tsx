import { useState, type FormEvent } from "react";
import { changePassword, deleteAccount, logoutEverywhere } from "../api/auth";
import { errorText } from "../api/client";
import { unblockUser, updateMe } from "../api/users";
import type { Messenger } from "../hooks/useMessenger";
import { nameOf } from "../lib/format";
import { ErrorText, Modal, UserAvatar } from "./ui";

function useAction() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState("");
  async function run(action: () => Promise<void>, success = "") {
    setBusy(true);
    setError("");
    setDone("");
    try {
      await action();
      setDone(success);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }
  return { busy, error, done, run };
}

function Feedback({ error, done }: { error: string; done: string }) {
  if (error) return <ErrorText>{error}</ErrorText>;
  if (done)
    return (
      <p className="success" role="status">
        {done}
      </p>
    );
  return null;
}

export function SettingsDialog({ messenger, close, signedOut }: { messenger: Messenger; close: () => void; signedOut: () => void }) {
  const me = messenger.me!;
  const profile = useAction();
  const password = useAction();
  const sessions = useAction();
  const blocks = useAction();
  const danger = useAction();

  function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    void profile.run(async () => {
      messenger.setMe(await updateMe(String(data.get("name")).trim(), String(data.get("avatar")).trim()));
    }, "Profile saved.");
  }

  function savePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    void password.run(async () => {
      await changePassword(me.email, String(data.get("current")), String(data.get("next")));
      form.reset();
    }, "Password changed. Other devices were signed out.");
  }

  function removeAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    if (!confirm("Delete your account permanently?")) return;
    void danger.run(async () => {
      await deleteAccount(String(data.get("password")));
      signedOut();
    });
  }

  return (
    <Modal title="Settings" close={close}>
      <section className="settings-section">
        <h3>Profile</h3>
        <div className="profile-row">
          <UserAvatar user={me} size="lg" />
          <span>
            <strong>{nameOf(me)}</strong>
            <small>{me.email}</small>
          </span>
        </div>
        <form className="stack" onSubmit={saveProfile}>
          <label>
            Display name
            <input name="name" defaultValue={me.display_name} required maxLength={100} />
          </label>
          <label>
            Avatar URL
            <input name="avatar" type="url" pattern="https?://.*" defaultValue={me.avatar_url} placeholder="https://" />
          </label>
          <Feedback error={profile.error} done={profile.done} />
          <button className="secondary" disabled={profile.busy}>
            Save profile
          </button>
        </form>
      </section>

      <section className="settings-section">
        <h3>Password</h3>
        <form className="stack" onSubmit={savePassword}>
          <label>
            Current password
            <input name="current" type="password" autoComplete="current-password" required />
          </label>
          <label>
            New password
            <input name="next" type="password" autoComplete="new-password" minLength={8} maxLength={72} required />
          </label>
          <Feedback error={password.error} done={password.done} />
          <button className="secondary" disabled={password.busy}>
            Change password
          </button>
        </form>
      </section>

      <section className="settings-section">
        <h3>Blocked users</h3>
        <Feedback error={blocks.error} done={blocks.done} />
        {messenger.blocked.length ? (
          <ul className="people">
            {messenger.blocked.map((user) => (
              <li key={user.id} className="person">
                <UserAvatar user={user} size="sm" />
                <span className="person-name">
                  <strong>{nameOf(user)}</strong>
                  <small>{user.email}</small>
                </span>
                <span className="person-actions">
                  <button
                    className="ghost"
                    disabled={blocks.busy}
                    onClick={() =>
                      void blocks.run(async () => {
                        await unblockUser(user.id);
                        messenger.setBlocked((old) => old.filter((item) => item.id !== user.id));
                      })
                    }
                  >
                    Unblock
                  </button>
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="muted">Nobody is blocked.</p>
        )}
      </section>

      <section className="settings-section">
        <h3>Sessions</h3>
        <Feedback error={sessions.error} done={sessions.done} />
        <button
          className="secondary"
          disabled={sessions.busy}
          onClick={() =>
            void sessions.run(async () => {
              await logoutEverywhere();
              signedOut();
            })
          }
        >
          Sign out on all devices
        </button>
      </section>

      <section className="settings-section danger-zone">
        <h3>Delete account</h3>
        <p className="muted">Your profile becomes “Deleted account”. Conversations stay visible to others.</p>
        <form className="inline-form" onSubmit={removeAccount}>
          <input name="password" type="password" aria-label="Confirm password" placeholder="Password" autoComplete="current-password" required />
          <button className="danger" disabled={danger.busy}>
            Delete
          </button>
        </form>
        <Feedback error={danger.error} done={danger.done} />
      </section>
    </Modal>
  );
}
