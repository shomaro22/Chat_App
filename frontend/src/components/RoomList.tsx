import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { setToken } from '../lib/api';
import type { MeUser, RoomSummary } from '../types/chat';

type Props = {
  rooms: RoomSummary[];
  me: MeUser | null;
  onOpenCreateModal: () => void;
};

function toDisplayId(v?: string | null): string {
  const s = (v ?? '').trim();
  if (!s) return '—';
  return s.length > 12 ? `${s.slice(0, 8)}…` : s;
}

export function RoomList({ rooms, me, onOpenCreateModal }: Props) {
  const navigate = useNavigate();
  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false);

  const handleLogout = () => {
    setToken(null);
    navigate('/login', { replace: true });
  };

  return (
    <>
      <aside className="rooms-list">
        <h2>ルーム</h2>
        {me && (
          <div className="account-self">
          <div className="account-self-title">{me.name?.trim() ? `${me.name} さん` : 'アカウント情報'}</div>
          <div className="account-self-row">
            <span className="account-self-k">ユーザー名</span>
            <span className="account-self-v">
              {me.name?.trim() || '—'}
            </span>
          </div>
            <div className="account-self-row">
              <span className="account-self-k">メール</span>
              <span className="account-self-v" title={me.email}>
                {me.email}
              </span>
            </div>
            <div className="account-self-row">
              <span className="account-self-k">ユーザーID</span>
              <span className="account-self-v account-self-id">
                {toDisplayId(me.id)}
              </span>
            </div>
            <div className="account-admin-link">
              <Link to="/settings">設定</Link>
              {' · '}
              <Link to="/search">検索</Link>
              {' · '}
              <Link to="/bookmarks">ブックマーク</Link>
              {' · '}
              <button type="button" className="account-inline-link-button" onClick={() => setShowLogoutConfirm(true)}>
                ログアウト
              </button>
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
              <div className="room-item-top">
                <div className="room-name">{r.name ?? r.direct_peer_name ?? 'ダイレクトルーム'}</div>
                <span className={`room-kind-badge ${r.direct_peer_name?.trim() ? 'is-direct' : 'is-group'}`}>
                  {r.direct_peer_name?.trim() ? '1対1' : 'グループ'}
                </span>
              </div>
              {r.last_message?.preview && <div className="room-preview">{r.last_message.preview}</div>}
            </Link>
          ))}
        </div>
      </aside>
      {showLogoutConfirm && (
        <div className="modal-overlay" onClick={() => setShowLogoutConfirm(false)}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <h3>ログアウト</h3>
            <p>ログアウトしますか？</p>
            <div className="modal-actions">
              <button type="button" className="pill-button" onClick={() => setShowLogoutConfirm(false)}>
                キャンセル
              </button>
              <button
                type="button"
                className="pill-button danger-outline"
                onClick={() => {
                  setShowLogoutConfirm(false);
                  handleLogout();
                }}
              >
                ログアウト
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
