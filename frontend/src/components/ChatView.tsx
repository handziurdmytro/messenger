import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { errorText, readStored, store } from "../api/client";
import * as messagesApi from "../api/messages";
import type { Chat, Message, ReadMarker, User } from "../api/types";
import type { Messenger } from "../hooks/useMessenger";
import { formatTime, nameOf } from "../lib/format";
import { Avatar, ErrorText, Icon, IconButton, UserAvatar } from "./ui";

function mergeMessages(old: Message[], incoming: Message[]) {
  const byId = new Map(old.map((message) => [message.id, message]));
  incoming.forEach((message) => byId.set(message.id, { ...byId.get(message.id), ...message, pending: message.pending }));
  return [...byId.values()].sort((a, b) => a.createdAt.localeCompare(b.createdAt) || a.id.localeCompare(b.id));
}

function isRead(message: Message, markers: ReadMarker[], me: string) {
  const created = Date.parse(message.createdAt);
  return markers.some((marker) => marker.user_id !== me && Date.parse(marker.read_at) >= created);
}

export function ChatView({
  chat,
  messenger,
  onBack,
  onInfo,
  onBlock,
  onDelete,
}: {
  chat: Chat;
  messenger: Messenger;
  onBack: () => void;
  onInfo: () => void;
  onBlock: (user: User) => void;
  onDelete: () => void;
}) {
  const me = messenger.me!;
  const { realtime, profiles, markChatRead, ensureProfiles, status } = messenger;
  const draftKey = `draft:${me.id}:${chat.id}`;
  const [messages, setMessages] = useState<Message[]>([]);
  const [markers, setMarkers] = useState<ReadMarker[]>([]);
  const [draft, setDraft] = useState(() => readStored<string>(draftKey, ""));
  const [editing, setEditing] = useState<Message | null>(null);
  const [loading, setLoading] = useState(true);
  const [hasMore, setHasMore] = useState(false);
  const [error, setError] = useState("");
  const bottom = useRef<HTMLDivElement>(null);
  const lastMarked = useRef("");
  const peer = messenger.peerOf(chat);
  const blockedPeer = !!peer && messenger.blocked.some((user) => user.id === peer.id);
  const canWrite = status === "online" && !peer?.deleted && !blockedPeer;

  useEffect(() => {
    store(draftKey, draft);
  }, [draft, draftKey]);

  useEffect(() => {
    const controller = new AbortController();
    (async () => {
      try {
        const [history, readMarkers] = await Promise.all([
          messagesApi.getHistory(chat.id, undefined, controller.signal),
          messagesApi.getReadMarkers(chat.id),
        ]);
        if (controller.signal.aborted) return;
        setMessages((old) => mergeMessages(old, history));
        setMarkers(readMarkers);
        setHasMore(history.length === messagesApi.historyPageSize);
        await ensureProfiles(history.map((message) => message.fromUser));
      } catch (err) {
        if (!controller.signal.aborted) setError(errorText(err));
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    })();
    return () => controller.abort();
  }, [chat.id, ensureProfiles]);

  useEffect(() => {
    return realtime.subscribe((event) => {
      switch (event.type) {
        case "message.created":
          if (event.message.chatId === chat.id) setMessages((old) => mergeMessages(old, [{ ...event.message, pending: false }]));
          break;
        case "message.edited":
          if (event.message.chatId === chat.id) {
            setMessages((old) =>
              old.map((message) =>
                message.id === event.message.id ? { ...message, text: event.message.text, editedAt: event.message.editedAt } : message,
              ),
            );
          }
          break;
        case "message.deleted":
          if (event.chatId === chat.id) {
            setMessages((old) =>
              old.map((message) => (message.id === event.messageId ? { ...message, text: "", deletedAt: event.deletedAt } : message)),
            );
          }
          break;
        case "message.read":
          if (event.chatId === chat.id) {
            setMarkers((old) => [...old.filter((marker) => marker.user_id !== event.marker.user_id), event.marker]);
          }
          break;
        case "error":
          if (event.messageId) {
            setMessages((old) => old.map((message) => (message.id === event.messageId ? { ...message, failed: event.error } : message)));
          } else {
            setError(event.error);
          }
          break;
      }
    });
  }, [realtime, chat.id]);

  const latest = messages.at(-1);

  useEffect(() => {
    bottom.current?.scrollIntoView({ block: "end" });
  }, [latest?.id]);

  const markLatestRead = useCallback(() => {
    const incoming = [...messages].reverse().find((message) => message.fromUser !== me.id && !message.pending);
    if (!incoming || incoming.id === lastMarked.current || document.visibilityState !== "visible") return;
    lastMarked.current = incoming.id;
    markChatRead(chat.id);
    void messagesApi.markRead(chat.id, incoming.id).catch(() => {
      lastMarked.current = "";
    });
  }, [messages, me.id, chat.id, markChatRead]);

  useEffect(() => {
    markLatestRead();
    document.addEventListener("visibilitychange", markLatestRead);
    return () => document.removeEventListener("visibilitychange", markLatestRead);
  }, [markLatestRead]);

  async function loadEarlier() {
    const oldest = messages.find((message) => !message.pending);
    if (!oldest) return;
    setLoading(true);
    try {
      const page = await messagesApi.getHistory(chat.id, oldest);
      await ensureProfiles(page.map((message) => message.fromUser));
      setMessages((old) => mergeMessages(old, page));
      setHasMore(page.length === messagesApi.historyPageSize);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setLoading(false);
    }
  }

  async function submit(event?: FormEvent) {
    event?.preventDefault();
    const text = draft.trim();
    if (!text) return;
    setError("");

    if (editing) {
      try {
        const updated = await messagesApi.editMessage(chat.id, editing.id, text);
        setMessages((old) => old.map((message) => (message.id === updated.id ? { ...message, text: updated.text, editedAt: updated.editedAt } : message)));
        setEditing(null);
        setDraft("");
      } catch (err) {
        setError(errorText(err));
      }
      return;
    }

    const id = crypto.randomUUID();
    const sent = realtime.send({ message_id: id, chat_id: chat.id, type: "text", payload: { text } });
    if (!sent) {
      setError("You are offline. Your draft has been kept.");
      return;
    }
    setMessages((old) =>
      mergeMessages(old, [{ id, chatId: chat.id, fromUser: me.id, type: "text", text, createdAt: new Date().toISOString(), pending: true }]),
    );
    setDraft("");
  }

  async function remove(message: Message) {
    if (!confirm("Delete this message for everyone?")) return;
    try {
      await messagesApi.deleteMessage(chat.id, message.id);
      setMessages((old) => old.map((item) => (item.id === message.id ? { ...item, text: "", deletedAt: new Date().toISOString() } : item)));
    } catch (err) {
      setError(errorText(err));
    }
  }

  function startEdit(message: Message) {
    setEditing(message);
    setDraft(message.text);
  }

  function cancelEdit() {
    setEditing(null);
    setDraft("");
  }

  const title = messenger.chatTitle(chat);
  const subtitle =
    chat.type === "group"
      ? `${chat.participants.length} members`
      : status === "online"
        ? peer?.email ?? ""
        : status === "connecting"
          ? "Connecting…"
          : "Offline";

  return (
    <section className="chat" aria-label={`Chat with ${title}`}>
      <header className="chat-head">
        <IconButton icon="back" label="Back" onClick={onBack} />
        {chat.type === "group" ? <Avatar label={title} /> : <UserAvatar user={peer} />}
        <div className="chat-title">
          <h2>{title}</h2>
          <span>{subtitle}</span>
        </div>
        <div className="chat-actions">
          {chat.type === "group" && <IconButton icon="info" label="Group info" onClick={onInfo} />}
          {chat.type === "private" && peer && !peer.deleted && !blockedPeer && (
            <IconButton icon="block" label="Block user" onClick={() => onBlock(peer)} />
          )}
          <IconButton icon="trash" label={chat.type === "group" ? "Leave group" : "Delete chat"} onClick={onDelete} danger />
        </div>
      </header>

      {status === "offline" && (
        <div className="banner">
          Connection lost. Retrying…
          <button className="link" onClick={() => realtime.reconnect()}>
            Reconnect now
          </button>
        </div>
      )}

      <div className="messages" role="log" aria-label="Messages" aria-live="polite">
        {hasMore && (
          <button className="ghost load-earlier" disabled={loading} onClick={() => void loadEarlier()}>
            Load earlier messages
          </button>
        )}
        {loading && !messages.length && <p className="muted center">Loading…</p>}
        {!loading && !messages.length && <p className="muted center">No messages yet. Say hello.</p>}
        {messages.map((message, index) => {
          const mine = message.fromUser === me.id;
          const prev = messages[index - 1];
          const newDay = !prev || prev.createdAt.slice(0, 10) !== message.createdAt.slice(0, 10);
          const showAuthor = chat.type === "group" && !mine && (newDay || prev?.fromUser !== message.fromUser);
          return (
            <div key={message.id}>
              {newDay && (
                <div className="day">
                  {new Date(message.createdAt).toLocaleDateString([], { month: "long", day: "numeric", year: "numeric" })}
                </div>
              )}
              <div className={`message${mine ? " mine" : ""}${message.deletedAt ? " deleted" : ""}`} data-message-id={message.id}>
                <div className="bubble">
                  {showAuthor && <span className="author">{nameOf(profiles[message.fromUser])}</span>}
                  <p>{message.deletedAt ? "Message deleted" : message.text}</p>
                  <span className="meta">
                    {message.editedAt && !message.deletedAt && "edited · "}
                    {formatTime(message.createdAt)}
                    {mine && !message.deletedAt && (
                      <span className="ticks" aria-label={message.failed ? "Failed" : message.pending ? "Sending" : isRead(message, markers, me.id) ? "Read" : "Sent"}>
                        {message.failed ? " !" : message.pending ? " …" : <Icon name={isRead(message, markers, me.id) ? "checks" : "check"} size={14} />}
                      </span>
                    )}
                  </span>
                  {message.failed && <span className="failed">{message.failed}</span>}
                </div>
                {mine && !message.deletedAt && !message.pending && (
                  <div className="message-tools">
                    <IconButton icon="edit" label="Edit message" onClick={() => startEdit(message)} />
                    <IconButton icon="trash" label="Delete message" onClick={() => void remove(message)} />
                  </div>
                )}
              </div>
            </div>
          );
        })}
        <div ref={bottom} />
      </div>

      {error && <ErrorText>{error}</ErrorText>}

      <form className="composer" onSubmit={(event) => void submit(event)}>
        {editing && (
          <div className="editing">
            <span>Editing message</span>
            <button type="button" className="link" onClick={cancelEdit}>
              Cancel
            </button>
          </div>
        )}
        <div className="composer-row">
          <textarea
            aria-label="Message"
            rows={1}
            placeholder={peer?.deleted ? "This account was deleted" : blockedPeer ? "You blocked this user" : "Message"}
            value={draft}
            disabled={!!peer?.deleted || blockedPeer}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape" && editing) cancelEdit();
              if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
                event.preventDefault();
                void submit();
              }
            }}
          />
          <button className="send" aria-label={editing ? "Save message" : "Send message"} disabled={!draft.trim() || (!editing && !canWrite)}>
            <Icon name={editing ? "check" : "send"} size={18} />
          </button>
        </div>
      </form>
    </section>
  );
}
