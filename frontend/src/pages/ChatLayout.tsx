import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, Route, Routes, useNavigate, useParams } from 'react-router-dom';
import { fetchWithAuth, getToken, setToken } from '../lib/api';

type RoomSummary = {
  id: string;
  name?: string | null;
  direct_peer_name?: string | null;
  unread_count?: number;
  last_message?: {
    preview?: string | null;
  };
};

type Message = {
  id: string;
  room_id: string;
  content?: string | null;
  image_url?: string | null;
  sender_id?: string;
  sender_name?: string;
  created_at: string;
};

type ReadStateResponse = {
  my_last_read_message_id?: string;
  unread_count?: number;
  member_reads?: Array<{
    user_id: string;
    user_name?: string;
    last_read_message_id?: string;
  }>;
};

type ReaderInfo = {
  userId: string;
  userName?: string;
  lastReadMessageId: string;
};

function toErrorMessage(status: number, fallback: string): string {
  if (status === 401) return '認証が切れました。再ログインしてください。';
  if (status === 403) return 'この操作は許可されていません。';
  if (status === 404) return '対象が見つかりません。';
  if (status === 429) return 'リクエストが多すぎます。少し待って再試行してください。';
  return fallback;
}

function RoomMessages() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [messages, setMessages] = useState<Message[]>([]);
  const [membersOnline, setMembersOnline] = useState<Record<string, boolean>>({});
  const [readInfo, setReadInfo] = useState<Record<string, string>>({});
  const [readUsers, setReadUsers] = useState<Record<string, string>>({});
  const [selfUserId, setSelfUserId] = useState<string | null>(null);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [readStateError, setReadStateError] = useState<string | null>(null);
  const [wsStatus, setWsStatus] = useState<'connecting' | 'open' | 'closed'>('connecting');
  const [retryCount, setRetryCount] = useState(0);
  const messageListRef = useRef<HTMLDivElement | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const shouldReconnectRef = useRef(true);

  const latestMessageId = useMemo(
    () => (messages.length > 0 ? messages[messages.length - 1].id : null),
    [messages],
  );

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

  const appendMessageUnique = (nextMessage: Message) => {
    setMessages((prev) => {
      if (prev.some((m) => m.id === nextMessage.id)) {
        return prev;
      }
      return [...prev, nextMessage];
    });
  };

  const markAsRead = async (messageId: string) => {
    if (!id || !messageId) return;
    const res = await fetchWithAuth(`/rooms/${id}/read`, {
      method: 'POST',
      body: JSON.stringify({ message_id: messageId }),
    });
    if (!res.ok && res.status !== 409) {
      if (res.status === 401) {
        setToken(null);
        navigate('/login', { replace: true });
        return;
      }
      setError(toErrorMessage(res.status, '既読更新に失敗しました。'));
    }
  };

  const loadMessagesAndReadState = async () => {
    if (!id) return;
    setError(null);
    setReadStateError(null);
    try {
      const [messageRes, readStateRes] = await Promise.all([
        fetchWithAuth(`/rooms/${id}/messages?order=asc`),
        fetchWithAuth(`/rooms/${id}/read-state`),
      ]);
      if (messageRes.status === 401 || readStateRes.status === 401) {
        setToken(null);
        navigate('/login', { replace: true });
        return;
      }
      if (!messageRes.ok) {
        setError(toErrorMessage(messageRes.status, 'メッセージの取得に失敗しました。'));
        return;
      }
      const messageData = await messageRes.json();
      setMessages(messageData.messages ?? messageData.Messages ?? []);

      if (readStateRes.ok) {
        const readData = (await readStateRes.json()) as ReadStateResponse;
        const nextReadInfo: Record<string, string> = {};
        const nextReadUsers: Record<string, string> = {};
        for (const item of readData.member_reads ?? []) {
          if (item.user_id && item.user_name) {
            nextReadUsers[item.user_id] = item.user_name;
          }
          if (item.user_id && item.last_read_message_id) {
            nextReadInfo[item.user_id] = item.last_read_message_id;
          }
        }
        setReadInfo(nextReadInfo);
        setReadUsers(nextReadUsers);
      } else {
        setReadStateError(toErrorMessage(readStateRes.status, '既読状態の取得に失敗しました。'));
      }
    } catch {
      setError('通信に失敗しました。接続を確認してください。');
    }
  };

  useEffect(() => {
    if (!id) return;
    const load = async () => {
      setError(null);
      await loadMessagesAndReadState();
    };
    load();
  }, [id, navigate]);

  useEffect(() => {
    if (!id || !latestMessageId) return;
    markAsRead(latestMessageId);
  }, [id, latestMessageId]);

  useEffect(() => {
    if (!id) return;
    const token = getToken();
    if (!token) {
      navigate('/login', { replace: true });
      return;
    }

    shouldReconnectRef.current = true;
    setRetryCount(0);

    const connect = (attempt: number) => {
      if (!shouldReconnectRef.current) return;
      setWsStatus('connecting');
      const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
      const ws = new WebSocket(
        `${protocol}://${window.location.host}/ws?access_token=${encodeURIComponent(token)}`,
      );
      wsRef.current = ws;

      ws.onopen = () => {
        setWsStatus('open');
        setRetryCount(0);
        ws.send(JSON.stringify({ type: 'subscribe', room_id: id, schema_version: 1 }));
      };

      ws.onmessage = (event) => {
        let payload: {
          type?: string;
          room_id?: string;
          message?: Message;
          user_id?: string;
          is_online?: boolean;
          last_read_message_id?: string;
          code?: string;
          message_text?: string;
        };
        try {
          payload = JSON.parse(event.data);
        } catch {
          return;
        }

        if (payload.type === 'session_ready' && payload.user_id) {
          setSelfUserId(payload.user_id);
          return;
        }

        if (payload.type === 'error') {
          if (payload.code === 'SCHEMA_VERSION_MISMATCH') {
            setError('WebSocket のスキーマ不一致です。');
            shouldReconnectRef.current = false;
          } else if (payload.code === 'FORBIDDEN_NOT_MEMBER') {
            setError('このルームの購読権限がありません。');
          } else {
            setError('WebSocket エラーが発生しました。');
          }
          return;
        }

        if (payload.type === 'message_created' && payload.room_id === id && payload.message) {
          appendMessageUnique(payload.message);
        }

        if (
          payload.type === 'read_updated' &&
          payload.room_id === id &&
          payload.user_id &&
          payload.last_read_message_id
        ) {
          setReadInfo((prev) => ({
            ...prev,
            [payload.user_id!]: payload.last_read_message_id!,
          }));
        }

        if (
          payload.type === 'presence_updated' &&
          payload.user_id &&
          typeof payload.is_online === 'boolean'
        ) {
          setMembersOnline((prev) => ({
            ...prev,
            [payload.user_id!]: payload.is_online!,
          }));
        }
      };

      ws.onclose = (event) => {
        setWsStatus('closed');
        if (!shouldReconnectRef.current) return;

        if (event.code === 1008) {
          setToken(null);
          navigate('/login', { replace: true });
          return;
        }

        const nextAttempt = attempt + 1;
        setRetryCount(nextAttempt);
        const wait = Math.min(1000 * 2 ** attempt, 10000);
        reconnectTimerRef.current = window.setTimeout(() => {
          connect(nextAttempt);
        }, wait);
      };

      ws.onerror = () => {
        setError('WebSocket 接続に失敗しました。再接続を試行します。');
      };
    };

    connect(0);

    return () => {
      shouldReconnectRef.current = false;
      if (reconnectTimerRef.current != null) {
        window.clearTimeout(reconnectTimerRef.current);
      }
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({ type: 'unsubscribe', room_id: id, schema_version: 1 }),
        );
      }
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [id, navigate]);

  const handleScroll = () => {
    if (!messageListRef.current || !latestMessageId) return;
    const el = messageListRef.current;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 16;
    if (atBottom) {
      markAsRead(latestMessageId);
    }
  };

  const handleSend = async () => {
    if (!id || !input.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${id}/messages`, {
        method: 'POST',
        body: JSON.stringify({ content: input }),
      });
      const data = await res.json().catch(() => ({}));
      if (res.ok && data.message) {
        appendMessageUnique(data.message);
        setInput('');
      } else if (!res.ok) {
        if (res.status === 401) {
          setToken(null);
          navigate('/login', { replace: true });
          return;
        }
        setError(toErrorMessage(res.status, 'メッセージ送信に失敗しました。'));
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSelectImage = async (file: File) => {
    if (!id) return;
    setUploading(true);
    setError(null);
    try {
      const token = getToken();
      const form = new FormData();
      form.append('file', file);
      const uploadRes = await fetch('/media/images', {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: form,
      });
      const uploadData = await uploadRes.json().catch(() => ({}));
      if (!uploadRes.ok || !uploadData.url) {
        if (uploadRes.status === 401) {
          setToken(null);
          navigate('/login', { replace: true });
          return;
        }
        setError(toErrorMessage(uploadRes.status, '画像アップロードに失敗しました。'));
        return;
      }
      const sendRes = await fetchWithAuth(`/rooms/${id}/messages`, {
        method: 'POST',
        body: JSON.stringify({ image_url: uploadData.url }),
      });
      const sendData = await sendRes.json().catch(() => ({}));
      if (sendRes.ok && sendData.message) {
        appendMessageUnique(sendData.message);
      } else {
        if (sendRes.status === 401) {
          setToken(null);
          navigate('/login', { replace: true });
          return;
        }
        setError(toErrorMessage(sendRes.status, '画像メッセージ送信に失敗しました。'));
      }
    } finally {
      setUploading(false);
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
      <div className="messages" ref={messageListRef} onScroll={handleScroll}>
        {messages.map((m) => (
          <div key={m.id} className="message-row">
            <div className="message-meta">
              <span className="sender">{m.sender_name ?? m.sender_id ?? 'unknown'}</span>
              <span className="time">
                {new Date(m.created_at).toLocaleTimeString()}
              </span>
            </div>
            {m.content && <div className="bubble">{m.content}</div>}
            {m.image_url && (
              <div className="bubble image">
                <img src={m.image_url} alt="attached" />
              </div>
            )}
            {m.id === latestMessageId && !!readersByMessage[m.id]?.length && (
              <div className="read-receipt">
                {readersByMessage[m.id].length === 1 ? (
                  <>
                    既読{' '}
                    <span className="readers-inline">
                      {readersByMessage[m.id][0].userName ??
                        readersByMessage[m.id][0].userId.slice(0, 8)}
                    </span>
                  </>
                ) : (
                  <>
                    既読 {readersByMessage[m.id].length}
                    <span className="readers-inline">
                      {' '}
                      (
                      {readersByMessage[m.id]
                        .slice(0, 3)
                        .map((r) => r.userName ?? r.userId.slice(0, 8))
                        .join(', ')}
                      {readersByMessage[m.id].length > 3 ? ', ...' : ''})
                    </span>
                  </>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
      <div className="chat-status">
        <span>Online: {Object.values(membersOnline).filter(Boolean).length}</span>
        <span>Read Updates: {Object.keys(readInfo).length}</span>
        <span>WS: {wsStatus}</span>
        {retryCount > 0 && <span>Reconnect: {retryCount}</span>}
      </div>
      {error && <div className="error-text">{error}</div>}
      {readStateError && (
        <div className="error-inline-row">
          <span className="error-text">{readStateError}</span>
          <button className="pill-button" type="button" onClick={loadMessagesAndReadState}>
            再取得
          </button>
        </div>
      )}
      <div className="chat-input-row">
        <label className="icon-button" aria-label="画像送信">
          画像
          <input
            type="file"
            accept="image/*"
            style={{ display: 'none' }}
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) {
                handleSelectImage(file);
              }
              e.currentTarget.value = '';
            }}
          />
        </label>
        <input
          className="chat-input"
          placeholder="メッセージを入力..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              handleSend();
            }
          }}
        />
        <button
          className="button"
          style={{ width: 'auto', paddingInline: 20, marginLeft: 8 }}
          onClick={handleSend}
          disabled={loading || uploading || !input.trim()}
        >
          {loading ? '送信中...' : uploading ? '画像送信中...' : '送信'}
        </button>
      </div>
    </div>
  );
}

export function ChatLayout() {
  const navigate = useNavigate();
  const [rooms, setRooms] = useState<RoomSummary[]>([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createType, setCreateType] = useState<'direct' | 'group'>('direct');
  const [targetUserId, setTargetUserId] = useState('');
  const [groupName, setGroupName] = useState('');
  const [memberIdsText, setMemberIdsText] = useState('');
  const [roomError, setRoomError] = useState<string | null>(null);
  const [creatingRoom, setCreatingRoom] = useState(false);

  useEffect(() => {
    if (!showCreateModal) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setShowCreateModal(false);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [showCreateModal]);

  const loadRooms = async () => {
    const res = await fetchWithAuth('/rooms');
    if (res.status === 401) {
      setToken(null);
      navigate('/login', { replace: true });
      return;
    }
    if (!res.ok) {
      setRoomError(toErrorMessage(res.status, 'ルーム一覧の取得に失敗しました。'));
      return;
    }
    const data = await res.json();
    setRooms(data.rooms ?? data.Rooms ?? []);
  };

  useEffect(() => {
    loadRooms();
  }, [navigate]);

  const handleCreateRoom = async (): Promise<string | null> => {
    setCreatingRoom(true);
    setRoomError(null);
    try {
      if (createType === 'direct') {
        if (!targetUserId.trim()) {
          setRoomError('target_user_id を入力してください。');
          return null;
        }
        const res = await fetchWithAuth('/rooms/direct', {
          method: 'POST',
          body: JSON.stringify({ target_user_id: targetUserId.trim() }),
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
          setRoomError(toErrorMessage(res.status, data.error ?? 'ダイレクトルーム作成に失敗しました。'));
          return null;
        }
        if (!data.id) {
          setRoomError('作成レスポンスに room id がありません。');
          return null;
        }
        const createdRoomId = String(data.id);
        setTargetUserId('');
        setGroupName('');
        setMemberIdsText('');
        await loadRooms();
        return createdRoomId;
      } else {
        const memberIds = memberIdsText
          .split(/[,\n]/)
          .map((v) => v.trim())
          .filter(Boolean);
        if (!groupName.trim()) {
          setRoomError('グループ名を入力してください。');
          return null;
        }
        if (memberIds.length === 0) {
          setRoomError('member_ids を1件以上入力してください。');
          return null;
        }
        const res = await fetchWithAuth('/rooms/groups', {
          method: 'POST',
          body: JSON.stringify({ name: groupName.trim(), member_ids: memberIds }),
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
          setRoomError(toErrorMessage(res.status, data.error ?? 'グループルーム作成に失敗しました。'));
          return null;
        }
        if (!data.id) {
          setRoomError('作成レスポンスに room id がありません。');
          return null;
        }
        const createdRoomId = String(data.id);
        setTargetUserId('');
        setGroupName('');
        setMemberIdsText('');
        await loadRooms();
        return createdRoomId;
      }
    } finally {
      setCreatingRoom(false);
    }
  };

  return (
    <div className="chat-layout">
      <div className="chat-shell">
        <aside className="rooms-list">
          <h2>ルーム</h2>
          <button className="button create-open-button" type="button" onClick={() => setShowCreateModal(true)}>
            ルーム作成
          </button>
          <div className="rooms-scroll">
            {rooms.map((r) => (
              <Link key={r.id} to={`/rooms/${r.id}`} className="room-item">
                <div className="room-name">{r.name ?? r.direct_peer_name ?? 'ダイレクトルーム'}</div>
                {r.last_message?.preview && (
                  <div className="room-preview">
                    {r.last_message.preview}
                  </div>
                )}
              </Link>
            ))}
          </div>
        </aside>
        <main className="chat-main">
          <Routes>
            <Route path=":id" element={<RoomMessages />} />
            <Route
              path="*"
              element={<div className="chat-empty">ルームを選択してください</div>}
            />
          </Routes>
        </main>
      </div>
      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <h3>ルーム作成</h3>
            <div className="switch-row">
              <button
                className={`pill-button ${createType === 'direct' ? 'active' : ''}`}
                onClick={() => setCreateType('direct')}
                type="button"
              >
                Direct
              </button>
              <button
                className={`pill-button ${createType === 'group' ? 'active' : ''}`}
                onClick={() => setCreateType('group')}
                type="button"
              >
                Group
              </button>
            </div>
            {createType === 'direct' ? (
              <input
                className="room-create-input"
                value={targetUserId}
                onChange={(e) => setTargetUserId(e.target.value)}
                placeholder="target_user_id"
              />
            ) : (
              <>
                <input
                  className="room-create-input"
                  value={groupName}
                  onChange={(e) => setGroupName(e.target.value)}
                  placeholder="group name"
                />
                <textarea
                  className="room-create-textarea"
                  value={memberIdsText}
                  onChange={(e) => setMemberIdsText(e.target.value)}
                  placeholder="member_ids (comma or newline)"
                />
              </>
            )}
            {roomError && <div className="error-text">{roomError}</div>}
            <div className="modal-actions">
              <button className="pill-button" type="button" onClick={() => setShowCreateModal(false)}>
                閉じる
              </button>
              <button
                className="button"
                type="button"
                onClick={async () => {
                  const createdRoomId = await handleCreateRoom();
                  if (createdRoomId) {
                    setShowCreateModal(false);
                    navigate(`/rooms/${createdRoomId}`);
                  }
                }}
                disabled={creatingRoom}
              >
                {creatingRoom ? '作成中...' : '作成'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

