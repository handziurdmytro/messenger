import { useState, type FormEvent } from "react";
import * as chatsApi from "../api/chats";
import { errorText } from "../api/client";
import type { Chat } from "../api/types";
import type { Messenger } from "../hooks/useMessenger";
import { nameOf } from "../lib/format";
import { PeoplePicker } from "./PeoplePicker";
import { ErrorText, IconButton, Modal, UserAvatar } from "./ui";

export function GroupDialog({ chat, messenger, close }: { chat: Chat; messenger: Messenger; close: () => void }) {
  const me = messenger.me!;
  const owner = chat.owner_id === me.id;
  const [adding, setAdding] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function update(action: () => Promise<Chat | null>) {
    setBusy(true);
    setError("");
    try {
      const updated = await action();
      if (updated) {
        await messenger.ensureProfiles(updated.participants);
        messenger.upsertChat(updated);
      }
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  function rename(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const title = String(new FormData(event.currentTarget).get("title")).trim();
    void update(() => chatsApi.renameGroup(chat.id, title));
  }

  return (
    <Modal title="Group" close={close}>
      {error && <ErrorText>{error}</ErrorText>}
      {owner ? (
        <form className="inline-form" onSubmit={rename}>
          <input name="title" aria-label="Group name" defaultValue={chat.title} required maxLength={128} />
          <button className="secondary" disabled={busy}>
            Rename
          </button>
        </form>
      ) : (
        <h3 className="group-title">{chat.title}</h3>
      )}

      <p className="section-label">{chat.participants.length} members</p>
      <ul className="people">
        {chat.participants.map((id) => {
          const user = messenger.profiles[id];
          return (
            <li key={id} className="person">
              <UserAvatar user={user} size="sm" />
              <span className="person-name">
                <strong>
                  {nameOf(user)}
                  {id === me.id && " (you)"}
                </strong>
                <small>{id === chat.owner_id ? "Owner" : "Member"}</small>
              </span>
              {owner && id !== me.id && (
                <span className="person-actions">
                  <IconButton
                    icon="close"
                    label={`Remove ${nameOf(user)}`}
                    disabled={busy}
                    onClick={() => void update(() => chatsApi.removeParticipant(chat.id, id))}
                  />
                </span>
              )}
            </li>
          );
        })}
      </ul>

      {owner &&
        (adding ? (
          <PeoplePicker
            me={me}
            contacts={messenger.contacts}
            exclude={chat.participants}
            actions={(user) => (
              <button
                className="secondary"
                disabled={busy}
                onClick={() => {
                  messenger.rememberProfiles([user]);
                  void update(() => chatsApi.addParticipants(chat.id, [user.id]));
                }}
              >
                Add
              </button>
            )}
          />
        ) : (
          <button className="ghost" onClick={() => setAdding(true)}>
            Add members
          </button>
        ))}
    </Modal>
  );
}
