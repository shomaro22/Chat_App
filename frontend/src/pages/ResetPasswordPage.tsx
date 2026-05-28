import { useState, type FormEvent } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { fetchApi } from '../lib/api';

export function ResetPasswordPage() {
  const [params] = useSearchParams();
  const [pw, setPw] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setErr(null);
    const token = params.get('token');
    if (!token) {
      setErr('トークンがありません。');
      return;
    }
    const res = await fetchApi('/auth/reset-password', {
      method: 'POST',
      body: JSON.stringify({ token, new_password: pw }),
    });
    if (res.ok) {
      setMsg('パスワードを更新しました。ログインしてください。');
      return;
    }
    const data = await res.json().catch(() => ({}));
    setErr(typeof data.error === 'string' ? data.error : '失敗しました。');
  };

  return (
    <div className="auth-page">
      <div className="card">
        <h1>パスワード再設定</h1>
        {msg ? (
          <p>{msg}</p>
        ) : (
          <form onSubmit={submit}>
            {err && <p className="error-text">{err}</p>}
            <div className="field">
              <label htmlFor="npw">新しいパスワード</label>
              <div className="password-input-wrap">
                <input
                  id="npw"
                  type={showPassword ? 'text' : 'password'}
                  value={pw}
                  onChange={(e) => setPw(e.target.value)}
                  required
                  minLength={8}
                />
                <button
                  type="button"
                  className="password-toggle"
                  aria-label={showPassword ? '新しいパスワードを隠す' : '新しいパスワードを表示'}
                  aria-pressed={showPassword}
                  onClick={() => setShowPassword((v) => !v)}
                >
                  {showPassword ? '隠す' : '表示'}
                </button>
              </div>
            </div>
            <button className="button" type="submit">
              更新
            </button>
          </form>
        )}
        <p>
          <Link to="/login">ログインへ</Link>
        </p>
      </div>
    </div>
  );
}
