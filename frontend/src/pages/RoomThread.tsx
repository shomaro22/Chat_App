import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { fetchWithAuth } from '../lib/api';
import { requestRoomsRefresh } from '../lib/roomsRefresh';
import { toErrorMessage } from '../chat/errors';
import { markMessageDeletedInList, sortMessagesAsc } from '../chat/messageUtils';
import { MessageComposer } from '../components/MessageComposer';
import { MessageList } from '../components/MessageList';
import { useReadState } from '../hooks/useReadState';
import { useRoomDetail } from '../hooks/useRoomDetail';
import { useRoomMessages } from '../hooks/useRoomMessages';
import { useRoomWebSocket, type RoomWsHandlerBag } from '../hooks/useRoomWebSocket';
import type { Message, ReaderInfo, ReadStateResponse, ReactionChip, ReactionRow } from '../types/chat';
import { MESSAGE_PAGE_LIMIT } from '../types/chat';

function aggregateReactions(rows: ReactionRow[], selfId: string | null): ReactionChip[] {
  const map = new Map<string, { count: number; mine: boolean }>();
  for (const r of rows) {
    const cur = map.get(r.emoji) ?? { count: 0, mine: false };
    cur.count++;
    if (selfId && r.user_id === selfId) cur.mine = true;
    map.set(r.emoji, cur);
  }
  return [...map.entries()].map(([emoji, v]) => ({ emoji, count: v.count, mine: v.mine }));
}

