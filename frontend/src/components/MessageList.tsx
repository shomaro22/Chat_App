import { useState, type RefObject } from 'react';
import { resolvePublicAssetUrl } from '../lib/api';
import type { Message, ReaderInfo, ReactionChip } from '../types/chat';

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
  onEditMessage?: (id: string, content: string) => void;
  reactionsByMessage?: Record<string, ReactionChip[]>;
  onToggleReaction?: (messageId: string, emoji: string) => void;
  onReply?: (m: Message) => void;
  onPinMessage?: (messageId: string) => void;
  onClearPinMessage?: () => void;
  pinnedMessageId?: string | null;
  bookmarkedByMessage?: Record<string, boolean>;
  onToggleBookmark?: (messageId: string) => void;
  /** グループルームで API の read_count をメッセージごとに表示 */
  showGroupReadCounts?: boolean;
};

const QUICK_EMOJIS = ['👍', '❤️', '😂'];

function toDisplayId(v?: string | null): string {
  const s = (v ?? '').trim();
  if (!s) return '不明';
  return s.length > 12 ? `${s.slice(0, 8)}…` : s;
}

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
  onEditMessage,
  reactionsByMessage,
  onToggleReaction,
  onReply,
  onPinMessage,
  onClearPinMessage,
  pinnedMessageId,
  bookmarkedByMessage,
  onToggleBookmark,
  showGroupReadCounts,
}: Props) {
  const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
  const [editingDraft, setEditingDraft] = useState('');

  const byId: Record<string, Message> = {};
  messages.forEach((m) => {
    byId[m.id] = m;
  });

  return (
    <div className="messages" ref={messageListRef} onScroll={onScroll} data-testid="message-list">
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
      {messages.map((m) => {
        const isSelfMessage = Boolean(selfUserId && m.sender_id === selfUserId);
        return (
        <div
          key={m.id}
          className={`message-row${isSelfMessage ? ' message-row-self' : ' message-row-other'}${m.is_deleted ? ' message-row-deleted' : ''}${m.id === pinnedMessageId ? ' message-row-pinned' : ''}`}
          data-message-id={m.id}
        >
          <div className="message-meta">
            <span className="sender">{m.sender_name?.trim() || toDisplayId(m.sender_id)}</span>
            <span className="time">{new Date(m.created_at).toLocaleTimeString()}</span>
            {m.id === pinnedMessageId && <span className="message-pin-badge">ピン留め中</span>}
            {showGroupReadCounts && !m.is_deleted && m.read_count != null && (
              <span className="message-read-count-badge" title="このメッセージまで既読のメンバー数（送信者除く集計はサーバー定義）">
                既読 {m.read_count} 人
              </span>
            )}
            {isSelfMessage &&
              !m.is_deleted &&
              (deletingId === m.id ? (
                <span className="message-delete-status">削除中…</span>
              ) : (
                <>
                  {onEditMessage && (
                    <button
                      type="button"
                      className="message-delete-btn"
                      disabled={deletingId !== null}
                      onClick={() => {
                        setEditingMessageId(m.id);
                        setEditingDraft(m.content ?? '');
                      }}
                    >
                      編集
                    </button>
                  )}
                  <button
                    type="button"
                    className="message-delete-btn"
                    disabled={deletingId !== null}
                    onClick={() => void onDeleteMessage(m.id)}
                  >
                    削除
                  </button>
                </>
              ))}
          </div>
          {m.is_deleted ? (
            <div className="bubble bubble-deleted">このメッセージは削除されました</div>
          ) : (
            <>
              {m.parent_message_id && (
                <div className="muted" style={{ fontSize: 12 }}>
                  返信先:{' '}
                  {byId[m.parent_message_id]?.content?.slice(0, 60) ??
                    `${m.parent_message_id.slice(0, 8)}…`}
                  {(byId[m.parent_message_id]?.content?.length ?? 0) > 60 ? '…' : ''}
                </div>
              )}
              {editingMessageId === m.id ? (
                <div className="message-edit-inline">
                  <input
                    className="message-edit-input"
                    value={editingDraft}
                    onChange={(e) => setEditingDraft(e.target.value)}
                    autoFocus
                    onKeyDown={(e) => {
                      if (e.nativeEvent.isComposing || e.keyCode === 229) {
                        return;
                      }
                      if (e.key === 'Escape') {
                        setEditingMessageId(null);
                        setEditingDraft('');
                        return;
                      }
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault();
                        const next = editingDraft.trim();
                        if (!next || !onEditMessage) return;
                        void onEditMessage(m.id, next);
                        setEditingMessageId(null);
                        setEditingDraft('');
                      }
                    }}
                  />
                  <div className="message-edit-actions">
                    <button
                      type="button"
                      className="pill-button"
                      onClick={() => {
                        setEditingMessageId(null);
                        setEditingDraft('');
                      }}
                    >
                      キャンセル
                    </button>
                    <button
                      type="button"
                      className="pill-button reply-action-button"
                      disabled={!editingDraft.trim() || !onEditMessage}
                      onClick={() => {
                        const next = editingDraft.trim();
                        if (!next || !onEditMessage) return;
                        void onEditMessage(m.id, next);
                        setEditingMessageId(null);
                        setEditingDraft('');
                      }}
                    >
                      保存
                    </button>
                  </div>
                </div>
              ) : null}
              {m.content && <div className="bubble">{m.content}</div>}
              {m.image_url && (
                <div className="bubble image">
                  <img src={resolvePublicAssetUrl(m.image_url)} alt="添付画像" />
                </div>
              )}
              {m.file_url && (
                <div className="bubble">
                  <a href={resolvePublicAssetUrl(m.file_url)} target="_blank" rel="noreferrer">
                    {m.file_name ?? '添付ファイル'}
                  </a>
                </div>
              )}
              {reactionsByMessage?.[m.id] && reactionsByMessage[m.id].length > 0 && (
                <div className="message-reactions">
                  {reactionsByMessage[m.id].map((c) => (
                    (() => {
                      const names = c.userNames ?? [];
                      const firstTwo = names.slice(0, 2).join(', ');
                      const remain = names.length - 2;
                      const whoText = names.length === 0 ? '' : remain > 0 ? `${firstTwo} +${remain}` : firstTwo;
                      const title = names.length
                        ? `${c.emoji}: ${names.join(', ')}`
                        : c.mine
                          ? 'クリックで解除'
                          : 'クリックで追加 / 自分の分は解除';
                      return (
                        <button
                          key={c.emoji}
                          type="button"
                          className={`reaction-chip${c.mine ? ' mine' : ''}`}
                          disabled={!onToggleReaction}
                          onClick={() => onToggleReaction?.(m.id, c.emoji)}
                          title={title}
                        >
                          {c.emoji} {c.count}
                          {whoText ? ` (${whoText})` : ''}
                        </button>
                      );
                    })()
                  ))}
                </div>
              )}
              <div className="message-row-actions">
                {onReply && (
                  <button type="button" className="pill-button reply-action-button" onClick={() => onReply(m)}>
                    返信
                  </button>
                )}
                {onPinMessage && (
                  <button
                    type="button"
                    className={`pill-button pin-action-button${m.id === pinnedMessageId ? ' is-pinned' : ''}`}
                    onClick={() => {
                      if (m.id === pinnedMessageId) {
                        onClearPinMessage?.();
                        return;
                      }
                      onPinMessage(m.id);
                    }}
                  >
                    {m.id === pinnedMessageId ? 'ピン留め中' : 'ピン'}
                  </button>
                )}
                {onToggleBookmark && (
                  <button
                    type="button"
                    className={`pill-button bookmark-action-button${bookmarkedByMessage?.[m.id] ? ' is-bookmarked' : ''}`}
                    onClick={() => onToggleBookmark(m.id)}
                  >
                    {bookmarkedByMessage?.[m.id] ? 'ブックマーク解除' : 'ブックマーク'}
                  </button>
                )}
                {onToggleReaction &&
                  QUICK_EMOJIS.map((em) => (
                    <button
                      key={em}
                      type="button"
                      className="pill-button emoji-action-button"
                      onClick={() => onToggleReaction(m.id, em)}
                    >
                      {em}
                    </button>
                  ))}
              </div>
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
        );
      })}
    </div>
  );
}
