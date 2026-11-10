import { useState } from "react";
import * as chatsApi from "../api/chats";
import { errorText } from "../api/client";
import type { Chat, User } from "../api/types";
import { blockUser, removeContact } from "../api/users";
import { useMessenger, type Messenger } from "../hooks/useMessenger";
import { formatTime, nameOf } from "../lib/format";
import { ChatView } from "./ChatView";
import { GroupDialog } from "./GroupDialog";
import { NewChatDialog } from "./NewChatDialog";
import { SettingsDialog } from "./SettingsDialog";
import { Avatar, ErrorText, Icon, IconButton, UserAvatar } from "./ui";

function preview(messenger: Messenger, chat: Chat) {
  const message = messenger.previews[chat.id];
  if (!message) return chat.type === "group" ? `${chat.participants.length} members` : "No messages yet";
  const text = message.deletedAt ? "Message deleted" : message.text;
  if (message.fromUser === messenger.me?.id) return `You: ${text}`;
  if (chat.type === "group") return `${nameOf(messenger.profiles[message.fromUser])}: ${text}`;
  return text;
}

function ChatList({ messenger, filter }: { messenger: Messenger; filter: string }) {
  const [loadingMore, setLoadingMore] = useState(false);
  const chats = messenger.chats.filter((chat) => messenger.chatTitle(chat).toLowerCase().includes(filter.toLowerCase()));

  if (!chats.length) return <p className="muted center">{filter ? "No matches." : "No conversations yet."}</p>;

  return (
    <ul className="list">
      {chats.map((chat) => {
        const title = messenger.chatTitle(chat);
        const count = messenger.unread[chat.id] ?? 0;
        const last = messenger.previews[chat.id];
        return (
          <li key={chat.id}>
            <button
              className={`list-item${messenger.activeChat?.id === chat.id ? " active" : ""}`}
              onClick={() => messenger.select(chat.id)}
            >
              {chat.type === "group" ? <Avatar label={title} /> : <UserAvatar user={messenger.peerOf(chat)} />}
              <span className="list-text">
                <span className="list-line">
                  <strong>{title}</strong>
                  <time>{formatTime(last?.createdAt ?? chat.created_at)}</time>
                </span>
                <span className="list-line">
                  <small>{preview(messenger, chat)}</small>
                  {count > 0 && (
                    <span className="badge" aria-label={`${count} unread`}>
                      {count > 99 ? "99+" : count}
                    </span>
                  )}
                </span>
              </span>
            </button>
          </li>
        );
      })}
      {messenger.hasMoreChats && !filter && (
        <li>
          <button
            className="ghost load-more"
            disabled={loadingMore}
            onClick={() => {
              setLoadingMore(true);
              void messenger.loadMoreChats().finally(() => setLoadingMore(false));
            }}
          >
            Load more
          </button>
        </li>
      )}
    </ul>
  );
}

function ContactList({
  messenger,
  filter,
  onError,
  onBlock,
}: {
  messenger: Messenger;
  filter: string;
  onError: (error: string) => void;
  onBlock: (user: User) => void;
}) {
  const contacts = messenger.contacts.filter((user) =>
    `${nameOf(user)} ${user.email}`.toLowerCase().includes(filter.toLowerCase()),
  );
  if (!contacts.length) return <p className="muted center">{filter ? "No matches." : "No contacts yet."}</p>;

  return (
    <ul className="list">
      {contacts.map((user) => (
        <li key={user.id} className="list-item static">
          <UserAvatar user={user} />
          <span className="list-text">
            <strong>{nameOf(user)}</strong>
            <small>{user.email}</small>
          </span>
          <span className="row-actions">
            <IconButton
              icon="chat"
              label={`Message ${nameOf(user)}`}
              onClick={() => void messenger.openPrivate(user).catch((err) => onError(errorText(err)))}
            />
            <IconButton icon="block" label={`Block ${nameOf(user)}`} onClick={() => onBlock(user)} />
            <IconButton
              icon="trash"
              label={`Remove ${nameOf(user)}`}
              danger
              onClick={() => {
                if (!confirm(`Remove ${nameOf(user)} from contacts?`)) return;
                void removeContact(user.id)
                  .then(() => messenger.setContacts((old) => old.filter((item) => item.id !== user.id)))
                  .catch((err) => onError(errorText(err)));
              }}
            />
          </span>
        </li>
      ))}
    </ul>
  );
}

