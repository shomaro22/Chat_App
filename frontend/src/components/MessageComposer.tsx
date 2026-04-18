type Props = {
  input: string;
  onInputChange: (v: string) => void;
  onSend: () => void;
  onFileSelected: (file: File) => void;
  loading: boolean;
  uploading: boolean;
};

export function MessageComposer({
  input,
  onInputChange,
  onSend,
  onFileSelected,
  loading,
  uploading,
}: Props) {
  return (
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
              onFileSelected(file);
            }
            e.currentTarget.value = '';
          }}
        />
      </label>
      <input
        className="chat-input"
        placeholder="メッセージを入力..."
        value={input}
        onChange={(e) => onInputChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            onSend();
          }
        }}
      />
      <button
        className="button"
        style={{ width: 'auto', paddingInline: 20, marginLeft: 8 }}
        type="button"
        onClick={onSend}
        disabled={loading || uploading || !input.trim()}
      >
        {loading ? '送信中...' : uploading ? '画像送信中...' : '送信'}
      </button>
    </div>
  );
}
