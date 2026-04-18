import type { RefObject } from 'react';
import type { Message, ReaderInfo } from '../types/chat';

type Props = {
  messageListRef: RefObject<HTMLDivElement | null>;
  messages: Message[];
  hasMoreOlder: boolean;
  loadingOlder: boolean;
  onScroll: () => void;
  onLoadOlderClick: () => void;
  latestVisibleMessageId: string | null;
  readersByMessage: Record<string, ReaderInfo[]>;
  selfUserId: string | null;
  deletingId: string | null;
  onDeleteMessage: (id: string) => void;
};

export function MessageList({
  messageListRef,
  messages,
  hasMoreOlder,
  loadingOlder,
  onScroll,
  onLoadOlderClick,
  latestVisibleMessageId,
  readersByMessage,
  selfUserId,
  deletingId,
  onDeleteMessage,
}: Props) {
  return (
    <div className="messages" ref={messageListRef} onScroll={onScroll}>
      {hasMoreOlder && (
        <div className="messages-load-older">
          {loadingOlder ? (
            <span className="muted">過去のメッセージを読み込み中…</span>
          ) : (
            <>
              <span className="muted">上端付近で自動読み込み</span>
              <button type="button" className="pill-button" onClick={onLoadOlderClick}>
                さらに読み込む
              </button>
            </>
          )}
        </div>
      )}
      {messages.map((m) => (
        <div key={m.id} className={`message-row${m.is_deleted ? ' message-row-deleted' : ''}`}>
          <div className="message-meta">
            <span className="sender">{m.sender_name ?? m.sender_id ?? '不明'}</span>
            <span className="time">{new Date(m.created_at).toLocaleTimeString()}</span>
            {selfUserId &&
              m.sender_id === selfUserId &&
              !m.is_deleted &&
              (deletingId === m.id ? (
                <span className="message-delete-status">削除中…</span>
              ) : (
                <button
                  type="button"
                  className="message-delete-btn"
                  disabled={deletingId !== null}
                  onClick={() => void onDeleteMessage(m.id)}
                >
                  削除
                </button>
              ))}
          </div>
          {m.is_deleted ? (
            <div className="bubble bubble-deleted">このメッセージは削除されました</div>
          ) : (
            <>
              {m.content && <div className="bubble">{m.content}</div>}
              {m.image_url && (
                <div className="bubble image">
                  <img src={m.image_url} alt="添付画像" />
                </div>
              )}
            </>
          )}
          {m.id === latestVisibleMessageId && !!readersByMessage[m.id]?.length && (
            <div className="read-receipt">
              {readersByMessage[m.id].length === 1 ? (
                <>
                  既読{' '}
                  <span className="readers-inline">
                    {readersByMessage[m.id][0].userName ?? readersByMessage[m.id][0].userId.slice(0, 8)}
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
                    {readersByMessage[m.id].length > 3 ? ', …' : ''})
                  </span>
                </>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
