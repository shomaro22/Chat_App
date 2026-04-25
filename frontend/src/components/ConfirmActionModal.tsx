type ConfirmAction =
  | null
  | { type: 'close-room' }
  | { type: 'leave-room'; userId: string }
  | { type: 'remove-member'; userId: string };

type ConfirmActionModalProps = {
  confirmAction: ConfirmAction;
  roomActionBusy: boolean;
  onClose: () => void;
  onConfirm: () => void;
};

export function ConfirmActionModal({
  confirmAction,
  roomActionBusy,
  onClose,
  onConfirm,
}: ConfirmActionModalProps) {
  if (!confirmAction) return null;

  const confirmTitle =
    confirmAction.type === 'close-room'
      ? 'ルームを閉じる'
      : confirmAction.type === 'leave-room'
        ? 'ルームから退出'
        : 'メンバーを削除';

  const confirmMessage =
    confirmAction.type === 'close-room'
      ? 'このルームを閉じますか？（メンバー全員がアクセスできなくなります）'
      : confirmAction.type === 'leave-room'
        ? 'このルームから退出しますか？'
        : 'このメンバーを削除しますか？';

  return (
    <div className="modal-overlay" onClick={() => (!roomActionBusy ? onClose() : null)}>
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <h3>{confirmTitle}</h3>
        <p>{confirmMessage}</p>
        <div className="modal-actions">
          <button
            type="button"
            className="pill-button"
            disabled={roomActionBusy}
            onClick={onClose}
          >
            キャンセル
          </button>
          <button
            type="button"
            className="pill-button danger-outline"
            disabled={roomActionBusy}
            onClick={onConfirm}
          >
            実行
          </button>
        </div>
      </div>
    </div>
  );
}
