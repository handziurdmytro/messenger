import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiError, errorText } from "../api/client";
import * as chatsApi from "../api/chats";
import { getHistory, getUnread } from "../api/messages";
import type { Chat, Message, User } from "../api/types";
import * as usersApi from "../api/users";
import { Realtime, type ConnectionStatus } from "../realtime/socket";
import { nameOf } from "../lib/format";

async function loadMe(): Promise<User> {
  for (let attempt = 0; ; attempt++) {
    try {
      return await usersApi.getMe();
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 404 || attempt >= 5) throw error;
      await new Promise((resolve) => setTimeout(resolve, 500 * (attempt + 1)));
    }
  }
}

function activityOf(chat: Chat, preview?: Message) {
  return preview?.createdAt ?? chat.created_at;
}

export function useMessenger() {
  const realtime = useMemo(() => new Realtime(), []);
  const [me, setMe] = useState<User | null>(null);
  const [chats, setChats] = useState<Chat[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [profiles, setProfiles] = useState<Record<string, User>>({});
  const [previews, setPreviews] = useState<Record<string, Message>>({});
  const [unread, setUnread] = useState<Record<string, number>>({});
  const [contacts, setContacts] = useState<User[]>([]);
  const [blocked, setBlocked] = useState<User[]>([]);
  const [status, setStatus] = useState<ConnectionStatus>("connecting");
  const [activeId, setActiveId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [reload, setReload] = useState(0);

  const profilesRef = useRef(profiles);
  const chatsRef = useRef(chats);
  const activeRef = useRef(activeId);
  const meRef = useRef(me);
  useEffect(() => {
    profilesRef.current = profiles;
    chatsRef.current = chats;
    activeRef.current = activeId;
    meRef.current = me;
  });

  const rememberProfiles = useCallback((users: User[]) => {
    if (!users.length) return;
    setProfiles((old) => {
      const next = { ...old };
      users.forEach((user) => {
        next[user.id] = user;
      });
      return next;
    });
  }, []);

  const ensureProfiles = useCallback(
    async (ids: string[]) => {
      const missing = ids.filter((id) => id && !profilesRef.current[id]);
      if (!missing.length) return;
      rememberProfiles(await usersApi.getProfiles(missing));
    },
    [rememberProfiles],
  );

  const hydrateChats = useCallback(
    async (list: Chat[]) => {
      await ensureProfiles(list.flatMap((chat) => chat.participants));
      const details = await Promise.all(
        list.map(async (chat) => {
          const [history, count] = await Promise.all([
            getHistory(chat.id).catch(() => [] as Message[]),
            getUnread(chat.id).catch(() => ({ count: 0, capped: false })),
          ]);
          return { chat, preview: history[0], count: count.count };
        }),
      );
      setPreviews((old) => {
        const next = { ...old };
        details.forEach(({ chat, preview }) => {
          if (preview) next[chat.id] = preview;
        });
        return next;
      });
      setUnread((old) => {
        const next = { ...old };
        details.forEach(({ chat, count }) => {
          next[chat.id] = chat.id === activeRef.current ? 0 : count;
        });
        return next;
      });
    },
    [ensureProfiles],
  );

  const upsertChat = useCallback((chat: Chat) => {
    setChats((old) => [chat, ...old.filter((item) => item.id !== chat.id)]);
  }, []);

  const removeChat = useCallback((id: string) => {
    setChats((old) => old.filter((chat) => chat.id !== id));
    setActiveId((current) => (current === id ? null : current));
  }, []);

  const refreshChats = useCallback(async () => {
    const page = await chatsApi.listChats();
    const known = new Set(chatsRef.current.map((chat) => chat.id));
    setChats((old) => {
      const fresh = new Map(page.chats.map((chat) => [chat.id, chat]));
      const older = old.filter((chat) => !fresh.has(chat.id) && !page.next);
      return [...page.chats, ...older];
    });
    const added = page.chats.filter((chat) => !known.has(chat.id));
    if (added.length) await hydrateChats(added);
    else await ensureProfiles(page.chats.flatMap((chat) => chat.participants));
  }, [ensureProfiles, hydrateChats]);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [user, people, blockedUsers, page] = await Promise.all([
          loadMe(),
          usersApi.getContacts(),
          usersApi.getBlocked(),
          chatsApi.listChats(),
        ]);
        if (!active) return;
        setMe(user);
        setContacts(people);
        setBlocked(blockedUsers);
        rememberProfiles([user, ...people, ...blockedUsers]);
        setChats(page.chats);
        setNextCursor(page.next);
        await hydrateChats(page.chats);
        if (active) setError("");
      } catch (err) {
        if (active) setError(errorText(err));
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, [reload, hydrateChats, rememberProfiles]);

  useEffect(() => {
    realtime.start();
    const offStatus = realtime.onStatus(setStatus);
    return () => {
      offStatus();
      realtime.stop();
    };
  }, [realtime]);

  useEffect(() => {
    return realtime.subscribe((event) => {
      if (event.type === "message.created") {
        const message = event.message;
        setPreviews((old) => ({ ...old, [message.chatId]: message }));
        const known = chatsRef.current.some((chat) => chat.id === message.chatId);
        if (!known) {
          void chatsApi
            .getChat(message.chatId)
            .then(async (chat) => {
              await ensureProfiles(chat.participants);
              upsertChat(chat);
            })
            .catch(() => undefined);
        }
        const mine = message.fromUser === meRef.current?.id;
        const visible = activeRef.current === message.chatId && document.visibilityState === "visible";
        if (!mine && !visible) {
          setUnread((old) => ({ ...old, [message.chatId]: (old[message.chatId] ?? 0) + 1 }));
        }
        void ensureProfiles([message.fromUser]);
      }
      if (event.type === "message.edited") {
        setPreviews((old) =>
          old[event.message.chatId]?.id === event.message.id
            ? { ...old, [event.message.chatId]: { ...old[event.message.chatId], text: event.message.text, editedAt: event.message.editedAt } }
            : old,
        );
      }
      if (event.type === "message.deleted") {
        setPreviews((old) =>
          old[event.chatId]?.id === event.messageId
            ? { ...old, [event.chatId]: { ...old[event.chatId], text: "", deletedAt: event.deletedAt } }
            : old,
        );
      }
    });
  }, [realtime, ensureProfiles, upsertChat]);

  useEffect(() => {
    if (!me) return;
    const timer = setInterval(() => void refreshChats().catch(() => undefined), 30000);
    return () => clearInterval(timer);
  }, [me, refreshChats]);

  const loadMoreChats = useCallback(async () => {
    if (!nextCursor) return;
    const page = await chatsApi.listChats(nextCursor);
    setChats((old) => [...old, ...page.chats.filter((chat) => !old.some((item) => item.id === chat.id))]);
    setNextCursor(page.next);
    await hydrateChats(page.chats);
  }, [nextCursor, hydrateChats]);

  const openPrivate = useCallback(
    async (user: User) => {
      const chat = await chatsApi.createPrivateChat(user.id);
      rememberProfiles([user]);
      upsertChat(chat);
      setActiveId(chat.id);
      return chat;
    },
    [rememberProfiles, upsertChat],
  );

  const createGroup = useCallback(
    async (title: string, members: User[]) => {
      const chat = await chatsApi.createGroup(title, members.map((user) => user.id));
      rememberProfiles(members);
      upsertChat(chat);
      setActiveId(chat.id);
      return chat;
    },
    [rememberProfiles, upsertChat],
  );

  const markChatRead = useCallback((chatId: string) => {
    setUnread((old) => (old[chatId] ? { ...old, [chatId]: 0 } : old));
  }, []);

  const select = useCallback(
    (chatId: string | null) => {
      setActiveId(chatId);
      if (chatId) markChatRead(chatId);
    },
    [markChatRead],
  );

  const sortedChats = useMemo(
    () => [...chats].sort((a, b) => activityOf(b, previews[b.id]).localeCompare(activityOf(a, previews[a.id]))),
    [chats, previews],
  );

  const chatTitle = useCallback(
    (chat: Chat) => {
      if (chat.type === "group") return chat.title || "Group";
      const other = chat.participants.find((id) => id !== me?.id);
      return nameOf(other ? profiles[other] : undefined);
    },
    [me, profiles],
  );

  const peerOf = useCallback(
    (chat: Chat) => {
      if (chat.type !== "private") return undefined;
      const other = chat.participants.find((id) => id !== me?.id);
      return other ? profiles[other] : undefined;
    },
    [me, profiles],
  );

  return {
    realtime,
    me,
    setMe: (user: User) => {
      setMe(user);
      rememberProfiles([user]);
    },
    chats: sortedChats,
    hasMoreChats: !!nextCursor,
    loadMoreChats,
    profiles,
    ensureProfiles,
    rememberProfiles,
    previews,
    unread,
    contacts,
    setContacts,
    blocked,
    setBlocked,
    status,
    activeChat: chats.find((chat) => chat.id === activeId) ?? null,
    select,
    upsertChat,
    removeChat,
    refreshChats,
    openPrivate,
    createGroup,
    markChatRead,
    chatTitle,
    peerOf,
    error,
    loading,
    retry: () => {
      setLoading(true);
      setError("");
      setReload((value) => value + 1);
    },
  };
}

export type Messenger = ReturnType<typeof useMessenger>;
