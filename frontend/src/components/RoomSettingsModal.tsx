type RoomMember = {
  user_id: string;
  name: string;
  online: boolean;
};

type RoomMeta = {
  type: 'direct' | 'group';
  is_frozen: boolean;
};

type DirectPeer = {
  user_id: string;
};

type RoomSettingsModalProps = {
  open: boolean;
  roomActionBusy: boolean;
  roomModalError: string | null;
  roomMeta: RoomMeta | null;
  directPeer: DirectPeer | null;
  peerBlocked: boolean;
  blockBusy: boolean;
  canRenameGroup: boolean;
  renameDraft: string;
  canAddMember: boolean;
  addMemberRef: string;
  roomMembers: RoomMember[];
  selfUserId: string | null;
  isGroup: boolean;
  isRoomCreator: boolean;
  canCloseRoom: boolean;
  onClose: () => void;
  onRenameDraftChange: (v: string) => void;
  onPatchRoomName: () => void;
  onAddMemberRefChange: (v: string) => void;
  onAddMember: () => void;
  onToggleDirectPeerBlock: () => void;
  onOpenLeaveConfirm: (userId: string) => void;
  onOpenRemoveConfirm: (userId: string) => void;
  onOpenCloseConfirm: () => void;
};

export function RoomSettingsModal({
  open,
  roomActionBusy,
  roomModalError,
  roomMeta,
  directPeer,
  peerBlocked,
  blockBusy,
  canRenameGroup,
  renameDraft,
  canAddMember,
  addMemberRef,
  roomMembers,
  selfUserId,
  isGroup,
  isRoomCreator,
  canCloseRoom,
  onClose,
  onRenameDraftChange,
  onPatchRoomName,
  onAddMemberRefChange,
  onAddMember,
  onToggleDirectPeerBlock,
  onOpenLeaveConfirm,
  onOpenRemoveConfirm,
  onOpenCloseConfirm,
}: RoomSettingsModalProps) {
  if (!open) return null;

  return (
    <div className="modal-overlay" onClick={() => (!roomActionBusy ? onClose() : null)}>
      <div className="modal-card modal-card-room" onClick={(e) => e.stopPropagation()}>
        <h3>ルーム設定</h3>
        {roomModalError && <div className="error-text">{roomModalError}</div>}
        {roomMeta?.is_frozen && (
          <p className="muted">このルームは凍結中です。名前変更・メンバー追加・送信はサーバー側の制限に従います。</p>
        )}
        {roomMeta?.type === 'direct' && directPeer && (
          <div className="room-settings-block">
            <label className="room-settings-label">ユーザー制御</label>
            <p className="muted">
              {peerBlocked
                ? 'このユーザーをブロック中です。相手の新着メッセージは表示・通知されません。'
                : 'ブロックすると相手の新着メッセージは表示・通知されません。'}
            </p>
            <button
              type="button"
              className={`pill-button ${peerBlocked ? '' : 'danger-outline'}`}
              disabled={roomActionBusy || blockBusy}
              onClick={onToggleDirectPeerBlock}
            >
              {blockBusy ? '処理中…' : peerBlocked ? 'ブロック解除' : 'このユーザーをブロック'}
            </button>
          </div>
        )}
        {canRenameGroup && (
          <div className="room-settings-block">
            <label className="room-settings-label">グループ名</label>
            <input
              className="room-create-input"
              value={renameDraft}
              onChange={(e) => onRenameDraftChange(e.target.value)}
              disabled={roomActionBusy}
            />
            <button
              type="button"
              className="pill-button"
              disabled={roomActionBusy || !renameDraft.trim()}
              onClick={onPatchRoomName}
            >
              名前を更新
            </button>
          </div>
        )}
        {canAddMember && (
          <div className="room-settings-block">
            <label className="room-settings-label">メンバー追加（メールまたはユーザー ID）</label>
            <input
              className="room-create-input"
              value={addMemberRef}
              onChange={(e) => onAddMemberRefChange(e.target.value)}
              placeholder="メールまたはユーザーID（API: member_id）"
              disabled={roomActionBusy}
            />
            <button
              type="button"
              className="pill-button"
              disabled={roomActionBusy || !addMemberRef.trim()}
              onClick={onAddMember}
            >
              追加
            </button>
          </div>
        )}
        <div className="room-settings-block">
          <div className="room-settings-label">メンバー</div>
          <ul className="room-member-actions">
            {roomMembers.map((m) => {
              const isSelf = selfUserId === m.user_id;
              const canKick = isGroup && isRoomCreator && !isSelf && !roomMeta?.is_frozen;
              return (
                <li key={m.user_id} className="room-member-actions-row">
                  <span>
                    {m.name || m.user_id.slice(0, 8)}
                    {m.online ? ' ●' : ''}
                  </span>
                  <span className="room-member-actions-btns">
                    {isSelf && isGroup && (
                      <button
                        type="button"
                        className="pill-button danger-outline"
                        disabled={roomActionBusy}
                        onClick={() => onOpenLeaveConfirm(m.user_id)}
                      >
                        ルームから退出
                      </button>
                    )}
                    {canKick && (
                      <button
                        type="button"
                        className="pill-button danger-outline"
                        disabled={roomActionBusy}
                        onClick={() => onOpenRemoveConfirm(m.user_id)}
                      >
                        メンバーを削除
                      </button>
                    )}
                  </span>
                </li>
              );
            })}
          </ul>
        </div>
        {canCloseRoom && (
          <div className="room-settings-block">
            <button
              type="button"
              className="pill-button danger-outline room-close-button"
              disabled={roomActionBusy}
              onClick={onOpenCloseConfirm}
            >
              ルームを閉じる
            </button>
          </div>
        )}
        <div className="modal-actions">
          <button type="button" className="pill-button" disabled={roomActionBusy} onClick={onClose}>
            閉じる
          </button>
        </div>
      </div>
    </div>
  );
}