export function Workspace({ signOut, signedOut }: { signOut: () => void; signedOut: () => void }) {
  const messenger = useMessenger();
  const [tab, setTab] = useState<"chats" | "contacts">("chats");
  const [filter, setFilter] = useState("");
  const [dialog, setDialog] = useState<"" | "new" | "settings" | "group">("");
  const [actionError, setActionError] = useState("");
  const active = messenger.activeChat;
  const totalUnread = Object.values(messenger.unread).reduce((sum, count) => sum + count, 0);

  async function block(user: User) {
    if (!confirm(`Block ${nameOf(user)}? They will be removed from your contacts and cannot message you.`)) return;
    try {
      await blockUser(user.id);
      messenger.setBlocked((old) => [...old.filter((item) => item.id !== user.id), user]);
      messenger.setContacts((old) => old.filter((item) => item.id !== user.id));
    } catch (err) {
      setActionError(errorText(err));
    }
  }

  async function deleteActive() {
    if (!active) return;
    const question = active.type === "group" ? "Leave this group?" : "Delete this chat for both participants?";
    if (!confirm(question)) return;
    try {
      await chatsApi.deleteChat(active.id);
      messenger.removeChat(active.id);
    } catch (err) {
      setActionError(errorText(err));
    }
  }

  return (
    <main className={`workspace${active ? " has-chat" : ""}`}>
      <nav className="rail" aria-label="Navigation">
        <button
          className={`rail-button${tab === "chats" ? " active" : ""}`}
          aria-label="Chats"
          aria-pressed={tab === "chats"}
          onClick={() => setTab("chats")}
        >
          <Icon name="chat" />
          {totalUnread > 0 && <span className="dot" aria-hidden="true" />}
        </button>
        <button
          className={`rail-button${tab === "contacts" ? " active" : ""}`}
          aria-label="Contacts"
          aria-pressed={tab === "contacts"}
          onClick={() => setTab("contacts")}
        >
          <Icon name="people" />
        </button>
        <span className="rail-spacer" />
        <button className="rail-button" aria-label="Settings" disabled={!messenger.me} onClick={() => setDialog("settings")}>
          <Icon name="settings" />
        </button>
        <button className="rail-button" aria-label="Sign out" onClick={signOut}>
          <Icon name="logout" />
        </button>
      </nav>

      <aside className="sidebar">
        <header className="sidebar-head">
          <h1>{tab === "chats" ? "Chats" : "Contacts"}</h1>
          <IconButton icon="plus" label="New conversation" disabled={!messenger.me} onClick={() => setDialog("new")} />
        </header>
        <label className="search">
          <Icon name="search" size={16} />
          <input
            aria-label={tab === "chats" ? "Filter chats" : "Filter contacts"}
            placeholder="Search"
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
          />
        </label>
        {messenger.error && (
          <ErrorText>
            {messenger.error}{" "}
            <button className="link" onClick={messenger.retry}>
              Retry
            </button>
          </ErrorText>
        )}
        {actionError && <ErrorText>{actionError}</ErrorText>}
        {messenger.loading ? (
          <p className="muted center">Loading…</p>
        ) : tab === "chats" ? (
          <ChatList messenger={messenger} filter={filter} />
        ) : (
          <ContactList messenger={messenger} filter={filter} onError={setActionError} onBlock={(user) => void block(user)} />
        )}
      </aside>

      {active && messenger.me ? (
        <ChatView
          key={active.id}
          chat={active}
          messenger={messenger}
          onBack={() => messenger.select(null)}
          onInfo={() => setDialog("group")}
          onBlock={(user) => void block(user)}
          onDelete={() => void deleteActive()}
        />
      ) : (
        <section className="empty">
          <Icon name="chat" size={32} />
          <p>Select a conversation or start a new one.</p>
          <button className="secondary" disabled={!messenger.me} onClick={() => setDialog("new")}>
            New conversation
          </button>
        </section>
      )}

      {dialog === "new" && messenger.me && <NewChatDialog messenger={messenger} close={() => setDialog("")} />}
      {dialog === "settings" && messenger.me && (
        <SettingsDialog messenger={messenger} close={() => setDialog("")} signedOut={signedOut} />
      )}
      {dialog === "group" && active?.type === "group" && (
        <GroupDialog chat={active} messenger={messenger} close={() => setDialog("")} />
      )}
    </main>
  );
}