export function RoomThread() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [selfUserId, setSelfUserId] = useState<string | null>(null);
  const [membersOnline, setMembersOnline] = useState<Record<string, boolean>>({});
  const [wsStatus, setWsStatus] = useState<'connecting' | 'open' | 'closed'>('connecting');
  const [retryCount, setRetryCount] = useState(0);
  const [showRoomModal, setShowRoomModal] = useState(false);
  const [roomModalError, setRoomModalError] = useState<string | null>(null);
  const [renameDraft, setRenameDraft] = useState('');
  const [addMemberRef, setAddMemberRef] = useState('');
  const [roomActionBusy, setRoomActionBusy] = useState(false);
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [reactionsByMessage, setReactionsByMessage] = useState<Record<string, ReactionChip[]>>({});
  const [pin, setPin] = useState<{ message_id: string } | null>(null);
  const [bookmarkedByMessage, setBookmarkedByMessage] = useState<Record<string, boolean>>({});
  const [typingUserIds, setTypingUserIds] = useState<string[]>([]);
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
    roomDetailUnread != null ? roomDetailUnread : readStateUnreadCount != null ? readStateUnreadCount : null;
  const displayMyLastReadId = roomDetailMyLastRead ?? readStateMyLastReadId;

  const loadMessagesAndReadState = useCallback(async () => {
    if (!id) return;
    setError(null);
    setReadStateError(null);
    try {
      const [messageRes, readStateRes] = await Promise.all([
        fetchWithAuth(`/rooms/${id}/messages?order=desc&limit=${MESSAGE_PAGE_LIMIT}&include_read_count=1`),
        fetchWithAuth(`/rooms/${id}/read-state`),
      ]);
      if (messageRes.status === 401 || readStateRes.status === 401) {
        return;
      }
      if (!messageRes.ok) {
        setError(toErrorMessage(messageRes.status, 'メッセージの取得に失敗しました。'));
        return;
      }
      const messageData = await messageRes.json();
      const rawList: Message[] = messageData.messages ?? messageData.Messages ?? [];
      setHasMoreOlder(rawList.length === MESSAGE_PAGE_LIMIT);
      setMessages(sortMessagesAsc([...rawList].reverse()));

      if (readStateRes.ok) {
        const readData = (await readStateRes.json()) as ReadStateResponse;
        applyReadPayload(readData);
        setReadStateError(null);
      } else {
        setReadStateError(toErrorMessage(readStateRes.status, '既読状態の取得に失敗しました。'));
      }
    } catch {
      setError('通信に失敗しました。接続を確認してください。');
    }
  }, [id, setMessages, setHasMoreOlder, applyReadPayload, setReadStateError]);

  const runLoadOlderWithScrollCapture = useCallback(async () => {
    const listEl = messageListRef.current;
    if (listEl) {
      scrollAfterOlderRef.current = { sh: listEl.scrollHeight, st: listEl.scrollTop };
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
    if (roomMeta?.type === 'direct' && directPeer) {
      return directPeer.name?.trim() || directPeer.user_id.slice(0, 8);
    }
    if (roomMeta?.type === 'group') {
      return roomMeta.name?.trim() || 'グループ';
    }
    return 'ルーム';
  }, [roomMeta, directPeer]);

  const wsStatusLabel = useMemo(() => {
    if (wsStatus === 'connecting') return '接続中';
    if (wsStatus === 'open') return '接続済み';
    return '切断';
  }, [wsStatus]);

  const memberNameById = useMemo(() => {
    const m: Record<string, string> = {};
    roomMembers.forEach((x) => {
      m[x.user_id] = x.name?.trim() || x.user_id.slice(0, 8);
    });
    return m;
  }, [roomMembers]);

  const typingLabel = useMemo(() => {
    if (typingUserIds.length === 0) return '';
    return typingUserIds
      .map((uid) => memberNameById[uid] ?? readUsers[uid] ?? uid.slice(0, 8))
      .join('、');
  }, [typingUserIds, memberNameById, readUsers]);

  const pinnedPreview = useMemo(() => {
    if (!pin?.message_id) return null;
    const pm = messages.find((x) => x.id === pin.message_id);
    if (!pm) return { text: `メッセージ ${pin.message_id.slice(0, 8)}…` };
    const t = (pm.content ?? pm.file_name ?? (pm.image_url ? '[画像]' : '')).slice(0, 120);
    return { text: t || '（内容なし）' };
  }, [pin, messages]);

  const navigateToLogin = useCallback(() => {
    navigate('/login', { replace: true });
  }, [navigate]);

  const refreshReactionsForOne = useCallback(
    async (messageId: string) => {
      if (!id) return;
      const res = await fetchWithAuth(`/rooms/${id}/messages/${messageId}/reactions`);
      if (!res.ok) return;
      const data = await res.json().catch(() => ({}));
      const rows = (data.reactions ?? []) as ReactionRow[];
      setReactionsByMessage((prev) => ({
        ...prev,
        [messageId]: aggregateReactions(rows, selfUserId),
      }));
    },
    [id, selfUserId],
  );

  const refreshReactionsVisible = useCallback(async () => {
    if (!id) return;
    const mids = messages.filter((m) => !m.is_deleted).slice(-24).map((m) => m.id);
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
      setRoomMembers((prev) => prev.map((m) => (m.user_id === userId ? { ...m, online: isOnline } : m)));
    },
    onError: (msg) => setError(msg),
    onSchemaMismatch: () => setError('WebSocket のスキーマ不一致です。'),
    onForbiddenMember: () => setError('このルームの購読権限がありません。'),
    onWsTransportError: () => setError('WebSocket 接続に失敗しました。再接続を試行します。'),
    onConnecting: () => setWsStatus('connecting'),
    onOpen: () => {
      setWsStatus('open');
      setRetryCount(0);
    },
    onClosed: () => setWsStatus('closed'),
    onReconnectScheduled: (nextAttempt) => setRetryCount(nextAttempt),
    onTyping: (userId, isTyping) => {
      if (!selfUserId || userId === selfUserId) return;
      if (isTyping) {
        setTypingUserIds((prev) => (prev.includes(userId) ? prev : [...prev, userId]));
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
    if (typingIdleTimerRef.current != null) window.clearTimeout(typingIdleTimerRef.current);
    typingIdleTimerRef.current = window.setTimeout(() => {
      sendTyping(false);
      typingIdleTimerRef.current = null;
    }, 2000);
  }, [sendTyping]);

  useEffect(() => {
    return () => {
      if (typingIdleTimerRef.current != null) window.clearTimeout(typingIdleTimerRef.current);
      Object.values(typingTimeoutsRef.current).forEach((tid) => window.clearTimeout(tid));
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
      const res = await fetchWithAuth('/me/bookmarks?limit=200');
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
    const el = messageListRef.current;
    if (!id || !el || messages.length === 0) return;
    if (initialScrolledRoomRef.current === id) return;
    initialScrolledRoomRef.current = id;
    el.scrollTop = el.scrollHeight;
  }, [id, messages]);

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
      if (document.visibilityState === 'visible') {
        void loadMessagesAndReadState();
        void loadRoomDetail();
      }
    };
    window.addEventListener('online', onBrowserOnline);
    document.addEventListener('visibilitychange', onVisibility);
    return () => {
      window.removeEventListener('online', onBrowserOnline);
      document.removeEventListener('visibilitychange', onVisibility);
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

  const isRoomCreator = Boolean(selfUserId && roomMeta && roomMeta.created_by === selfUserId);
  const isGroup = roomMeta?.type === 'group';
  const canRenameGroup = isGroup && isRoomCreator && roomMeta && !roomMeta.is_frozen;
  const canAddMember = isGroup && roomMeta && !roomMeta.is_frozen;
  const canCloseRoom =
    roomMeta &&
    ((roomMeta.type === 'group' && isRoomCreator) || roomMeta.type === 'direct') &&
    !roomMeta.is_frozen;

  const handleEditMessage = async (messageId: string, content: string) => {
    if (!id || !content.trim()) return;
    setError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${id}/messages/${messageId}`, {
        method: 'PATCH',
        body: JSON.stringify({ content: content.trim() }),
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(toErrorMessage(res.status, (data as { error?: string }).error ?? '編集に失敗しました。'));
        return;
      }
      const data = await res.json().catch(() => ({}));
      if (data.message) upsertMessage(data.message as Message);
      requestRoomsRefresh();
    } catch {
      setError('通信エラーが発生しました。');
    }
  };

  const handleDeleteMessage = async (messageId: string) => {
    if (!id || !messageId || deletingId) return;
    setDeletingId(messageId);
    setError(null);
    try {
      const res = await fetchWithAuth(`/messages/${messageId}`, { method: 'DELETE' });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(toErrorMessage(res.status, (data as { error?: string }).error ?? '削除に失敗しました。'));
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
            { method: 'DELETE' },
          )
        : await fetchWithAuth(`/rooms/${id}/messages/${messageId}/reactions`, {
            method: 'POST',
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
      method: 'POST',
      body: JSON.stringify({ message_id: messageId }),
    });
    if (res.status === 401) return;
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      setError(toErrorMessage(res.status, (data as { error?: string }).error ?? 'ピンに失敗しました。'));
      return;
    }
    setPin({ message_id: messageId });
  };

  const handleClearPin = async () => {
    if (!id) return;
    const res = await fetchWithAuth(`/rooms/${id}/pin`, { method: 'DELETE' });
    if (res.status === 401) return;
    if (res.ok) setPin(null);
  };

  const handleToggleBookmark = async (messageId: string) => {
    const on = bookmarkedByMessage[messageId];
    const res = await fetchWithAuth(`/messages/${messageId}/bookmark`, { method: on ? 'DELETE' : 'POST' });
    if (res.status === 401) return;
    if (!res.ok) return;
    setBookmarkedByMessage((prev) => ({ ...prev, [messageId]: !on }));
  };

  const scrollToPinnedMessage = () => {
    if (!pin?.message_id) return;
    const root = messageListRef.current;
    if (!root) return;
    const el = root.querySelector(`[data-message-id="${pin.message_id}"]`);
    (el as HTMLElement | null)?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  };

  const handleSend = async () => {
    if (!id || !input.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const body: Record<string, string> = { content: input.trim() };
      if (replyTo) body.parent_message_id = replyTo.id;
      const res = await fetchWithAuth(`/rooms/${id}/messages`, {
        method: 'POST',
        body: JSON.stringify(body),
      });
      const data = await res.json().catch(() => ({}));
      if (res.ok && data.message) {
        upsertMessage(data.message as Message);
        setInput('');
        setReplyTo(null);
        void loadRoomDetail();
        requestRoomsRefresh();
      } else if (!res.ok) {
        if (res.status === 401) return;
        setError(toErrorMessage(res.status, 'メッセージ送信に失敗しました。'));
      }
    } finally {
      setLoading(false);
    }
  };

  const handleImageFile = async (file: File) => {
    if (!id) return;
    setUploading(true);
    setError(null);
    try {
      const form = new FormData();
      form.append('file', file);
      const uploadRes = await fetchWithAuth('/media/images', {
        method: 'POST',
        body: form,
      });
      const uploadData = await uploadRes.json().catch(() => ({}));
      if (!uploadRes.ok || !uploadData.url) {
        if (uploadRes.status === 401) return;
        setError(toErrorMessage(uploadRes.status, '画像アップロードに失敗しました。'));
        return;
      }
      const sendBody: Record<string, string> = { image_url: uploadData.url };
      const cap = input.trim();
      if (cap) sendBody.content = cap;
      if (replyTo) sendBody.parent_message_id = replyTo.id;
      const sendRes = await fetchWithAuth(`/rooms/${id}/messages`, {
        method: 'POST',
        body: JSON.stringify(sendBody),
      });
      const sendData = await sendRes.json().catch(() => ({}));
      if (sendRes.ok && sendData.message) {
        upsertMessage(sendData.message as Message);
        setInput('');
        setReplyTo(null);
        void loadRoomDetail();
        requestRoomsRefresh();
      } else if (!sendRes.ok) {
        if (sendRes.status === 401) return;
        setError(toErrorMessage(sendRes.status, '画像メッセージ送信に失敗しました。'));
      }
    } finally {
      setUploading(false);
    }
  };

  const handleGenericFile = async (file: File) => {
    if (!id) return;
    setUploading(true);
    setError(null);
    try {
      const form = new FormData();
      form.append('file', file);
      const uploadRes = await fetchWithAuth('/media/files', {
        method: 'POST',
        body: form,
      });
      const uploadData = await uploadRes.json().catch(() => ({}));
      if (!uploadRes.ok || !uploadData.url) {
        if (uploadRes.status === 401) return;
        setError(toErrorMessage(uploadRes.status, 'ファイルのアップロードに失敗しました。'));
        return;
      }
      const caption = input.trim();
      const sendBody: Record<string, string> = {
        file_url: uploadData.url,
        file_name: uploadData.filename ?? file.name,
        file_content_type: uploadData.content_type ?? file.type,
      };
      if (caption) sendBody.content = caption;
      if (replyTo) sendBody.parent_message_id = replyTo.id;
      const sendRes = await fetchWithAuth(`/rooms/${id}/messages`, {
        method: 'POST',
        body: JSON.stringify(sendBody),
      });
      const sendData = await sendRes.json().catch(() => ({}));
      if (sendRes.ok && sendData.message) {
        upsertMessage(sendData.message as Message);
        setInput('');
        setReplyTo(null);
        void loadRoomDetail();
        requestRoomsRefresh();
      } else if (!sendRes.ok) {
        if (sendRes.status === 401) return;
        setError(toErrorMessage(sendRes.status, 'ファイル付きメッセージの送信に失敗しました。'));
      }
    } finally {
      setUploading(false);
    }
  };

  const patchRoomName = async () => {
    if (!id || !renameDraft.trim()) return;
    setRoomActionBusy(true);
    setRoomModalError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ name: renameDraft.trim() }),
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomModalError(toErrorMessage(res.status, (data as { error?: string }).error ?? '更新に失敗しました。'));
        return;
      }
      setRoomMeta((m) => (m ? { ...m, name: renameDraft.trim() } : m));
      requestRoomsRefresh();
    } finally {
      setRoomActionBusy(false);
    }
  };

  const addMember = async () => {
    if (!id || !addMemberRef.trim()) return;
    setRoomActionBusy(true);
    setRoomModalError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${id}/members`, {
        method: 'POST',
        body: JSON.stringify({ member_id: addMemberRef.trim() }),
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomModalError(toErrorMessage(res.status, (data as { error?: string }).error ?? '追加に失敗しました。'));
        return;
      }
      setAddMemberRef('');
      await loadRoomDetail();
      requestRoomsRefresh();
    } finally {
      setRoomActionBusy(false);
    }
  };

  const removeMember = async (targetUserId: string) => {
    if (!id || !targetUserId) return;
    setRoomActionBusy(true);
    setRoomModalError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${id}/members/${targetUserId}`, { method: 'DELETE' });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomModalError(toErrorMessage(res.status, (data as { error?: string }).error ?? '操作に失敗しました。'));
        return;
      }
      if (selfUserId === targetUserId) {
        setShowRoomModal(false);
        requestRoomsRefresh();
        navigate('/rooms');
        return;
      }
      await loadRoomDetail();
      requestRoomsRefresh();
    } finally {
      setRoomActionBusy(false);
    }
  };

  const closeRoom = async () => {
    if (!id) return;
    if (!window.confirm('このルームを閉じますか？（メンバー全員がアクセスできなくなります）')) return;
    setRoomActionBusy(true);
    setRoomModalError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${id}`, { method: 'DELETE' });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomModalError(toErrorMessage(res.status, (data as { error?: string }).error ?? 'クローズに失敗しました。'));
        return;
      }
      setShowRoomModal(false);
      requestRoomsRefresh();
      navigate('/rooms');
    } finally {
      setRoomActionBusy(false);
    }
  };

  if (!id) {
    return (
      <div className="chat-empty">
        ルームを選択してください
      </div>
    );
  }

  return (
    <div className="chat-pane">
      <div className="room-toolbar">
        <h2 className="room-toolbar-title">{roomTitle}</h2>
        {roomMeta?.is_frozen && <span className="room-frozen-badge">凍結</span>}
        <button
          type="button"
          className="pill-button"
          onClick={() => {
            setShowRoomModal(true);
            setRoomModalError(null);
            setRenameDraft(roomMeta?.name ?? '');
            setAddMemberRef('');
          }}
        >
          ルーム設定
        </button>
      </div>
      {pin && pinnedPreview && (
        <div className="pinned-banner">
          <span>
            ピン留め: {pinnedPreview.text}
          </span>
          <span>
            <button type="button" className="pill-button" onClick={scrollToPinnedMessage}>
              移動
            </button>{' '}
            <button type="button" className="pill-button" onClick={() => void handleClearPin()}>
              ピン解除
            </button>
          </span>
        </div>
      )}
      {typingLabel ? <div className="typing-banner">入力中: {typingLabel}</div> : null}
      <div className="room-detail-strip" aria-label="ルーム詳細（GET /rooms/:id および既読APIの要約）">
        {roomMeta?.type === 'direct' && directPeer && (
          <span className="room-detail-item" title={directPeer.user_id}>
            相手: {directPeer.name?.trim() || directPeer.user_id.slice(0, 8)}
          </span>
        )}
        <span className="room-detail-item">未読: {displayUnread != null ? String(displayUnread) : '—'}</span>
        <span className="room-detail-item" title={displayMyLastReadId ?? ''}>
          自分の最終既読:{' '}
          {displayMyLastReadId ? `${displayMyLastReadId.slice(0, 8)}…` : '—'}
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
        bookmarkedByMessage={bookmarkedByMessage}
        onToggleBookmark={handleToggleBookmark}
        showGroupReadCounts={roomMeta?.type === 'group'}
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
                  <span className={`presence-pill${m.online ? ' is-online' : ' is-offline'}`}>
                    {m.online ? 'オンライン' : 'オフライン'}
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
          <button className="pill-button" type="button" onClick={() => void loadMessagesAndReadState()}>
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
      {showRoomModal && (
        <div
          className="modal-overlay"
          onClick={() => {
            if (!roomActionBusy) setShowRoomModal(false);
          }}
        >
          <div className="modal-card modal-card-room" onClick={(e) => e.stopPropagation()}>
            <h3>ルーム設定</h3>
            {roomModalError && <div className="error-text">{roomModalError}</div>}
            {roomMeta?.is_frozen && (
              <p className="muted">このルームは凍結中です。名前変更・メンバー追加・送信はサーバー側の制限に従います。</p>
            )}
            {canRenameGroup && (
              <div className="room-settings-block">
                <label className="room-settings-label">グループ名</label>
                <input
                  className="room-create-input"
                  value={renameDraft}
                  onChange={(e) => setRenameDraft(e.target.value)}
                  disabled={roomActionBusy}
                />
                <button
                  type="button"
                  className="pill-button"
                  disabled={roomActionBusy || !renameDraft.trim()}
                  onClick={() => void patchRoomName()}
                >
                  名前を更新
                </button>
              </div>
            )}
            {canAddMember && (
              <div className="room-settings-block">
                <label className="room-settings-label">メンバー追加（メールまたはユーザー ID）</label>
                <input
                  className="room-create-input"
                  value={addMemberRef}
                  onChange={(e) => setAddMemberRef(e.target.value)}
                  placeholder="メールまたはユーザーID（API: member_id）"
                  disabled={roomActionBusy}
                />
                <button
                  type="button"
                  className="pill-button"
                  disabled={roomActionBusy || !addMemberRef.trim()}
                  onClick={() => void addMember()}
                >
                  追加
                </button>
              </div>
            )}
            <div className="room-settings-block">
              <div className="room-settings-label">メンバー</div>
              <ul className="room-member-actions">
                {roomMembers.map((m) => {
                  const isSelf = selfUserId === m.user_id;
                  const canKick = isGroup && isRoomCreator && !isSelf && !roomMeta?.is_frozen;
                  return (
                    <li key={m.user_id} className="room-member-actions-row">
                      <span>
                        {m.name || m.user_id.slice(0, 8)}
                        {m.online ? ' ●' : ''}
                      </span>
                      <span className="room-member-actions-btns">
                        {isSelf && (
                          <button
                            type="button"
                            className="pill-button danger-outline"
                            disabled={roomActionBusy}
                            onClick={() => {
                              if (window.confirm('このルームから退出しますか？')) void removeMember(m.user_id);
                            }}
                          >
                            退出
                          </button>
                        )}
                        {canKick && (
                          <button
                            type="button"
                            className="pill-button danger-outline"
                            disabled={roomActionBusy}
                            onClick={() => {
                              if (window.confirm('このメンバーをキックしますか？')) void removeMember(m.user_id);
                            }}
                          >
                            キック
                          </button>
                        )}
                      </span>
                    </li>
                  );
                })}
              </ul>
            </div>
            {canCloseRoom && (
              <div className="room-settings-block">
                <button
                  type="button"
                  className="pill-button danger-outline"
                  disabled={roomActionBusy}
                  onClick={() => void closeRoom()}
                >
                  ルームを閉じる
                </button>
              </div>
            )}
            <div className="modal-actions">
              <button
                type="button"
                className="pill-button"
                disabled={roomActionBusy}
                onClick={() => setShowRoomModal(false)}
              >
                閉じる
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
