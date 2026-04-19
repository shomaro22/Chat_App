import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { fetchWithAuth, getToken } from '../lib/api';

type MeUser = {
  id: string;
  email: string;
  name: string;
  role: string;
};

type AdminUserRow = {
  id: string;
  name: string;
  email: string;
  role: string;
  active: boolean;
  created_at: string;
};

type AdminRoomRow = {
  id: string;
  type: string;
  name?: string | null;
  is_frozen: boolean;
  updated_at: string;
};

function toErrorMessage(status: number, fallback: string): string {
  if (status === 401) return '認証が切れました。再ログインしてください。';
  if (status === 403) return '管理者権限がありません。';
  if (status === 404) return '対象が見つかりません。';
  return fallback;
}

type Tab = 'users' | 'rooms';

export function AdminPage() {
  const navigate = useNavigate();
  const [me, setMe] = useState<MeUser | null>(null);
  const [meError, setMeError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>('users');

  const [userSearch, setUserSearch] = useState('');
  const [userQuery, setUserQuery] = useState('');
  const [users, setUsers] = useState<AdminUserRow[]>([]);
  const [userTotal, setUserTotal] = useState(0);
  const [userOffset, setUserOffset] = useState(0);
  const [userLoading, setUserLoading] = useState(false);
  const [userMsg, setUserMsg] = useState<string | null>(null);
  const [selectedUserIds, setSelectedUserIds] = useState<Record<string, boolean>>({});
  const [rowBusy, setRowBusy] = useState<string | null>(null);
  const [bulkBusy, setBulkBusy] = useState(false);

  const [roomSearch, setRoomSearch] = useState('');
  const [roomQuery, setRoomQuery] = useState('');
  const [rooms, setRooms] = useState<AdminRoomRow[]>([]);
  const [roomTotal, setRoomTotal] = useState(0);
  const [roomOffset, setRoomOffset] = useState(0);
  const [roomLoading, setRoomLoading] = useState(false);
  const [roomMsg, setRoomMsg] = useState<string | null>(null);
  const [selectedRoomId, setSelectedRoomId] = useState<string | null>(null);
  const [kickUserId, setKickUserId] = useState('');
  const [roomOpBusy, setRoomOpBusy] = useState(false);

  const loadMe = useCallback(async () => {
    setMeError(null);
    const res = await fetchWithAuth('/me');
    if (res.status === 401) {
      return;
    }
    if (!res.ok) {
      setMeError(toErrorMessage(res.status, 'ユーザー情報の取得に失敗しました。'));
      return;
    }
    const data = await res.json().catch(() => ({}));
    const u = data.user as MeUser | undefined;
    if (u?.id) {
      setMe(u);
    }
  }, []);

  const loadUsers = useCallback(async () => {
    setUserLoading(true);
    setUserMsg(null);
    try {
      const params = new URLSearchParams();
      if (userQuery.trim()) params.set('q', userQuery.trim());
      params.set('limit', '100');
      params.set('offset', String(userOffset));
      const res = await fetchWithAuth(`/admin/users?${params.toString()}`);
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setUserMsg(toErrorMessage(res.status, (data as { error?: string }).error ?? '一覧の取得に失敗しました。'));
        return;
      }
      const data = await res.json().catch(() => ({}));
      const list = (data.users ?? data.Users ?? []) as AdminUserRow[];
      setUsers(list);
      setUserTotal(Number(data.total ?? data.Total ?? 0));
    } catch {
      setUserMsg('通信に失敗しました。');
    } finally {
      setUserLoading(false);
    }
  }, [userQuery, userOffset]);

  const loadRooms = useCallback(async () => {
    setRoomLoading(true);
    setRoomMsg(null);
    try {
      const params = new URLSearchParams();
      if (roomQuery.trim()) params.set('q', roomQuery.trim());
      params.set('limit', '100');
      params.set('offset', String(roomOffset));
      const res = await fetchWithAuth(`/admin/rooms?${params.toString()}`);
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomMsg(toErrorMessage(res.status, (data as { error?: string }).error ?? '一覧の取得に失敗しました。'));
        return;
      }
      const data = await res.json().catch(() => ({}));
      const list = (data.rooms ?? data.Rooms ?? []) as AdminRoomRow[];
      setRooms(list);
      setRoomTotal(Number(data.total ?? data.Total ?? 0));
    } catch {
      setRoomMsg('通信に失敗しました。');
    } finally {
      setRoomLoading(false);
    }
  }, [roomQuery, roomOffset]);

  useEffect(() => {
    if (!getToken()) {
      navigate('/login', { replace: true });
      return;
    }
    void loadMe();
  }, [navigate, loadMe]);

  useEffect(() => {
    if (!me || me.role !== 'admin') return;
    void loadUsers();
  }, [me, loadUsers]);

  useEffect(() => {
    if (!me || me.role !== 'admin') return;
    if (tab !== 'rooms') return;
    void loadRooms();
  }, [me, tab, loadRooms]);

  const selectedCount = useMemo(
    () => Object.entries(selectedUserIds).filter(([, v]) => v).length,
    [selectedUserIds],
  );

  const toggleUserSelect = (id: string) => {
    setSelectedUserIds((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const setUserActiveOne = async (id: string, active: boolean) => {
    setRowBusy(id);
    setUserMsg(null);
    try {
      const res = await fetchWithAuth(`/admin/users/${id}/active`, {
        method: 'PATCH',
        body: JSON.stringify({ active }),
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setUserMsg(toErrorMessage(res.status, (data as { error?: string }).error ?? '更新に失敗しました。'));
        return;
      }
      await loadUsers();
    } catch {
      setUserMsg('通信に失敗しました。');
    } finally {
      setRowBusy(null);
    }
  };

  const bulkSetActive = async (active: boolean) => {
    const ids = Object.entries(selectedUserIds)
      .filter(([, v]) => v)
      .map(([k]) => k);
    if (ids.length === 0) {
      setUserMsg('ユーザーを選択してください。');
      return;
    }
    if (!window.confirm(active ? `${ids.length} 名を有効化しますか？` : `${ids.length} 名を BAN（無効化）しますか？`)) {
      return;
    }
    setBulkBusy(true);
    setUserMsg(null);
    try {
      for (const id of ids) {
        const res = await fetchWithAuth(`/admin/users/${id}/active`, {
          method: 'PATCH',
          body: JSON.stringify({ active }),
        });
        if (!res.ok && res.status !== 401) {
          const data = await res.json().catch(() => ({}));
          setUserMsg(toErrorMessage(res.status, (data as { error?: string }).error ?? `更新失敗: ${id}`));
          break;
        }
      }
      setSelectedUserIds({});
      await loadUsers();
    } catch {
      setUserMsg('通信に失敗しました。');
    } finally {
      setBulkBusy(false);
    }
  };

  const applyRoomFreeze = async (frozen: boolean) => {
    if (!selectedRoomId) {
      setRoomMsg('ルームを選択してください。');
      return;
    }
    setRoomOpBusy(true);
    setRoomMsg(null);
    try {
      const res = await fetchWithAuth(`/admin/rooms/${selectedRoomId}/frozen`, {
        method: 'PATCH',
        body: JSON.stringify({ frozen }),
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomMsg(toErrorMessage(res.status, (data as { error?: string }).error ?? '凍結の更新に失敗しました。'));
        return;
      }
      await loadRooms();
    } catch {
      setRoomMsg('通信に失敗しました。');
    } finally {
      setRoomOpBusy(false);
    }
  };

  const forceKickMember = async () => {
    if (!selectedRoomId || !kickUserId.trim()) {
      setRoomMsg('ルームと除外するユーザー ID を指定してください。');
      return;
    }
    if (!window.confirm('選択したルームからこのユーザーを強制除外しますか？')) return;
    setRoomOpBusy(true);
    setRoomMsg(null);
    try {
      const res = await fetchWithAuth(`/admin/rooms/${selectedRoomId}/members/${kickUserId.trim()}`, {
        method: 'DELETE',
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomMsg(toErrorMessage(res.status, (data as { error?: string }).error ?? '除外に失敗しました。'));
        return;
      }
      setKickUserId('');
      await loadRooms();
    } catch {
      setRoomMsg('通信に失敗しました。');
    } finally {
      setRoomOpBusy(false);
    }
  };

  if (meError) {
    return (
      <div className="auth-page">
        <div className="card admin-card">
          <p className="error-text">{meError}</p>
          <Link to="/rooms">ルーム一覧へ</Link>
        </div>
      </div>
    );
  }

  if (!me) {
    return (
      <div className="auth-page">
        <div className="card admin-card">
          <p className="muted">読み込み中…</p>
        </div>
      </div>
    );
  }

  if (me.role !== 'admin') {
    return (
      <div className="auth-page">
        <div className="card admin-card">
          <h1>管理者画面</h1>
          <p>このアカウントは管理者ではありません。</p>
          <Link to="/rooms">ルーム一覧へ</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-page admin-page-wrap">
      <div className="card admin-card admin-card-wide">
        <h1>管理者画面</h1>
        <p className="muted admin-intro">
          ユーザー一覧・検索・選択に基づく BAN（無効化）／有効化。ルームは一覧から選択して凍結・強制除外を行います。
        </p>

        <div className="admin-tabs">
          <button
            type="button"
            data-testid="admin-tab-users"
            className={`pill-button${tab === 'users' ? ' active' : ''}`}
            onClick={() => setTab('users')}
          >
            ユーザー管理
          </button>
          <button
            type="button"
            data-testid="admin-tab-rooms"
            className={`pill-button${tab === 'rooms' ? ' active' : ''}`}
            onClick={() => setTab('rooms')}
          >
            ルーム運用
          </button>
        </div>

        {tab === 'users' && (
          <section className="admin-section">
            <h2>ユーザー一覧</h2>
            <div className="admin-toolbar">
              <input
                className="room-create-input admin-search"
                placeholder="名前・メール・UUID の一部で検索"
                value={userSearch}
                onChange={(e) => setUserSearch(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    setUserOffset(0);
                    setUserQuery(userSearch);
                  }
                }}
              />
              <button
                type="button"
                className="pill-button"
                onClick={() => {
                  setUserOffset(0);
                  setUserQuery(userSearch);
                }}
              >
                検索
              </button>
              <span className="muted admin-total">全 {userTotal} 件</span>
            </div>
            <div className="admin-bulk-row">
              <button type="button" className="pill-button danger-outline" disabled={bulkBusy || selectedCount === 0} onClick={() => void bulkSetActive(false)}>
                {bulkBusy ? '処理中…' : `選択を BAN（${selectedCount}）`}
              </button>
              <button type="button" className="pill-button" disabled={bulkBusy || selectedCount === 0} onClick={() => void bulkSetActive(true)}>
                {bulkBusy ? '処理中…' : `選択を有効化（${selectedCount}）`}
              </button>
            </div>
            {userMsg && <p className={userMsg.includes('失敗') || userMsg.includes('通信') ? 'error-text' : 'admin-ok'}>{userMsg}</p>}
            <div className="admin-table-wrap">
              {userLoading ? (
                <p className="muted">読み込み中…</p>
              ) : (
                <table className="admin-table">
                  <thead>
                    <tr>
                      <th className="admin-th-check">
                        <input
                          type="checkbox"
                          aria-label="表示中をすべて選択"
                          checked={users.length > 0 && users.every((u) => selectedUserIds[u.id])}
                          onChange={(e) => {
                            const on = e.target.checked;
                            const next: Record<string, boolean> = { ...selectedUserIds };
                            for (const u of users) {
                              next[u.id] = on;
                            }
                            setSelectedUserIds(next);
                          }}
                        />
                      </th>
                      <th>名前</th>
                      <th>メール</th>
                      <th>ロール</th>
                      <th>状態</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {users.map((u) => (
                      <tr key={u.id} data-testid={`admin-user-row-${u.email}`}>
                        <td>
                          <input type="checkbox" checked={Boolean(selectedUserIds[u.id])} onChange={() => toggleUserSelect(u.id)} />
                        </td>
                        <td title={u.id}>{u.name}</td>
                        <td className="admin-td-email">{u.email}</td>
                        <td>{u.role}</td>
                        <td>
                          <span className={u.active ? 'presence-pill is-online' : 'presence-pill is-offline'}>{u.active ? '有効' : 'BAN'}</span>
                        </td>
                        <td className="admin-td-actions">
                          {u.active ? (
                            <button
                              type="button"
                              className="pill-button danger-outline"
                              data-testid={`admin-ban-${u.id}`}
                              disabled={rowBusy === u.id}
                              onClick={() => void setUserActiveOne(u.id, false)}
                            >
                              BAN
                            </button>
                          ) : (
                            <button type="button" className="pill-button" disabled={rowBusy === u.id} onClick={() => void setUserActiveOne(u.id, true)}>
                              有効化
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
            <div className="admin-pager">
              <button type="button" className="pill-button" disabled={userOffset === 0 || userLoading} onClick={() => setUserOffset((o) => Math.max(0, o - 100))}>
                前へ
              </button>
              <button
                type="button"
                className="pill-button"
                disabled={userLoading || userOffset + users.length >= userTotal}
                onClick={() => setUserOffset((o) => o + 100)}
              >
                次へ
              </button>
            </div>
          </section>
        )}

        {tab === 'rooms' && (
          <section className="admin-section">
            <h2>ルーム一覧</h2>
            <div className="admin-toolbar">
              <input
                className="room-create-input admin-search"
                placeholder="ルーム ID・名前の一部で検索"
                value={roomSearch}
                onChange={(e) => setRoomSearch(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    setRoomOffset(0);
                    setRoomQuery(roomSearch);
                  }
                }}
              />
              <button
                type="button"
                className="pill-button"
                onClick={() => {
                  setRoomOffset(0);
                  setRoomQuery(roomSearch);
                }}
              >
                検索
              </button>
              <span className="muted admin-total">全 {roomTotal} 件</span>
            </div>
            {roomMsg && <p className={roomMsg.includes('失敗') || roomMsg.includes('通信') ? 'error-text' : 'admin-ok'}>{roomMsg}</p>}
            <div className="admin-table-wrap">
              {roomLoading ? (
                <p className="muted">読み込み中…</p>
              ) : (
                <table className="admin-table" data-testid="admin-room-table">
                  <thead>
                    <tr>
                      <th className="admin-th-radio">選択</th>
                      <th>種別</th>
                      <th>名前</th>
                      <th>凍結</th>
                      <th>更新</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rooms.map((r) => (
                      <tr key={r.id}>
                        <td>
                          <input type="radio" name="admin-room-pick" checked={selectedRoomId === r.id} onChange={() => setSelectedRoomId(r.id)} />
                        </td>
                        <td>{r.type}</td>
                        <td className="admin-td-mono" title={r.id}>
                          {r.name?.trim() || r.id.slice(0, 8)}
                        </td>
                        <td>{r.is_frozen ? 'はい' : 'いいえ'}</td>
                        <td className="admin-td-mono">{new Date(r.updated_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
            <div className="admin-room-actions">
              <button type="button" className="pill-button danger-outline" disabled={roomOpBusy || !selectedRoomId} onClick={() => void applyRoomFreeze(true)}>
                選択ルームを凍結
              </button>
              <button type="button" className="pill-button" disabled={roomOpBusy || !selectedRoomId} onClick={() => void applyRoomFreeze(false)}>
                選択ルームの凍結を解除
              </button>
            </div>
            <div className="admin-force-row">
              <label className="room-settings-label">強制除外（ルームは上で選択）</label>
              <select className="room-create-input admin-select" value={kickUserId} onChange={(e) => setKickUserId(e.target.value)}>
                <option value="">ユーザー一覧から選択…</option>
                {users.map((u) => (
                  <option key={u.id} value={u.id}>
                    {u.name} ({u.email})
                  </option>
                ))}
              </select>
              <button type="button" className="pill-button danger-outline" disabled={roomOpBusy || !selectedRoomId || !kickUserId} onClick={() => void forceKickMember()}>
                強制除外
              </button>
            </div>
            <p className="muted admin-hint">ユーザー候補は「ユーザー管理」タブで最後に検索した一覧です。見つからない場合はユーザー側で検索してください。</p>
            <div className="admin-pager">
              <button type="button" className="pill-button" disabled={roomOffset === 0 || roomLoading} onClick={() => setRoomOffset((o) => Math.max(0, o - 100))}>
                前へ
              </button>
              <button
                type="button"
                className="pill-button"
                disabled={roomLoading || roomOffset + rooms.length >= roomTotal}
                onClick={() => setRoomOffset((o) => o + 100)}
              >
                次へ
              </button>
            </div>
          </section>
        )}

        <p className="auth-footer-link">
          <Link to="/rooms">ルーム一覧へ</Link>
        </p>
      </div>
    </div>
  );
}
