import { Link } from 'react-router-dom';
import type { MeUser, RoomSummary } from '../types/chat';

type Props = {
  rooms: RoomSummary[];
  me: MeUser | null;
  onOpenCreateModal: () => void;
};

export function RoomList({ rooms, me, onOpenCreateModal }: Props) {
  return (
    <aside className="rooms-list">
      <h2>ルーム</h2>
      {me && (
        <div className="account-self">
          <div className="account-self-title">あなたのアカウント</div>
          <div className="account-self-row">
            <span className="account-self-k">メール</span>
            <span className="account-self-v" title={me.email}>
              {me.email}
            </span>
          </div>
          <div className="account-self-row">
            <span className="account-self-k">ユーザーID</span>
            <span className="account-self-v account-self-id" title={me.id}>
              {me.id}
            </span>
          </div>
          {me.role === 'admin' && (
            <div className="account-admin-link">
              <Link to="/admin">管理者画面</Link>
            </div>
          )}
        </div>
      )}
      <button className="button create-open-button" type="button" onClick={onOpenCreateModal}>
        ルーム作成
      </button>
      <div className="rooms-scroll">
        {rooms.map((r) => (
          <Link key={r.id} to={`/rooms/${r.id}`} className="room-item">
            <div className="room-name">{r.name ?? r.direct_peer_name ?? 'ダイレクトルーム'}</div>
            {r.last_message?.preview && <div className="room-preview">{r.last_message.preview}</div>}
          </Link>
        ))}
      </div>
    </aside>
  );
}
