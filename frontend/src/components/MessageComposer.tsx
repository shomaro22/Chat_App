import type { Message } from '../types/chat';

type Props = {
  input: string;
  onInputChange: (v: string) => void;
  onSend: () => void;
  onImageSelected: (file: File) => void;
  onGenericFileSelected: (file: File) => void;
  onTypingActivity?: () => void;
  loading: boolean;
  uploading: boolean;
  replyTo: Message | null;
  onClearReply: () => void;
};

export function MessageComposer({
  input,
  onInputChange,
  onSend,
  onImageSelected,
  onGenericFileSelected,
  onTypingActivity,
  loading,
  uploading,
  replyTo,
  onClearReply,
}: Props) {
  return (
    <div className="chat-input-block">
      {replyTo && (
        <div className="reply-draft-bar">
          <span className="muted">
            返信: {replyTo.sender_name ?? replyTo.sender_id?.slice(0, 8)} — {(replyTo.content ?? '').slice(0, 80)}
            {(replyTo.content?.length ?? 0) > 80 ? '…' : ''}
          </span>
          <button type="button" className="pill-button" onClick={onClearReply}>
            取消
          </button>
        </div>
      )}
      <div className="chat-input-row">
        <label className="icon-button" aria-label="画像を送信">
          画像
          <input
            type="file"
            accept="image/*"
            style={{ display: 'none' }}
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) {
                onImageSelected(file);
              }
              e.currentTarget.value = '';
            }}
          />
        </label>
        <label className="icon-button" aria-label="ファイルを送信（PDF / テキスト / ZIP）">
          ファイル
          <input
            type="file"
            accept=".pdf,.txt,.zip,application/pdf,text/plain,application/zip,application/x-zip-compressed"
            style={{ display: 'none' }}
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) {
                onGenericFileSelected(file);
              }
              e.currentTarget.value = '';
            }}
          />
        </label>
        <input
          className="chat-input"
          data-testid="chat-message-input"
          placeholder="メッセージを入力..."
          value={input}
          onChange={(e) => {
            onInputChange(e.target.value);
            onTypingActivity?.();
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              onSend();
            }
          }}
        />
        <button
          className="button"
          data-testid="chat-send-button"
          style={{ width: 'auto', paddingInline: 20, marginLeft: 8 }}
          type="button"
          onClick={onSend}
          disabled={loading || uploading || !input.trim()}
        >
          {loading ? '送信中...' : uploading ? '送信中...' : '送信'}
        </button>
      </div>
    </div>
  );
}
