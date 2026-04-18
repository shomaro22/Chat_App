import { useCallback, useState } from 'react';
import { fetchWithAuth } from '../lib/api';
import { toErrorMessage } from '../chat/errors';
import type { ReadStateResponse } from '../types/chat';

type Report = (message: string | null) => void;

export function useReadState(roomId: string | undefined, reportError: Report) {
  const [readInfo, setReadInfo] = useState<Record<string, string>>({});
  const [readUsers, setReadUsers] = useState<Record<string, string>>({});
  const [readStateError, setReadStateError] = useState<string | null>(null);
  const [readStateUnreadCount, setReadStateUnreadCount] = useState<number | null>(null);
  const [readStateMyLastReadId, setReadStateMyLastReadId] = useState<string | null>(null);

  const applyReadPayload = useCallback((readData: ReadStateResponse) => {
    const ur = readData.unread_count;
    if (typeof ur === 'number' && !Number.isNaN(ur)) {
      setReadStateUnreadCount(ur);
    } else if (ur != null && String(ur).trim() !== '') {
      const n = Number(ur);
      setReadStateUnreadCount(Number.isNaN(n) ? null : n);
    }
    const ml = readData.my_last_read_message_id;
    setReadStateMyLastReadId(ml != null && String(ml).trim() ? String(ml) : null);

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
  }, []);

  const loadReadState = useCallback(async () => {
    if (!roomId) return false;
    const readStateRes = await fetchWithAuth(`/rooms/${roomId}/read-state`);
    if (!readStateRes.ok) {
      if (readStateRes.status === 401) return false;
      setReadStateError(toErrorMessage(readStateRes.status, '既読状態の取得に失敗しました。'));
      return false;
    }
    const readData = (await readStateRes.json()) as ReadStateResponse;
    applyReadPayload(readData);
    setReadStateError(null);
    return true;
  }, [roomId, applyReadPayload]);

  const markAsRead = useCallback(
    async (messageId: string) => {
      if (!roomId || !messageId) return;
      const res = await fetchWithAuth(`/rooms/${roomId}/read`, {
        method: 'POST',
        body: JSON.stringify({ message_id: messageId }),
      });
      if (!res.ok && res.status !== 409) {
        if (res.status === 401) return;
        reportError(toErrorMessage(res.status, '既読更新に失敗しました。'));
      }
    },
    [roomId, reportError],
  );

  return {
    readInfo,
    setReadInfo,
    readUsers,
    setReadUsers,
    readStateError,
    setReadStateError,
    readStateUnreadCount,
    readStateMyLastReadId,
    loadReadState,
    markAsRead,
    applyReadPayload,
  };
}
