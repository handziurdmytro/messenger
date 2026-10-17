import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import {
  api,
  ApiError,
  authenticate,
  accessToken,
  errorText,
  hasSession,
  nameOf,
  normalizeMessage,
  readStored,
  setSession,
  socketURL,
  store,
  type Message,
  type User,
} from "./lib/api";

function Icon({ name, size = 22 }: { name: string; size?: number }) {
  const paths: Record<string, ReactNode> = {
    chat: (
      <>
        <path d="M21 11.5a8.5 8.5 0 0 1-8.5 8.5H4l-2 2V11.5a9.5 9.5 0 0 1 19 0Z" />
        <path d="M7 9h9M7 13h6" />
      </>
    ),
    people: (
      <>
        <circle cx="9" cy="8" r="3" />
        <path d="M2 21v-3a7 7 0 0 1 14 0v3M16 5a3 3 0 0 1 0 6M19 14a6 6 0 0 1 3 5v2" />
      </>
    ),
    search: (
      <>
        <circle cx="10.5" cy="10.5" r="6.5" />
        <path d="m16 16 5 5" />
      </>
    ),
    plus: <path d="M12 5v14M5 12h14" />,
    send: (
      <>
        <path d="m3 3 19 9-19 9 4-9-4-9ZM7 12h15" />
      </>
    ),
    back: <path d="m14 5-7 7 7 7" />,
    close: <path d="m6 6 12 12M6 18 18 6" />,
    logout: (
      <>
        <path d="M9 4H4v16h5M10 12h12m-5-5 5 5-5 5" />
      </>
    ),
    settings: (
      <>
        <circle cx="12" cy="8" r="4" />
        <path d="M4 22v-2a8 8 0 0 1 16 0v2" />
      </>
    ),
    check: <path d="m5 12 4 4L19 6" />,
    arrow: <path d="M4 12h16m-6-6 6 6-6 6" />,
  };
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {paths[name] || paths.chat}
    </svg>
  );
}
function Avatar({ user, large = false }: { user: User; large?: boolean }) {
  const [failed, setFailed] = useState(false);
  const safe = /^https?:\/\//.test(user.avatar_url);
  return (
    <span className={`avatar ${large ? "large" : ""}`}>
      {safe && !failed ? (
        <img
          src={user.avatar_url}
          alt=""
          referrerPolicy="no-referrer"
          onError={() => setFailed(true)}
        />
      ) : (
        nameOf(user).slice(0, 2).toUpperCase()
      )}
    </span>
  );
}
function ErrorBox({ children }: { children: ReactNode }) {
  return (
    <div className="error" role="alert">
      {children}
    </div>
  );
}
function Modal({
  title,
  children,
  close,
}: {
  title: string;
  children: ReactNode;
  close: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const dialog = ref.current!;
    dialog.showModal();
    return () => dialog.close();
  }, []);
  return (
    <dialog
      ref={ref}
      onCancel={close}
      onClick={(e) => {
        if (e.target === e.currentTarget) close();
      }}
    >
      <div className="modal-head">
        <h2>{title}</h2>
        <button
          className="icon-button"
          aria-label="Close dialog"
          onClick={close}
        >
          <Icon name="close" />
        </button>
      </div>
      {children}
    </dialog>
  );
}
function Auth({ onLogin }: { onLogin: () => void }) {
  const [mode, setMode] = useState("login"),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    const password = String(data.get("password"));
    if (
      mode === "register" &&
      (new TextEncoder().encode(password).length < 8 ||
        new TextEncoder().encode(password).length > 72)
    ) {
      setError("Use a password between 8 and 72 bytes.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await authenticate(mode, String(data.get("email")).trim(), password);
      onLogin();
    } catch (error) {
      setError(errorText(error));
    } finally {
      setBusy(false);
    }
  }
  return (
    <main className="auth">
      <section className="auth-story">
        <a className="brand" href="/">
          <span className="brand-mark">
            <Icon name="chat" />
          </span>
          messenger
        </a>
      </section>
      <section className="auth-panel">
        <div className="auth-form">
          <h2>
            {mode === "login" ? "Welcome back." : "Make yourself at home."}
          </h2>
          <p>
            {mode === "login"
              ? "Sign in to pick up where you left off."
              : "Create an account and start a conversation."}
          </p>
          <form onSubmit={submit}>
            <label>
              Email address
              <input
                name="email"
                type="email"
                placeholder="you@example.com"
                autoComplete="email"
                required
                disabled={busy}
              />
            </label>
            <label>
              Password
              <input
                name="password"
                type="password"
                placeholder={
                  mode === "login"
                    ? "Enter your password"
                    : "At least 8 characters"
                }
                autoComplete={
                  mode === "login" ? "current-password" : "new-password"
                }
                required
                disabled={busy}
              />
            </label>
            {error && <ErrorBox>{error}</ErrorBox>}
            <button className="primary wide" disabled={busy}>
              {busy
                ? "One moment…"
                : mode === "login"
                  ? "Sign in"
                  : "Create account"}
              <Icon name="arrow" size={18} />
            </button>
          </form>
          <p className="auth-switch">
            {mode === "login" ? "New around here?" : "Already have an account?"}{" "}
            <button
              className="text-button"
              disabled={busy}
              onClick={() => {
                setMode(mode === "login" ? "register" : "login");
                setError("");
              }}
            >
              {mode === "login" ? "Create an account" : "Sign in"}
            </button>
          </p>
        </div>
      </section>
    </main>
  );
}
interface Conversation {
  chatId: string;
  user: User;
}
interface ServerChat {
  id: string;
  participants: string[];
}
async function loadConversations(user: User): Promise<Conversation[]> {
  const chats = await api<ServerChat[]>("/chat/");
  return Promise.all(
    chats.map(async (chat) => {
      const id = chat.participants.find((id) => id !== user.id);
      if (!id) throw new Error("Invalid conversation participants.");
      const person = await api<User>(
        `/users/profiles/${encodeURIComponent(id)}`,
      );
      return { chatId: chat.id, user: person };
    }),
  );
}
async function loadProfile(): Promise<User> {
  for (let attempt = 0; ; attempt++) {
    try {
      return await api<User>("/users/me");
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 404 || attempt >= 5)
        throw error;
      await new Promise((resolve) => setTimeout(resolve, 500 * (attempt + 1)));
    }
  }
}
function Profile({
  user,
  saved,
  close,
}: {
  user: User;
  saved: (u: User) => void;
  close: () => void;
}) {
  const [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    setBusy(true);
    setError("");
    try {
      saved(
        await api<User>("/users/me", {
          method: "PATCH",
          body: JSON.stringify({
            display_name: String(data.get("name")).trim(),
            avatar_url: String(data.get("avatar")).trim(),
          }),
        }),
      );
      close();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title="Your profile" close={close}>
      <form onSubmit={submit}>
        <div className="profile-preview">
          <Avatar user={user} large />
          <div>
            <strong>{nameOf(user)}</strong>
            <p>{user.email}</p>
          </div>
        </div>
        <label>
          Display name
          <input
            name="name"
            defaultValue={user.display_name}
            required
            maxLength={100}
          />
        </label>
        <label>
          Avatar URL
          <input
            name="avatar"
            type="url"
            pattern="https?://.*"
            placeholder="https://example.com/avatar.jpg"
            defaultValue={user.avatar_url}
          />
        </label>
        {error && <ErrorBox>{error}</ErrorBox>}
        <button className="primary wide" disabled={busy}>
          {busy ? "Saving…" : "Save changes"}
        </button>
      </form>
    </Modal>
  );
}
function Discover({
  me,
  contacts,
  added,
  open,
  close,
}: {
  me: User;
  contacts: User[];
  added: (u: User) => void;
  open: (u: User) => Promise<void>;
  close: () => void;
}) {
  const [query, setQuery] = useState(""),
    [results, setResults] = useState<User[]>([]),
    [loading, setLoading] = useState(false),
    [error, setError] = useState(""),
    [busy, setBusy] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      if (!query.trim()) {
        setResults([]);
        setLoading(false);
        return;
      }
      setLoading(true);
      setError("");
      try {
        const users = await api<User[]>(
          `/users/search?q=${encodeURIComponent(query.trim())}`,
          { signal: controller.signal },
        );
        setResults((users || []).filter((u) => u.id !== me.id));
      } catch (e) {
        if (!controller.signal.aborted) setError(errorText(e));
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }, 300);
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [query, me.id]);
  async function choose(user: User, chat: boolean) {
    setBusy(user.id);
    setError("");
    try {
      if (chat) {
        await open(user);
        close();
      } else {
        await api("/users/me/contacts", {
          method: "POST",
          body: JSON.stringify({ contact_id: user.id }),
        });
        added(user);
      }
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy("");
    }
  }
  return (
    <Modal title="Start a conversation" close={close}>
      <p className="muted">Find someone by their name or email.</p>
      <div className="search">
        <Icon name="search" size={18} />
        <input
          autoFocus
          aria-label="Find people"
          placeholder="Search people…"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setResults([]);
          }}
        />
      </div>
      {error && <ErrorBox>{error}</ErrorBox>}
      <div className="discover-results">
        {loading ? (
          <p className="muted">Searching…</p>
        ) : (query.trim() ? results : contacts).length === 0 ? (
          <div className="small-empty">
            <Icon name="people" size={36} />
            <p>
              {query.trim()
                ? "No people found. Try another search."
                : "Search for someone to say hello."}
            </p>
          </div>
        ) : (
          (query.trim() ? results : contacts).map((u) => (
            <div className="person" key={u.id}>
              <Avatar user={u} />
              <div className="person-info">
                <strong>{nameOf(u)}</strong>
                <small>{u.email}</small>
              </div>
              {!contacts.some((c) => c.id === u.id) && (
                <button
                  className="icon-button"
                  disabled={!!busy}
                  aria-label={`Add ${nameOf(u)} to contacts`}
                  onClick={() => void choose(u, false)}
                >
                  <Icon name="plus" />
                </button>
              )}
              <button
                className="secondary"
                disabled={!!busy}
                onClick={() => void choose(u, true)}
              >
                {busy === u.id ? "…" : "Chat"}
              </button>
            </div>
          ))
        )}
      </div>
    </Modal>
  );
}
function Chat({
  conversation,
  me,
  back,
}: {
  conversation: Conversation;
  me: User;
  back: () => void;
}) {
  const [messages, setMessages] = useState<Message[]>([]),
    [draft, setDraft] = useState(() =>
      readStored<string>(`draft:${me.id}:${conversation.user.id}`, ""),
    ),
    [error, setError] = useState(""),
    [status, setStatus] = useState("Connecting"),
    [loading, setLoading] = useState(true),
    [more, setMore] = useState(false),
    [attempt, setAttempt] = useState(0);
  const ws = useRef<WebSocket | null>(null),
    bottom = useRef<HTMLDivElement>(null),
    room = useRef(conversation.chatId),
    cursor = useRef<{ createdAt: string; id: string } | null>(null);
  useEffect(() => {
    store(`draft:${me.id}:${conversation.user.id}`, draft);
  }, [draft, me.id, conversation.user.id]);
  useEffect(() => {
    let stopped = false;
    const controller = new AbortController();
    let socket: WebSocket | undefined;
    let syncTimer: ReturnType<typeof setInterval> | undefined;
    async function connect() {
      setStatus("Connecting");
      setLoading(true);
      setError("");
      setMessages([]);
      try {
        room.current = conversation.chatId;
        const token = await accessToken();
        if (stopped) return;
        socket = new WebSocket(socketURL(room.current), token);
        ws.current = socket;
        socket.onopen = () => {
          if (!stopped) setStatus("Connected");
        };
        socket.onclose = () => {
          if (!stopped) setStatus("Disconnected");
        };
        socket.onerror = () => {
          if (!stopped) setStatus("Disconnected");
        };
        socket.onmessage = (event) => {
          try {
            const incoming = JSON.parse(event.data) as Record<string, unknown>;
            if (incoming.type === "error") {
              setError(
                String(incoming.error || "The server rejected this message."),
              );
              return;
            }
            if (incoming.type !== "message.created") return;
            const msg = normalizeMessage(incoming);
            if (msg.room_id !== room.current) return;
            setMessages((old) => mergeMessages(old, [msg]));
          } catch {
            return;
          }
        };
        const raw = await api<Record<string, unknown>[] | null>(
          `/messages/chats/${encodeURIComponent(room.current)}/messages?limit=50`,
          { signal: controller.signal },
        );
        if (stopped) return;
        const list = (raw || []).map(normalizeMessage);
        const oldest = list.at(-1);
        cursor.current = oldest
          ? { createdAt: oldest.created_at, id: oldest.id }
          : null;
        setMessages((old) => mergeMessages(old, list));
        setMore(list.length === 50);
        syncTimer = setInterval(() => {
          void api<Record<string, unknown>[] | null>(
            `/messages/chats/${encodeURIComponent(room.current)}/messages?limit=50`,
            { signal: controller.signal },
          )
            .then((raw) => {
              if (!stopped)
                setMessages((old) =>
                  mergeMessages(old, (raw || []).map(normalizeMessage)),
                );
            })
            .catch(() => {});
        }, 10000);
      } catch (e) {
        if (!stopped) {
          setError(errorText(e));
          setStatus("Disconnected");
        }
      } finally {
        if (!stopped) setLoading(false);
      }
    }
    void connect();
    return () => {
      stopped = true;
      controller.abort();
      clearInterval(syncTimer);
      if (socket) {
        socket.onclose = null;
        socket.close();
      }
      ws.current = null;
    };
  }, [conversation.chatId, conversation.user.id, attempt]);
  const latestMessageId = messages.at(-1)?.id;
  useEffect(() => {
    bottom.current?.scrollIntoView({ behavior: "smooth" });
  }, [latestMessageId]);
  async function older() {
    setLoading(true);
    setError("");
    try {
      const raw = await api<Record<string, unknown>[] | null>(
        `/messages/chats/${encodeURIComponent(room.current)}/messages?limit=50&before=${encodeURIComponent(cursor.current?.createdAt || "")}&before_id=${encodeURIComponent(cursor.current?.id || "")}`,
      );
      const list = (raw || []).map(normalizeMessage);
      const oldest = list.at(-1);
      if (oldest)
        cursor.current = { createdAt: oldest.created_at, id: oldest.id };
      setMessages((old) => mergeMessages(old, list));
      setMore(list.length === 50);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setLoading(false);
    }
  }
  function send(e: FormEvent) {
    e.preventDefault();
    if (!draft.trim() || ws.current?.readyState !== WebSocket.OPEN) return;
    const messageID = crypto.randomUUID();
    const payload = JSON.stringify({
      message_id: messageID,
      chat_id: room.current,
      type: "text",
      payload: { text: draft.trim() },
    });
    if (new TextEncoder().encode(payload).length > 60 * 1024) {
      setError("This message is too long. Shorten it and try again.");
      return;
    }
    try {
      ws.current.send(payload);
      setMessages((old) => [
        ...old,
        {
          id: messageID,
          room_id: room.current,
          from_user: me.id,
          type: "text",
          body: draft.trim(),
          created_at: new Date().toISOString(),
          pending: true,
        },
      ]);
      setDraft("");
      setError("");
    } catch {
      setError("Message could not be sent. Your draft has been kept.");
    }
  }
  return (
    <section className="chat-pane">
      <header className="chat-header">
        <button
          className="icon-button mobile-back"
          aria-label="Back to conversations"
          onClick={back}
        >
          <Icon name="back" />
        </button>
        <Avatar user={conversation.user} />
        <div>
          <h2>{nameOf(conversation.user)}</h2>
          <span className="connection">
            <i className={status === "Connected" ? "connected" : ""} />
            {status}
          </span>
        </div>
      </header>
      {status === "Disconnected" && (
        <div className="connection-banner">
          Live connection unavailable. Your draft is safe.
          <button onClick={() => setAttempt((a) => a + 1)}>Reconnect</button>
        </div>
      )}
      {error && <ErrorBox>{error}</ErrorBox>}
      <div
        className="messages"
        role="log"
        aria-label="Messages"
        aria-live="polite"
      >
        {more && (
          <button
            className="secondary load-more"
            disabled={loading}
            onClick={() => void older()}
          >
            Load earlier messages
          </button>
        )}
        {loading && (
          <p className="center muted" role="status">
            Loading messages…
          </p>
        )}
        {!loading && !messages.length && (
          <div className="conversation-start">
            <Avatar user={conversation.user} large />
            <h2>This is where it begins.</h2>
            <p>Say hello to {nameOf(conversation.user)}.</p>
          </div>
        )}
        {messages.map((m, i) => (
          <div key={m.id}>
            {(i === 0 ||
              m.created_at.slice(0, 10) !==
                messages[i - 1].created_at.slice(0, 10)) && (
              <div className="date-divider">
                {new Date(m.created_at).toLocaleDateString(undefined, {
                  month: "long",
                  day: "numeric",
                  year: "numeric",
                })}
              </div>
            )}
            <div
              className={`message-row ${m.from_user === me.id ? "own" : ""}`}
            >
              <div className="bubble">
                <p>{m.body}</p>
                <span className="message-meta">
                  {new Date(m.created_at).toLocaleTimeString([], {
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                  {m.from_user === me.id && (
                    <span
                      title={
                        m.pending
                          ? "Waiting for persistence confirmation"
                          : "Saved by server"
                      }
                    >
                      {m.pending ? " · Unconfirmed" : " ✓"}
                    </span>
                  )}
                </span>
              </div>
            </div>
          </div>
        ))}
        <div ref={bottom} />
      </div>
      <form className="composer" onSubmit={send}>
        <div className="compose-field">
          <textarea
            aria-label="Message"
            placeholder={`Message ${nameOf(conversation.user)}…`}
            value={draft}
            rows={1}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (
                e.key === "Enter" &&
                !e.shiftKey &&
                !e.nativeEvent.isComposing
              ) {
                e.preventDefault();
                send(e);
              }
            }}
          />
          <button
            className="send"
            aria-label="Send message"
            disabled={!draft.trim() || status !== "Connected"}
          >
            <Icon name="send" size={20} />
          </button>
        </div>
        <span>Enter to send · Shift + Enter for a new line</span>
      </form>
    </section>
  );
}
function mergeMessages(old: Message[], incoming: Message[]) {
  const result = [...old];
  for (const msg of incoming) {
    const existing = result.findIndex((m) => m.id === msg.id);
    if (existing >= 0) {
      result[existing] = msg;
      continue;
    }
    result.push(msg);
  }
  return result.sort((a, b) => a.created_at.localeCompare(b.created_at));
}
function Workspace({ logout }: { logout: () => void }) {
  const [me, setMe] = useState<User | null>(null),
    [contacts, setContacts] = useState<User[]>([]),
    [conversations, setConversations] = useState<Conversation[]>([]),
    [selected, setSelected] = useState<Conversation | null>(null),
    [tab, setTab] = useState("chats"),
    [search, setSearch] = useState(""),
    [modal, setModal] = useState(""),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [retry, setRetry] = useState(0),
    [opening, setOpening] = useState(false);
  useEffect(() => {
    let active = true;
    Promise.all([loadProfile(), api<User[]>("/users/me/contacts")])
      .then(async ([user, people]) => {
        const chats = await loadConversations(user);
        if (active) {
          setMe(user);
          setContacts(people || []);
          setConversations(chats);
        }
      })
      .catch((e) => {
        if (active) setError(errorText(e));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [retry]);
  useEffect(() => {
    if (!me) return;
    let active = true;
    const timer = setInterval(() => {
      void loadConversations(me)
        .then((chats) => {
          if (active) setConversations(chats);
        })
        .catch((e) => {
          if (active) setError(errorText(e));
        });
    }, 10000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [me]);
  async function open(user: User) {
    if (!me) return;
    setOpening(true);
    try {
      const { id } = await api<{ id: string }>("/chat/", {
        method: "POST",
        body: JSON.stringify({ contact_id: user.id }),
      });
      const c = { chatId: id, user };
      setConversations((old) => {
        const list = [c, ...old.filter((item) => item.user.id !== user.id)];
        return list;
      });
      setSelected(c);
      setTab("chats");
    } finally {
      setOpening(false);
    }
  }
  const filtered = (
    tab === "contacts" ? contacts : conversations.map((c) => c.user)
  ).filter((u) =>
    `${nameOf(u)} ${u.email}`.toLowerCase().includes(search.toLowerCase()),
  );
  return (
    <main className={`workspace ${selected ? "has-chat" : ""}`}>
      <nav className="rail" aria-label="Main navigation">
        <span className="brand-mark">
          <Icon name="chat" />
        </span>
        <div className="rail-top">
          <button
            className={`rail-button ${tab === "chats" ? "active" : ""}`}
            aria-label="Conversations"
            aria-pressed={tab === "chats"}
            onClick={() => {
              setTab("chats");
              setSelected(null);
            }}
          >
            <Icon name="chat" />
          </button>
          <button
            className={`rail-button ${tab === "contacts" ? "active" : ""}`}
            aria-label="Contacts"
            aria-pressed={tab === "contacts"}
            onClick={() => {
              setTab("contacts");
              setSelected(null);
            }}
          >
            <Icon name="people" />
          </button>
        </div>
        <div className="rail-bottom">
          <button
            className="rail-button"
            aria-label="Edit profile"
            disabled={!me}
            onClick={() => setModal("profile")}
          >
            <Icon name="settings" />
          </button>
          <button
            className="rail-button"
            aria-label="Sign out"
            onClick={logout}
          >
            <Icon name="logout" />
          </button>
          {me && (
            <button
              className="avatar-button"
              aria-label="Your profile"
              onClick={() => setModal("profile")}
            >
              <Avatar user={me} />
            </button>
          )}
        </div>
      </nav>
      <aside className="sidebar">
        <header className="sidebar-header">
          <div>
            <h1>
              {tab === "chats" ? "Messages" : "Contacts"}
              <span className="count">
                {tab === "chats" ? conversations.length : contacts.length}
              </span>
            </h1>
          </div>
          <button
            className="new-chat"
            aria-label="New conversation"
            disabled={!me}
            onClick={() => setModal("discover")}
          >
            <Icon name="plus" />
          </button>
        </header>
        <div className="search">
          <Icon name="search" size={18} />
          <input
            aria-label={
              tab === "chats" ? "Search conversations" : "Search contacts"
            }
            placeholder={
              tab === "chats" ? "Search conversations" : "Search contacts"
            }
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="list-label">
          {tab === "chats" ? "ALL CONVERSATIONS" : "YOUR CONTACTS"}
          <span>{filtered.length}</span>
        </div>
        {error && (
          <ErrorBox>
            {error}
            <button
              className="text-button"
              onClick={() => {
                setLoading(true);
                setError("");
                setRetry((r) => r + 1);
              }}
            >
              Try again
            </button>
          </ErrorBox>
        )}
        <div className="conversation-list">
          {loading ? (
            <p className="center muted" role="status">
              Getting things ready…
            </p>
          ) : !filtered.length ? (
            <div className="small-empty">
              <Icon name={tab === "chats" ? "chat" : "people"} size={32} />
              <h3>
                {search
                  ? "No matches found"
                  : tab === "chats"
                    ? "Room for a hello"
                    : "Your people, right here"}
              </h3>
              <p>
                {search
                  ? "Try a different name or email."
                  : "Find someone and start your first conversation."}
              </p>
              {!search && (
                <button
                  className="text-button"
                  disabled={!me}
                  onClick={() => setModal("discover")}
                >
                  Find people <span>↗</span>
                </button>
              )}
            </div>
          ) : (
            filtered.map((u) => (
              <button
                key={u.id}
                className={`conversation ${selected?.user.id === u.id ? "selected" : ""}`}
                disabled={opening}
                onClick={() => {
                  setError("");
                  void open(u).catch((e) => setError(errorText(e)));
                }}
              >
                <Avatar user={u} />
                <span className="person-info">
                  <strong>{nameOf(u)}</strong>
                  <small>{u.email}</small>
                </span>
              </button>
            ))
          )}
        </div>
      </aside>
      {selected && me ? (
        <Chat
          key={selected.user.id}
          conversation={selected}
          me={me}
          back={() => setSelected(null)}
        />
      ) : (
        <section className="welcome">
          <h2>Your conversations</h2>
          <p>Select a chat or start a new one.</p>
          <button
            className="primary"
            disabled={!me}
            onClick={() => setModal("discover")}
          >
            <Icon name="plus" size={18} />
            Start a conversation
          </button>
        </section>
      )}
      {modal === "profile" && me && (
        <Profile user={me} saved={setMe} close={() => setModal("")} />
      )}
      {modal === "discover" && me && (
        <Discover
          me={me}
          contacts={contacts}
          added={(u) =>
            setContacts((old) =>
              old.some((c) => c.id === u.id) ? old : [...old, u],
            )
          }
          open={open}
          close={() => setModal("")}
        />
      )}
    </main>
  );
}
export default function App() {
  const [signedIn, setSignedIn] = useState(hasSession);
  useEffect(() => {
    const expired = () => setSignedIn(false);
    window.addEventListener("session-expired", expired);
    return () => window.removeEventListener("session-expired", expired);
  }, []);
  return signedIn ? (
    <Workspace
      logout={() => {
        setSession(null);
        setSignedIn(false);
      }}
    />
  ) : (
    <Auth onLogin={() => setSignedIn(true)} />
  );
}
