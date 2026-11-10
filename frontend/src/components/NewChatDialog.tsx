import { useState, type FormEvent } from "react";
import { errorText } from "../api/client";
import type { User } from "../api/types";
import { addContact } from "../api/users";
import type { Messenger } from "../hooks/useMessenger";
import { nameOf } from "../lib/format";
import { PeoplePicker } from "./PeoplePicker";
import { ErrorText, Modal } from "./ui";

export function NewChatDialog({ messenger, close }: { messenger: Messenger; close: () => void }) {
  const me = messenger.me!;
  const [tab, setTab] = useState<"person" | "group">("person");
  const [members, setMembers] = useState<User[]>([]);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  async function run(key: string, action: () => Promise<unknown>) {
    setBusy(key);
    setError("");
    try {
      await action();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy("");
    }
  }

  function toggle(user: User) {
    setMembers((old) => (old.some((item) => item.id === user.id) ? old.filter((item) => item.id !== user.id) : [...old, user]));
  }

  function createGroup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const title = String(new FormData(event.currentTarget).get("title")).trim();
    void run("group", async () => {
      await messenger.createGroup(title, members);
      close();
    });
  }

  return (
    <Modal title="New conversation" close={close}>
      <div className="tabs" role="tablist">
        <button role="tab" aria-selected={tab === "person"} className={tab === "person" ? "active" : ""} onClick={() => setTab("person")}>
          Person
        </button>
        <button role="tab" aria-selected={tab === "group"} className={tab === "group" ? "active" : ""} onClick={() => setTab("group")}>
          Group
        </button>
      </div>
      {error && <ErrorText>{error}</ErrorText>}

      {tab === "person" ? (
        <PeoplePicker
          me={me}
          contacts={messenger.contacts}
          actions={(user) => (
            <>
              {!messenger.contacts.some((contact) => contact.id === user.id) && (
                <button
                  className="ghost"
                  disabled={!!busy}
                  onClick={() =>
                    void run(`add:${user.id}`, async () => {
                      await addContact(user.id);
                      messenger.setContacts((old) => [...old, user]);
                    })
                  }
                >
                  Add
                </button>
              )}
              <button
                className="secondary"
                disabled={!!busy}
                onClick={() =>
                  void run(`chat:${user.id}`, async () => {
                    await messenger.openPrivate(user);
                    close();
                  })
                }
              >
                Message
              </button>
            </>
          )}
        />
      ) : (
        <form className="stack" onSubmit={createGroup}>
          <label>
            Group name
            <input name="title" required maxLength={128} />
          </label>
          {!!members.length && (
            <div className="chips">
              {members.map((user) => (
                <button type="button" key={user.id} className="chip" onClick={() => toggle(user)} aria-label={`Remove ${nameOf(user)}`}>
                  {nameOf(user)} ×
                </button>
              ))}
            </div>
          )}
          <PeoplePicker
            me={me}
            contacts={messenger.contacts}
            actions={(user) => {
              const selected = members.some((item) => item.id === user.id);
              return (
                <button type="button" className={selected ? "secondary" : "ghost"} onClick={() => toggle(user)}>
                  {selected ? "Added" : "Add"}
                </button>
              );
            }}
          />
          <button className="primary" disabled={!members.length || !!busy}>
            {busy === "group" ? "Creating…" : `Create group${members.length ? ` (${members.length + 1})` : ""}`}
          </button>
        </form>
      )}
    </Modal>
  );
}
