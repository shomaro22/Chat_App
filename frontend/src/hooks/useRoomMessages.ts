import { useCallback, useState } from 'react';
import { fetchWithAuth } from '../lib/api';
import { appendMessagesUniqueSorted, mergeMessageSorted, sortMessagesAsc } from '../chat/messageUtils';
import { toErrorMessage } from '../chat/errors';
import { MESSAGE_PAGE_LIMIT, type Message } from '../types/chat';

type Report = (message: string | null) => void;

export function useRoomMessages(roomId: string | undefined, reportError: Report) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [hasMoreOlder, setHasMoreOlder] = useState(true);
  const [loadingOlder, setLoadingOlder] = useState(false);

  const resetForRoom = useCallback(() => {
    setMessages([]);
    setHasMoreOlder(true);
  }, []);

  const loadOlderMessages = useCallback(async (): Promise<boolean> => {
    if (!roomId || loadingOlder || !hasMoreOlder) return false;
    const oldestId = messages[0]?.id;
    if (!oldestId) return false;

    setLoadingOlder(true);
    reportError(null);
    try {
      const res = await fetchWithAuth(
        `/rooms/${roomId}/messages?order=desc&limit=${MESSAGE_PAGE_LIMIT}&before_id=${encodeURIComponent(oldestId)}`,
      );
      if (!res.ok) {
        if (res.status === 401) return false;
        reportError(toErrorMessage(res.status, '過去のメッセージの取得に失敗しました。'));
        return false;
      }
      const messageData = await res.json().catch(() => ({}));
      const chunk: Message[] = messageData.messages ?? messageData.Messages ?? [];
      setHasMoreOlder(chunk.length === MESSAGE_PAGE_LIMIT);
      const chunkAsc = sortMessagesAsc([...chunk].reverse());
      setMessages((prev) => appendMessagesUniqueSorted(prev, chunkAsc));
      return true;
    } catch {
      reportError('通信に失敗しました。');
      return false;
    } finally {
      setLoadingOlder(false);
    }
  }, [roomId, messages, hasMoreOlder, loadingOlder, reportError]);

  const upsertMessage = useCallback((nextMessage: Message) => {
    setMessages((prev) => mergeMessageSorted(prev, nextMessage));
  }, []);

  return {
    messages,
    setMessages,
    hasMoreOlder,
    setHasMoreOlder,
    loadingOlder,
    resetForRoom,
    loadOlderMessages,
    upsertMessage,
  };
}
