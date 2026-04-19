import { useEffect, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { fetchWithAuth, getToken } from '../lib/api';
import { toErrorMessage } from '../chat/errors';
import type { MeUser } from '../types/chat';

export function SettingsPage() {
  const [me, setMe] = useState<MeUser | null>(null);
  const [name, setName] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');
  const [currentPw, setCurrentPw] = useState('');
  const [newPw, setNewPw] = useState('');
  const [msg, setMsg] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!getToken()) return;
    void (async () => {
      const res = await fetchWithAuth('/me');
      if (!res.ok) return;
      const data = await res.json().catch(() => ({}));
      const u = data.user as MeUser | undefined;
      if (u) {
        setMe(u);
        setName(u.name);
        setAvatarUrl(u.avatar_url ?? '');
      }
    })();
  }, []);

  const saveProfile = async (e: FormEvent) => {
    e.preventDefault();
    setErr(null);
    setMsg(null);
    const body: { name?: string; avatar_url?: string } = {};
    if (name.trim()) body.name = name.trim();
    if (avatarUrl.trim()) body.avatar_url = avatarUrl.trim();
    const res = await fetchWithAuth('/me', { method: 'PATCH', body: JSON.stringify(body) });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      setErr(toErrorMessage(res.status, (data as { error?: string }).error ?? '更新に失敗しました。'));
      return;
    }
    setMsg('プロフィールを保存しました。');
  };

  const changePassword = async (e: FormEvent) => {
    e.preventDefault();
    setErr(null);
    setMsg(null);
    const res = await fetchWithAuth('/me/change-password', {
      method: 'POST',
      body: JSON.stringify({ current_password: currentPw, new_password: newPw }),
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      setErr(toErrorMessage(res.status, (data as { error?: string }).error ?? 'パスワード変更に失敗しました。'));
      return;
    }
    setCurrentPw('');
    setNewPw('');
    setMsg('パスワードを変更しました。');
  };

  return (
    <div className="auth-page">
      <div className="card admin-card-wide">
        <h1>設定</h1>
        <p className="muted">
          <Link to="/rooms">← チャットに戻る</Link>
        </p>
        {me && (
          <p className="muted">
            {me.email} {me.email_verified === false && <span>（メール未確認）</span>}
          </p>
        )}
        {err && <p className="error-text">{err}</p>}
        {msg && <p style={{ color: 'seagreen' }}>{msg}</p>}
        <form onSubmit={saveProfile}>
          <h2>プロフィール</h2>
          <div className="field">
            <label htmlFor="set-name">表示名</label>
            <input id="set-name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="field">
            <label htmlFor="set-avatar">アバター URL（/static/uploads/...）</label>
            <input id="set-avatar" value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} />
          </div>
          <button className="button" type="submit">
            保存
          </button>
        </form>
        <form onSubmit={changePassword} style={{ marginTop: 24 }}>
          <h2>パスワード変更</h2>
          <div className="field">
            <label htmlFor="cur-pw">現在のパスワード</label>
            <input
              id="cur-pw"
              type="password"
              value={currentPw}
              onChange={(e) => setCurrentPw(e.target.value)}
              autoComplete="current-password"
            />
          </div>
          <div className="field">
            <label htmlFor="new-pw">新しいパスワード（8文字以上）</label>
            <input
              id="new-pw"
              type="password"
              value={newPw}
              onChange={(e) => setNewPw(e.target.value)}
              autoComplete="new-password"
            />
          </div>
          <button className="button" type="submit">
            パスワードを変更
          </button>
        </form>
        {import.meta.env.DEV && (
          <section style={{ marginTop: 28, paddingTop: 20, borderTop: '1px solid var(--border)' }}>
            <h2>接続・プレゼンス（開発用）</h2>
            <p className="muted" style={{ fontSize: 13, lineHeight: 1.5 }}>
              同一アカウントで複数タブを開くと WebSocket が複数張られ、相手のルーム画面のメンバー一覧ではオンラインが維持されます。片方のタブだけ閉じても、残りが1つあればオンラインのままです。すべて閉じるとオフラインになります。
            </p>
            <button
              type="button"
              className="pill-button"
              data-testid="dev-open-rooms-tab"
              onClick={() => {
                window.open('/rooms', '_blank', 'noopener,noreferrer');
              }}
            >
              同じログインでルームを別タブで開く
            </button>
          </section>
        )}
      </div>
    </div>
  );
}
