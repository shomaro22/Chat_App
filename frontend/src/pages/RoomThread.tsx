import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { fetchWithAuth } from "../lib/api";
import { requestRoomsRefresh } from "../lib/roomsRefresh";
import { toErrorMessage } from "../chat/errors";
import {
  markMessageDeletedInList,
  sortMessagesAsc,
} from "../chat/messageUtils";
import { MessageComposer } from "../components/MessageComposer";
import { ConfirmActionModal } from "../components/ConfirmActionModal";
import { MessageList } from "../components/MessageList";
import { RoomSettingsModal } from "../components/RoomSettingsModal";
import { useRoomComposerActions } from "../hooks/useRoomComposerActions";
import { useRoomSettingsActions } from "../hooks/useRoomSettingsActions";
import { useReadState } from "../hooks/useReadState";
import { useRoomDetail } from "../hooks/useRoomDetail";
import { useRoomMessages } from "../hooks/useRoomMessages";
import {
  useRoomWebSocket,
  type RoomWsHandlerBag,
} from "../hooks/useRoomWebSocket";
import type {
  Message,
  ReaderInfo,
  ReadStateResponse,
  ReactionChip,
  ReactionRow,
} from "../types/chat";
import { MESSAGE_PAGE_LIMIT } from "../types/chat";

function aggregateReactions(
  rows: ReactionRow[],
  selfId: string | null,
  resolveUserName: (userId: string) => string,
): ReactionChip[] {
  const map = new Map<
    string,
    { count: number; mine: boolean; userNames: string[] }
  >();
  for (const r of rows) {
    const cur = map.get(r.emoji) ?? { count: 0, mine: false, userNames: [] };
    cur.count++;
    if (selfId && r.user_id === selfId) cur.mine = true;
    cur.userNames.push(resolveUserName(r.user_id));
    map.set(r.emoji, cur);
  }
  return [...map.entries()].map(([emoji, v]) => ({
    emoji,
    count: v.count,
    mine: v.mine,
    userNames: v.userNames,
  }));
}

