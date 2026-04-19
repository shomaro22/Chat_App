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
          <div className="account-admin-link">
            <Link to="/settings">設定</Link>
            {' · '}
            <Link to="/search">検索</Link>
            {' · '}
            <Link to="/bookmarks">ブックマーク</Link>
            {me.role === 'admin' && (
              <>
                {' · '}
                <Link to="/admin">管理者</Link>
              </>
            )}
          </div>
        </div>
      )}
      <button className="button create-open-button" type="button" data-testid="sidebar-create-room" onClick={onOpenCreateModal}>
        ルーム作成
      </button>
      <div className="rooms-scroll" data-testid="sidebar-room-list">
        {rooms.map((r) => (
          <Link key={r.id} to={`/rooms/${r.id}`} className="room-item" data-testid={`sidebar-room-link-${r.id}`}>
            <div className="room-name">{r.name ?? r.direct_peer_name ?? 'ダイレクトルーム'}</div>
            {r.last_message?.preview && <div className="room-preview">{r.last_message.preview}</div>}
          </Link>
        ))}
      </div>
    </aside>
  );
}
