type CreateKind = 'direct' | 'group';

type Props = {
  open: boolean;
  onClose: () => void;
  createKind: CreateKind;
  onCreateKindChange: (k: CreateKind) => void;
  /** 1対1: API の target_user_id に渡す値（メールまたはユーザーID） */
  directPeerRef: string;
  onDirectPeerRefChange: (v: string) => void;
  groupDisplayName: string;
  onGroupDisplayNameChange: (v: string) => void;
  /** グループ: API の member_ids に渡す値（メールまたはユーザーIDをカンマ・改行区切り） */
  groupMemberRefsText: string;
  onGroupMemberRefsTextChange: (v: string) => void;
  roomError: string | null;
  creatingRoom: boolean;
  onSubmit: () => void;
  submitDisabled: boolean;
};

export function RoomCreateModal({
  open,
  onClose,
  createKind,
  onCreateKindChange,
  directPeerRef,
  onDirectPeerRefChange,
  groupDisplayName,
  onGroupDisplayNameChange,
  groupMemberRefsText,
  onGroupMemberRefsTextChange,
  roomError,
  creatingRoom,
  onSubmit,
  submitDisabled,
}: Props) {
  if (!open) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <h3>ルーム作成</h3>
        <div className="switch-row">
          <button
            className={`pill-button ${createKind === 'direct' ? 'active' : ''}`}
            onClick={() => onCreateKindChange('direct')}
            type="button"
          >
            1対1
          </button>
          <button
            className={`pill-button ${createKind === 'group' ? 'active' : ''}`}
            onClick={() => onCreateKindChange('group')}
            type="button"
          >
            グループ
          </button>
        </div>
        {createKind === 'direct' ? (
          <input
            className="room-create-input"
            data-testid="room-modal-direct-peer"
            value={directPeerRef}
            onChange={(e) => onDirectPeerRefChange(e.target.value)}
            placeholder="相手のメールアドレスまたはユーザーID"
          />
        ) : (
          <>
            <input
              className="room-create-input"
              value={groupDisplayName}
              onChange={(e) => onGroupDisplayNameChange(e.target.value)}
              placeholder="グループの表示名"
            />
            <textarea
              className="room-create-textarea"
              value={groupMemberRefsText}
              onChange={(e) => onGroupMemberRefsTextChange(e.target.value)}
              placeholder="参加メンバー（メールまたはユーザーIDをカンマ・改行区切り）"
            />
          </>
        )}
        {roomError && <div className="error-text">{roomError}</div>}
        <div className="modal-actions">
          <button className="pill-button" type="button" onClick={onClose}>
            閉じる
          </button>
          <button className="button" type="button" data-testid="room-modal-submit" onClick={onSubmit} disabled={submitDisabled}>
            {creatingRoom ? '作成中...' : '作成'}
          </button>
        </div>
      </div>
    </div>
  );
}