export function RoomThread() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const targetMessageId = searchParams.get("message_id")?.trim() || null;
  const [error, setError] = useState<string | null>(null);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [selfUserId, setSelfUserId] = useState<string | null>(null);
  const [membersOnline, setMembersOnline] = useState<Record<string, boolean>>(
    {},
  );
  const [wsStatus, setWsStatus] = useState<"connecting" | "open" | "closed">(
    "connecting",
  );
  const [retryCount, setRetryCount] = useState(0);
  const [showRoomModal, setShowRoomModal] = useState(false);
  const [roomModalError, setRoomModalError] = useState<string | null>(null);
  const [renameDraft, setRenameDraft] = useState("");
  const [addMemberRef, setAddMemberRef] = useState("");
  const [roomActionBusy, setRoomActionBusy] = useState(false);
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [reactionsByMessage, setReactionsByMessage] = useState<
    Record<string, ReactionChip[]>
  >({});
  const [pin, setPin] = useState<{ message_id: string } | null>(null);
  const [bookmarkedByMessage, setBookmarkedByMessage] = useState<
    Record<string, boolean>
  >({});
  const [typingUserIds, setTypingUserIds] = useState<string[]>([]);
  const [peerBlocked, setPeerBlocked] = useState(false);
  const [blockBusy, setBlockBusy] = useState(false);
  const [highlightedMessageId, setHighlightedMessageId] = useState<
    string | null
  >(null);
  const [confirmAction, setConfirmAction] = useState<
    | null
    | { type: "close-room" }
    | { type: "leave-room"; userId: string }
    | { type: "remove-member"; userId: string }
  >(null);
  const typingTimeoutsRef = useRef<Record<string, number>>({});
  const typingIdleTimerRef = useRef<number | null>(null);

  const messageListRef = useRef<HTMLDivElement | null>(null);
  const scrollAfterOlderRef = useRef<{ sh: number; st: number } | null>(null);
  const initialScrolledRoomRef = useRef<string | null>(null);
  const olderScrollLockRef = useRef(false);
  const readSyncTimerRef = useRef<number | null>(null);
  const handlersRef = useRef<RoomWsHandlerBag>({
    onSessionReady: () => {},
    onReloadData: () => {},
    onMessageCreated: () => {},
    onMessageUpdated: () => {},
    onMessageDeleted: () => {},
    onReadUpdated: () => {},
    onPresenceUpdated: () => {},
    onError: () => {},
    onSchemaMismatch: () => {},
    onForbiddenMember: () => {},
    onWsTransportError: () => {},
    onConnecting: () => {},
    onOpen: () => {},
    onClosed: () => {},
    onReconnectScheduled: () => {},
  });

  const {
    messages,
    setMessages,
    hasMoreOlder,
    setHasMoreOlder,
    loadingOlder,
    resetForRoom,
    loadOlderMessages,
    upsertMessage,
  } = useRoomMessages(id, setError);

  const read = useReadState(id, setError);
  const {
    readInfo,
    setReadInfo,
    readUsers,
    readStateError,
    setReadStateError,
    readStateUnreadCount,
    readStateMyLastReadId,
    markAsRead,
    applyReadPayload,
  } = read;

  const {
    roomMembers,
    setRoomMembers,
    roomMeta,
    setRoomMeta,
    directPeer,
    unreadCount: roomDetailUnread,
    myLastReadMessageId: roomDetailMyLastRead,
    loadRoomDetail,
  } = useRoomDetail(id);

  const displayUnread =
    roomDetailUnread != null
      ? roomDetailUnread
      : readStateUnreadCount != null
        ? readStateUnreadCount
        : null;
  const displayMyLastReadId = roomDetailMyLastRead ?? readStateMyLastReadId;

  const loadMessagesAndReadState = useCallback(async () => {
    if (!id) return;
    setError(null);
    setReadStateError(null);
    try {
      const [messageRes, readStateRes] = await Promise.all([
        fetchWithAuth(
          `/rooms/${id}/messages?order=desc&limit=${MESSAGE_PAGE_LIMIT}&include_read_count=1`,
        ),
        fetchWithAuth(`/rooms/${id}/read-state`),
      ]);
      if (messageRes.status === 401 || readStateRes.status === 401) {
        return;
      }
      if (!messageRes.ok) {
        setError(
          toErrorMessage(messageRes.status, "メッセージの取得に失敗しました。"),
        );
        return;
      }
      const messageData = await messageRes.json();
      const rawList: Message[] =
        messageData.messages ?? messageData.Messages ?? [];
      setHasMoreOlder(rawList.length === MESSAGE_PAGE_LIMIT);
      setMessages(sortMessagesAsc([...rawList].reverse()));

      if (readStateRes.ok) {
        const readData = (await readStateRes.json()) as ReadStateResponse;
        applyReadPayload(readData);
        setReadStateError(null);
      } else {
        setReadStateError(
          toErrorMessage(readStateRes.status, "既読状態の取得に失敗しました。"),
        );
      }
    } catch {
      setError("通信に失敗しました。接続を確認してください。");
    }
  }, [id, setMessages, setHasMoreOlder, applyReadPayload, setReadStateError]);

  const runLoadOlderWithScrollCapture = useCallback(async () => {
    const listEl = messageListRef.current;
    if (listEl) {
      scrollAfterOlderRef.current = {
        sh: listEl.scrollHeight,
        st: listEl.scrollTop,
      };
    }
    const ok = await loadOlderMessages();
    if (!ok) {
      scrollAfterOlderRef.current = null;
    }
  }, [loadOlderMessages]);

  const latestVisibleMessageId = useMemo(() => {
    for (let i = messages.length - 1; i >= 0; i--) {
      if (!messages[i].is_deleted) return messages[i].id;
    }
    return null;
  }, [messages]);

  const messageIndexMap = useMemo(() => {
    const map: Record<string, number> = {};
    messages.forEach((m, idx) => {
      map[m.id] = idx;
    });
    return map;
  }, [messages]);

  const readersByMessage = useMemo(() => {
    const readersMap: Record<string, ReaderInfo[]> = {};
    const entries = Object.entries(readInfo);
    messages.forEach((msg) => {
      const currentIndex = messageIndexMap[msg.id];
      if (currentIndex == null) return;
      const readers = entries
        .filter(([, lastReadMessageId]) => {
          const readIndex = messageIndexMap[lastReadMessageId];
          return readIndex != null && readIndex >= currentIndex;
        })
        .filter(([userId]) => userId !== selfUserId)
        .map(([userId, lastReadMessageId]) => ({
          userId,
          userName: readUsers[userId],
          lastReadMessageId,
        }));
      readersMap[msg.id] = readers;
    });
    return readersMap;
  }, [messages, messageIndexMap, readInfo, readUsers, selfUserId]);

  const roomTitle = useMemo(() => {
    if (roomMeta?.type === "direct" && directPeer) {
      return directPeer.name?.trim() || directPeer.user_id.slice(0, 8);
    }
    if (roomMeta?.type === "group") {
      return roomMeta.name?.trim() || "グループ";
    }
    return "ルーム";
  }, [roomMeta, directPeer]);

  const wsStatusLabel = useMemo(() => {
    if (wsStatus === "connecting") return "接続中";
    if (wsStatus === "open") return "接続済み";
    return "切断";
  }, [wsStatus]);

  const memberNameById = useMemo(() => {
    const m: Record<string, string> = {};
    roomMembers.forEach((x) => {
      m[x.user_id] = x.name?.trim() || x.user_id.slice(0, 8);
    });
    return m;
  }, [roomMembers]);

  const typingLabel = useMemo(() => {
    if (typingUserIds.length === 0) return "";
    return typingUserIds
      .map((uid) => memberNameById[uid] ?? readUsers[uid] ?? uid.slice(0, 8))
      .join("、");
  }, [typingUserIds, memberNameById, readUsers]);

  const pinnedPreview = useMemo(() => {
    if (!pin?.message_id) return null;
    const pm = messages.find((x) => x.id === pin.message_id);
    if (!pm) return { text: `メッセージ ${pin.message_id.slice(0, 8)}…` };
    const t = (
      pm.content ??
      pm.file_name ??
      (pm.image_url ? "[画像]" : "")
    ).slice(0, 120);
    return { text: t || "（内容なし）" };
  }, [pin, messages]);

  const navigateToLogin = useCallback(() => {
    navigate("/login", { replace: true });
  }, [navigate]);

  const refreshReactionsForOne = useCallback(
    async (messageId: string) => {
      if (!id) return;
      const res = await fetchWithAuth(
        `/rooms/${id}/messages/${messageId}/reactions`,
      );
      if (!res.ok) return;
      const data = await res.json().catch(() => ({}));
      const rows = (data.reactions ?? []) as ReactionRow[];
      setReactionsByMessage((prev) => ({
        ...prev,
        [messageId]: aggregateReactions(
          rows,
          selfUserId,
          (uid) => memberNameById[uid] ?? readUsers[uid] ?? uid.slice(0, 8),
        ),
      }));
    },
    [id, selfUserId, memberNameById, readUsers],
  );

  const refreshReactionsVisible = useCallback(async () => {
    if (!id) return;
    const mids = messages
      .filter((m) => !m.is_deleted)
      .slice(-24)
      .map((m) => m.id);
    const chunk = 6;
    for (let i = 0; i < mids.length; i += chunk) {
      const slice = mids.slice(i, i + chunk);
      await Promise.all(slice.map((mid) => refreshReactionsForOne(mid)));
    }
  }, [id, messages, refreshReactionsForOne]);

  handlersRef.current = {
    onSessionReady: (userId) => setSelfUserId(userId),
    onReloadData: () => {
      void loadMessagesAndReadState();
      void loadRoomDetail();
    },
    onMessageCreated: (msg) => upsertMessage(msg),
    onMessageUpdated: (msg) => upsertMessage(msg),
    onMessageDeleted: (messageId) => {
      setMessages((prev) => markMessageDeletedInList(prev, messageId));
    },
    onReadUpdated: (userId, lastReadMessageId) => {
      setReadInfo((prev) => ({
        ...prev,
        [userId]: lastReadMessageId,
      }));
    },
    onPresenceUpdated: (userId, isOnline) => {
      setMembersOnline((prev) => ({ ...prev, [userId]: isOnline }));
      setRoomMembers((prev) =>
        prev.map((m) =>
          m.user_id === userId ? { ...m, online: isOnline } : m,
        ),
      );
    },
    onError: (msg) => setError(msg),
    onSchemaMismatch: () => setError("WebSocket のスキーマ不一致です。"),
    onForbiddenMember: () => setError("このルームは閲覧できません"),
    onWsTransportError: () =>
      setError("WebSocket 接続に失敗しました。再接続を試行します。"),
    onConnecting: () => setWsStatus("connecting"),
    onOpen: () => {
      setWsStatus("open");
      setRetryCount(0);
    },
    onClosed: () => setWsStatus("closed"),
    onReconnectScheduled: (nextAttempt) => setRetryCount(nextAttempt),
    onTyping: (userId, isTyping) => {
      if (!selfUserId || userId === selfUserId) return;
      if (isTyping) {
        setTypingUserIds((prev) =>
          prev.includes(userId) ? prev : [...prev, userId],
        );
        const prevT = typingTimeoutsRef.current[userId];
        if (prevT != null) window.clearTimeout(prevT);
        typingTimeoutsRef.current[userId] = window.setTimeout(() => {
          setTypingUserIds((p) => p.filter((x) => x !== userId));
          delete typingTimeoutsRef.current[userId];
        }, 3500);
      } else {
        setTypingUserIds((p) => p.filter((x) => x !== userId));
        const prevT = typingTimeoutsRef.current[userId];
        if (prevT != null) window.clearTimeout(prevT);
        delete typingTimeoutsRef.current[userId];
      }
    },
  };

  const { sendTyping } = useRoomWebSocket(id, navigateToLogin, handlersRef);

  const notifyTypingActivity = useCallback(() => {
    sendTyping(true);
    if (typingIdleTimerRef.current != null)
      window.clearTimeout(typingIdleTimerRef.current);
    typingIdleTimerRef.current = window.setTimeout(() => {
      sendTyping(false);
      typingIdleTimerRef.current = null;
    }, 2000);
  }, [sendTyping]);

  useEffect(() => {
    return () => {
      if (typingIdleTimerRef.current != null)
        window.clearTimeout(typingIdleTimerRef.current);
      Object.values(typingTimeoutsRef.current).forEach((tid) =>
        window.clearTimeout(tid),
      );
      typingTimeoutsRef.current = {};
    };
  }, []);

  useEffect(() => {
    if (!id || messages.length === 0) return;
    const t = window.setTimeout(() => {
      void refreshReactionsVisible();
    }, 500);
    return () => window.clearTimeout(t);
  }, [id, messages, selfUserId, refreshReactionsVisible]);

  useEffect(() => {
    if (!id) return;
    setPin(null);
    void (async () => {
      const res = await fetchWithAuth(`/rooms/${id}/pin`);
      if (!res.ok) return;
      const data = await res.json().catch(() => ({}));
      const p = data.pin as { message_id?: string } | null | undefined;
      setPin(p?.message_id ? { message_id: String(p.message_id) } : null);
    })();
  }, [id]);

  useEffect(() => {
    if (!id) return;
    setBookmarkedByMessage({});
    void (async () => {
      const res = await fetchWithAuth("/me/bookmarks?limit=200");
      if (!res.ok) return;
      const data = await res.json().catch(() => ({}));
      const marks: Record<string, boolean> = {};
      for (const b of data.bookmarks ?? []) {
        if (String(b.room_id) === id) marks[String(b.message_id)] = true;
      }
      setBookmarkedByMessage(marks);
    })();
  }, [id]);

  useEffect(() => {
    setReplyTo(null);
    setReactionsByMessage({});
    setTypingUserIds([]);
  }, [id]);

  useLayoutEffect(() => {
    if (loadingOlder) return;
    const el = messageListRef.current;
    const anch = scrollAfterOlderRef.current;
    if (!el || !anch) return;
    scrollAfterOlderRef.current = null;
    el.scrollTop = el.scrollHeight - anch.sh + anch.st;
  }, [messages, loadingOlder]);

  useEffect(() => {
    if (!id) return;
    resetForRoom();
    initialScrolledRoomRef.current = null;
    scrollAfterOlderRef.current = null;
  }, [id, resetForRoom]);

  useEffect(() => {
    if (!id) return;
    setError(null);
    void Promise.all([loadMessagesAndReadState(), loadRoomDetail()]);
  }, [id, loadMessagesAndReadState, loadRoomDetail]);

  useEffect(() => {
    if (!id || roomMeta?.type !== "direct" || !directPeer?.user_id) {
      setPeerBlocked(false);
      return;
    }
    let alive = true;
    void (async () => {
      const res = await fetchWithAuth(`/users/${directPeer.user_id}/block`);
      if (!alive || !res.ok) return;
      const data = await res.json().catch(() => ({}));
      setPeerBlocked(Boolean(data.blocked));
    })();
    return () => {
      alive = false;
    };
  }, [id, roomMeta?.type, directPeer?.user_id]);

  useEffect(() => {
    const el = messageListRef.current;
    if (!id || !el || messages.length === 0) return;
    if (initialScrolledRoomRef.current === id) return;
    initialScrolledRoomRef.current = id;
    el.scrollTop = el.scrollHeight;
  }, [id, messages]);

  useEffect(() => {
    if (!id || !targetMessageId) {
      setHighlightedMessageId(null);
      return;
    }
    const found = messages.some((m) => m.id === targetMessageId);
    if (!found) {
      if (hasMoreOlder && !loadingOlder) {
        void runLoadOlderWithScrollCapture();
      }
      return;
    }
    const root = messageListRef.current;
    if (!root) return;
    const el = root.querySelector(
      `[data-message-id="${targetMessageId}"]`,
    ) as HTMLElement | null;
    if (!el) return;
    setHighlightedMessageId(targetMessageId);
    el.scrollIntoView({ behavior: "smooth", block: "center" });
    const tid = window.setTimeout(() => {
      setHighlightedMessageId((prev) =>
        prev === targetMessageId ? null : prev,
      );
    }, 2200);
    return () => window.clearTimeout(tid);
  }, [
    id,
    targetMessageId,
    messages,
    hasMoreOlder,
    loadingOlder,
    runLoadOlderWithScrollCapture,
  ]);

  useEffect(() => {
    if (!id || !latestVisibleMessageId) return;
    void markAsRead(latestVisibleMessageId);
    if (readSyncTimerRef.current != null) {
      window.clearTimeout(readSyncTimerRef.current);
    }
    readSyncTimerRef.current = window.setTimeout(() => {
      readSyncTimerRef.current = null;
      void loadRoomDetail();
    }, 700);
    return () => {
      if (readSyncTimerRef.current != null) {
        window.clearTimeout(readSyncTimerRef.current);
      }
    };
  }, [id, latestVisibleMessageId, markAsRead, loadRoomDetail]);

  useEffect(() => {
    if (!id) return;
    const onBrowserOnline = () => {
      void loadMessagesAndReadState();
      void loadRoomDetail();
    };
    const onVisibility = () => {
      if (document.visibilityState === "visible") {
        void loadMessagesAndReadState();
        void loadRoomDetail();
      }
    };
    window.addEventListener("online", onBrowserOnline);
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      window.removeEventListener("online", onBrowserOnline);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [id, loadMessagesAndReadState, loadRoomDetail]);

  const membersDisplay = useMemo(
    () =>
      roomMembers.map((m) => ({
        ...m,
        online: Object.prototype.hasOwnProperty.call(membersOnline, m.user_id)
          ? membersOnline[m.user_id]!
          : m.online,
      })),
    [roomMembers, membersOnline],
  );

  const handleScroll = () => {
    const el = messageListRef.current;
    if (!el) return;
    if (latestVisibleMessageId) {
      const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 16;
      if (atBottom) {
        void markAsRead(latestVisibleMessageId);
      }
    }
    if (
      el.scrollTop < 48 &&
      hasMoreOlder &&
      !loadingOlder &&
      messages.length > 0 &&
      !olderScrollLockRef.current
    ) {
      olderScrollLockRef.current = true;
      void runLoadOlderWithScrollCapture().finally(() => {
        olderScrollLockRef.current = false;
      });
    }
  };

  const isRoomCreator = Boolean(
    selfUserId && roomMeta && roomMeta.created_by === selfUserId,
  );
  const isGroup = roomMeta?.type === "group";
  const canRenameGroup = Boolean(
    isGroup && isRoomCreator && roomMeta && !roomMeta.is_frozen,
  );
  const canAddMember = Boolean(isGroup && roomMeta && !roomMeta.is_frozen);
  const canCloseRoom = Boolean(
    roomMeta &&
      ((roomMeta.type === "group" && isRoomCreator) ||
        roomMeta.type === "direct") &&
      !roomMeta.is_frozen,
  );

  const handleEditMessage = async (messageId: string, content: string) => {
    if (!id || !content.trim()) return;
    setError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${id}/messages/${messageId}`, {
        method: "PATCH",
        body: JSON.stringify({ content: content.trim() }),
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(
          toErrorMessage(
            res.status,
            (data as { error?: string }).error ?? "編集に失敗しました。",
          ),
        );
        return;
      }
      const data = await res.json().catch(() => ({}));
      if (data.message) upsertMessage(data.message as Message);
      requestRoomsRefresh();
    } catch {
      setError("通信エラーが発生しました。");
    }
  };

  const handleDeleteMessage = async (messageId: string) => {
    if (!id || !messageId || deletingId) return;
    setDeletingId(messageId);
    setError(null);
    try {
      const res = await fetchWithAuth(`/messages/${messageId}`, {
        method: "DELETE",
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(
          toErrorMessage(
            res.status,
            (data as { error?: string }).error ?? "削除に失敗しました。",
          ),
        );
        return;
      }
      setMessages((prev) => markMessageDeletedInList(prev, messageId));
      void loadRoomDetail();
      requestRoomsRefresh();
    } finally {
      setDeletingId(null);
    }
  };

  const handleToggleReaction = useCallback(
    async (messageId: string, emoji: string) => {
      if (!id || !emoji) return;
      const chips = reactionsByMessage[messageId] ?? [];
      const mine = chips.find((c) => c.emoji === emoji && c.mine);
      const res = mine
        ? await fetchWithAuth(
            `/rooms/${id}/messages/${messageId}/reactions?emoji=${encodeURIComponent(emoji)}`,
            { method: "DELETE" },
          )
        : await fetchWithAuth(`/rooms/${id}/messages/${messageId}/reactions`, {
            method: "POST",
            body: JSON.stringify({ emoji }),
          });
      if (res.status === 401) return;
      if (!res.ok) return;
      await refreshReactionsForOne(messageId);
    },
    [id, reactionsByMessage, refreshReactionsForOne],
  );

  const handlePinMessage = async (messageId: string) => {
    if (!id) return;
    const res = await fetchWithAuth(`/rooms/${id}/pin`, {
      method: "POST",
      body: JSON.stringify({ message_id: messageId }),
    });
    if (res.status === 401) return;
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      setError(
        toErrorMessage(
          res.status,
          (data as { error?: string }).error ?? "ピンに失敗しました。",
        ),
      );
      return;
    }
    setPin({ message_id: messageId });
  };

  const handleClearPin = async () => {
    if (!id) return;
    const res = await fetchWithAuth(`/rooms/${id}/pin`, { method: "DELETE" });
    if (res.status === 401) return;
    if (res.ok) setPin(null);
  };

  const handleToggleBookmark = async (messageId: string) => {
    const on = bookmarkedByMessage[messageId];
    const res = await fetchWithAuth(`/messages/${messageId}/bookmark`, {
      method: on ? "DELETE" : "POST",
    });
    if (res.status === 401) return;
    if (!res.ok) return;
    setBookmarkedByMessage((prev) => ({ ...prev, [messageId]: !on }));
  };

  const scrollToPinnedMessage = () => {
    if (!pin?.message_id) return;
    const root = messageListRef.current;
    if (!root) return;
    const el = root.querySelector(`[data-message-id="${pin.message_id}"]`);
    (el as HTMLElement | null)?.scrollIntoView({
      behavior: "smooth",
      block: "center",
    });
  };

  const { handleSend, handleImageFile, handleGenericFile } =
    useRoomComposerActions({
      roomId: id,
      input,
      replyTo,
      setInput,
      setReplyTo,
      setError,
      setLoading,
      setUploading,
      upsertMessage,
      loadRoomDetail,
    });

  const {
    patchRoomName,
    addMember,
    removeMember,
    closeRoom,
    toggleDirectPeerBlock,
  } = useRoomSettingsActions({
    roomId: id,
    renameDraft,
    addMemberRef,
    selfUserId,
    roomType: roomMeta?.type,
    directPeerUserId: directPeer?.user_id,
    peerBlocked,
    blockBusy,
    setRoomActionBusy,
    setRoomModalError,
    setRoomMeta,
    setAddMemberRef,
    setShowRoomModal,
    setBlockBusy,
    setPeerBlocked,
    loadRoomDetail,
    loadMessagesAndReadState,
    onExitRoom: () => navigate("/rooms"),
  });

  const handleConfirmAction = async () => {
    if (!confirmAction) return;
    const action = confirmAction;
    setConfirmAction(null);
    if (action.type === "close-room") {
      await closeRoom();
      return;
    }
    if (action.type === "leave-room") {
      await removeMember(action.userId);
      return;
    }
    await removeMember(action.userId);
  };

  if (!id) {
    return <div className="chat-empty">ルームを選択してください</div>;
  }

  return (
    <div className="chat-pane">
      <div className="room-toolbar">
        <h2 className="room-toolbar-title">{roomTitle}</h2>
        {roomMeta?.type && (
          <span
            className={`room-kind-badge ${roomMeta.type === "direct" ? "is-direct" : "is-group"}`}
          >
            {roomMeta.type === "direct" ? "1対1" : "グループ"}
          </span>
        )}
        {roomMeta?.is_frozen && <span className="room-frozen-badge">凍結</span>}
        <button
          type="button"
          className="pill-button"
          onClick={() => {
            setShowRoomModal(true);
            setRoomModalError(null);
            setRenameDraft(roomMeta?.name ?? "");
            setAddMemberRef("");
          }}
        >
          ルーム設定
        </button>
      </div>
      {pin && pinnedPreview && (
        <div className="pinned-banner">
          <span>ピン留め: {pinnedPreview.text}</span>
          <span>
            <button
              type="button"
              className="pill-button"
              onClick={scrollToPinnedMessage}
            >
              移動
            </button>{" "}
            <button
              type="button"
              className="pill-button"
              onClick={() => void handleClearPin()}
            >
              ピン解除
            </button>
          </span>
        </div>
      )}
      {typingLabel ? (
        <div className="typing-banner">入力中: {typingLabel}</div>
      ) : null}
      <div
        className="room-detail-strip"
        aria-label="ルーム詳細（GET /rooms/:id および既読APIの要約）"
      >
        {roomMeta?.type === "direct" && directPeer && (
          <span className="room-detail-item" title={directPeer.user_id}>
            相手: {directPeer.name?.trim() || directPeer.user_id.slice(0, 8)}
          </span>
        )}
        <span className="room-detail-item">
          未読: {displayUnread != null ? String(displayUnread) : "—"}
        </span>
        <span className="room-detail-item" title={displayMyLastReadId ?? ""}>
          自分の最終既読:{" "}
          {displayMyLastReadId ? `${displayMyLastReadId.slice(0, 8)}…` : "—"}
        </span>
      </div>
      <MessageList
        messageListRef={messageListRef}
        messages={messages}
        hasMoreOlder={hasMoreOlder}
        loadingOlder={loadingOlder}
        onScroll={handleScroll}
        onLoadOlderClick={() => void runLoadOlderWithScrollCapture()}
        latestVisibleMessageId={latestVisibleMessageId}
        readersByMessage={readersByMessage}
        selfUserId={selfUserId}
        deletingId={deletingId}
        onDeleteMessage={handleDeleteMessage}
        onEditMessage={handleEditMessage}
        reactionsByMessage={reactionsByMessage}
        onToggleReaction={handleToggleReaction}
        onReply={setReplyTo}
        onPinMessage={handlePinMessage}
        onClearPinMessage={() => void handleClearPin()}
        pinnedMessageId={pin?.message_id ?? null}
        bookmarkedByMessage={bookmarkedByMessage}
        onToggleBookmark={handleToggleBookmark}
        showGroupReadCounts={roomMeta?.type === "group"}
        highlightedMessageId={highlightedMessageId}
      />
      <div className="chat-status">
        <div className="chat-status-row chat-status-members-block">
          <span className="chat-status-label">メンバーと接続状態</span>
          <ul className="member-presence-list">
            {membersDisplay.length === 0 ? (
              <li className="muted">—</li>
            ) : (
              membersDisplay.map((m) => (
                <li key={m.user_id} className="member-presence-row">
                  <span className="member-presence-name" title={m.user_id}>
                    {m.name?.trim() || m.user_id.slice(0, 8)}
                  </span>
                  <span
                    className={`presence-pill${m.online ? " is-online" : " is-offline"}`}
                  >
                    {m.online ? "オンライン" : "オフライン"}
                  </span>
                </li>
              ))
            )}
          </ul>
        </div>
        <div className="chat-status-row compact">
          <span>既読メンバー: {Object.keys(readInfo).length}</span>
          <span>通信: {wsStatusLabel}</span>
          {retryCount > 0 && <span>再接続試行: {retryCount}</span>}
        </div>
      </div>
      {readStateError && (
        <div className="error-inline-row">
          <span className="error-text">{readStateError}</span>
          <button
            className="pill-button"
            type="button"
            onClick={() => void loadMessagesAndReadState()}
          >
            再取得
          </button>
        </div>
      )}
      <MessageComposer
        input={input}
        onInputChange={setInput}
        onSend={() => void handleSend()}
        onImageSelected={(file) => void handleImageFile(file)}
        onGenericFileSelected={(file) => void handleGenericFile(file)}
        onTypingActivity={notifyTypingActivity}
        loading={loading}
        uploading={uploading}
        replyTo={replyTo}
        onClearReply={() => setReplyTo(null)}
      />
      {error && <div className="error-text">{error}</div>}
      <RoomSettingsModal
        open={showRoomModal}
        roomActionBusy={roomActionBusy}
        roomModalError={roomModalError}
        roomMeta={
          roomMeta && (roomMeta.type === "direct" || roomMeta.type === "group")
            ? { type: roomMeta.type, is_frozen: roomMeta.is_frozen }
            : null
        }
        directPeer={directPeer ? { user_id: directPeer.user_id } : null}
        peerBlocked={peerBlocked}
        blockBusy={blockBusy}
        canRenameGroup={canRenameGroup}
        renameDraft={renameDraft}
        canAddMember={canAddMember}
        addMemberRef={addMemberRef}
        roomMembers={roomMembers}
        selfUserId={selfUserId}
        isGroup={isGroup}
        isRoomCreator={isRoomCreator}
        canCloseRoom={canCloseRoom}
        onClose={() => setShowRoomModal(false)}
        onRenameDraftChange={setRenameDraft}
        onPatchRoomName={() => void patchRoomName()}
        onAddMemberRefChange={setAddMemberRef}
        onAddMember={() => void addMember()}
        onToggleDirectPeerBlock={() => void toggleDirectPeerBlock()}
        onOpenLeaveConfirm={(userId) =>
          setConfirmAction({ type: "leave-room", userId })
        }
        onOpenRemoveConfirm={(userId) =>
          setConfirmAction({ type: "remove-member", userId })
        }
        onOpenCloseConfirm={() => setConfirmAction({ type: "close-room" })}
      />
      <ConfirmActionModal
        confirmAction={confirmAction}
        roomActionBusy={roomActionBusy}
        onClose={() => setConfirmAction(null)}
        onConfirm={() => void handleConfirmAction()}
      />
    </div>
  );
}
