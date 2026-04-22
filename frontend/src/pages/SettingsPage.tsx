import { useEffect, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { fetchWithAuth, getToken } from '../lib/api';
import { toErrorMessage } from '../chat/errors';
import type { MeUser } from '../types/chat';

export function SettingsPage() {
  const [me, setMe] = useState<MeUser | null>(null);
  const [name, setName] = useState('');
  const [currentPw, setCurrentPw] = useState('');
  const [newPw, setNewPw] = useState('');
  const [showNewPassword, setShowNewPassword] = useState(false);
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
      }
    })();
  }, []);

  const saveProfile = async (e: FormEvent) => {
    e.preventDefault();
    setErr(null);
    setMsg(null);
    const body: { name?: string } = {};
    if (name.trim()) body.name = name.trim();
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
      <div className="card admin-card">
        <h1>設定</h1>
        <p className="muted settings-back-link">
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
            <div className="password-input-wrap">
              <input
                id="new-pw"
                type={showNewPassword ? 'text' : 'password'}
                value={newPw}
                onChange={(e) => setNewPw(e.target.value)}
                autoComplete="new-password"
              />
              <button
                type="button"
                className="password-toggle"
                aria-label={showNewPassword ? '新しいパスワードを隠す' : '新しいパスワードを表示'}
                aria-pressed={showNewPassword}
                onClick={() => setShowNewPassword((v) => !v)}
              >
                {showNewPassword ? '隠す' : '表示'}
              </button>
            </div>
          </div>
          <button className="button" type="submit">
            パスワードを変更
          </button>
        </form>
      </div>
    </div>
  );
}
