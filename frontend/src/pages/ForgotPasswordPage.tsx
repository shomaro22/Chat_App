import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { connectionFailureHint, fetchApi } from '../lib/api';

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setErr(null);
    try {
      const res = await fetchApi('/auth/forgot-password', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim() }),
      });
      if (!res.ok) {
        setErr('送信に失敗しました。');
        return;
      }
      setSent(true);
    } catch {
      setErr(connectionFailureHint());
    }
  };

  return (
    <div className="auth-page">
      <div className="card">
        <h1>パスワードを忘れた場合</h1>
        {sent ? (
          <p>登録されている場合は、再設定用のメールを送信しました（ログに出力される場合があります）。</p>
        ) : (
          <form onSubmit={submit}>
            {err && <p className="error-text">{err}</p>}
            <div className="field">
              <label htmlFor="fp-email">メールアドレス</label>
              <input
                id="fp-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoComplete="email"
              />
            </div>
            <button className="button" type="submit">
              送信
            </button>
          </form>
        )}
        <p className="auth-footer-link">
          <Link to="/login">ログインへ</Link>
        </p>
      </div>
    </div>
  );
}
