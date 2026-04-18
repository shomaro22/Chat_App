import { useEffect, useRef, type MutableRefObject } from 'react';
import { buildWsUrl } from '../lib/ws';
import { getToken, setToken } from '../lib/token';
import type { Message } from '../types/chat';

export type RoomWsHandlerBag = {
  onSessionReady: (userId: string) => void;
  onReloadData: () => void;
  onMessageCreated: (message: Message) => void;
  onMessageDeleted: (messageId: string) => void;
  onReadUpdated: (userId: string, lastReadMessageId: string) => void;
  onPresenceUpdated: (userId: string, isOnline: boolean) => void;
  onError: (message: string) => void;
  onSchemaMismatch: () => void;
  onForbiddenMember: () => void;
  onWsTransportError: () => void;
  onConnecting: () => void;
  onOpen: () => void;
  onClosed: () => void;
  onReconnectScheduled: (nextAttempt: number) => void;
};

/**
 * roomId ごとに WS を張り、ハンドラは ref 経由で常に最新を参照する（effect の依存を抑える）。
 */
export function useRoomWebSocket(
  roomId: string | undefined,
  navigateToLogin: () => void,
  handlersRef: MutableRefObject<RoomWsHandlerBag>,
) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const shouldReconnectRef = useRef(true);

  useEffect(() => {
    if (!roomId) return;
    const token = getToken();
    if (!token) {
      navigateToLogin();
      return;
    }

    shouldReconnectRef.current = true;

    const connect = (attempt: number) => {
      if (!shouldReconnectRef.current || !roomId) return;
      handlersRef.current.onConnecting();
      const ws = new WebSocket(buildWsUrl(token));
      wsRef.current = ws;

      ws.onopen = () => {
        handlersRef.current.onOpen();
        ws.send(JSON.stringify({ type: 'subscribe', room_id: roomId, schema_version: 1 }));
        // 初回接続・再接続のいずれでも HTTP でメッセージ / 既読 / ルーム詳細を再取得し、切断中の取りこぼしを回復する
        handlersRef.current.onReloadData();
      };

      ws.onmessage = (event) => {
        let payload: {
          type?: string;
          room_id?: string;
          message?: Message;
          message_id?: string;
          user_id?: string;
          is_online?: boolean;
          last_read_message_id?: string;
          code?: string;
        };
        try {
          payload = JSON.parse(event.data);
        } catch {
          return;
        }

        if (payload.type === 'session_ready' && payload.user_id) {
          handlersRef.current.onSessionReady(payload.user_id);
          return;
        }

        if (payload.type === 'error') {
          if (payload.code === 'SCHEMA_VERSION_MISMATCH') {
            handlersRef.current.onSchemaMismatch();
            shouldReconnectRef.current = false;
          } else if (payload.code === 'FORBIDDEN_NOT_MEMBER') {
            handlersRef.current.onForbiddenMember();
          } else {
            handlersRef.current.onError('WebSocket エラーが発生しました。');
          }
          return;
        }

        if (payload.type === 'message_created' && payload.room_id === roomId && payload.message) {
          handlersRef.current.onMessageCreated(payload.message);
        }

        if (payload.type === 'message_deleted' && payload.room_id === roomId && payload.message_id) {
          handlersRef.current.onMessageDeleted(payload.message_id);
        }

        if (
          payload.type === 'read_updated' &&
          payload.room_id === roomId &&
          payload.user_id &&
          payload.last_read_message_id
        ) {
          handlersRef.current.onReadUpdated(payload.user_id, payload.last_read_message_id);
        }

        if (payload.type === 'presence_updated' && payload.user_id && typeof payload.is_online === 'boolean') {
          handlersRef.current.onPresenceUpdated(payload.user_id, payload.is_online);
        }
      };

      ws.onclose = (event) => {
        handlersRef.current.onClosed();
        if (!shouldReconnectRef.current) return;

        if (event.code === 1008) {
          setToken(null);
          navigateToLogin();
          return;
        }

        const nextAttempt = attempt + 1;
        handlersRef.current.onReconnectScheduled(nextAttempt);
        const wait = Math.min(1000 * 2 ** attempt, 10000);
        reconnectTimerRef.current = window.setTimeout(() => {
          connect(nextAttempt);
        }, wait);
      };

      ws.onerror = () => {
        handlersRef.current.onWsTransportError();
      };
    };

    connect(0);

    return () => {
      shouldReconnectRef.current = false;
      if (reconnectTimerRef.current != null) {
        window.clearTimeout(reconnectTimerRef.current);
      }
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN && roomId) {
        wsRef.current.send(JSON.stringify({ type: 'unsubscribe', room_id: roomId, schema_version: 1 }));
      }
      wsRef.current?.close();
      wsRef.current = null;
    };
    // handlersRef は毎レンダーで .current を差し替えるため依存に含めない
    // eslint-disable-next-line react-hooks/exhaustive-deps -- roomId / ログイン遷移のみで再接続する
  }, [roomId, navigateToLogin]);
}
