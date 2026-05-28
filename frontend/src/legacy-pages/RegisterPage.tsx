import { useEffect, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { connectionFailureHint, fetchApi, getToken, setToken } from '../lib/api';

function registerErrorMessage(status: number, apiError: string): string {
  if (status === 409) return 'このメールアドレスは既に登録されています。';
  if (status === 429) {
    return '試行回数が多すぎます。しばらく待ってから再度お試しください。';
  }
  if (apiError) return apiError;
  return '登録に失敗しました。';
}

export function RegisterPage() {
  const navigate = useNavigate();

  useEffect(() => {
    if (getToken()) {
      navigate('/rooms', { replace: true });
    }
  }, [navigate]);

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const res = await fetchApi('/auth/register', {
        method: 'POST',
        body: JSON.stringify({ name, email, password }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        const raw = typeof data.error === 'string' ? data.error : '';
        setError(registerErrorMessage(res.status, raw));
        setPassword('');
        return;
      }
      if (data.verification_required) {
        setError(null);
        setPassword('');
        alert('確認メールを送信しました。メール内のリンクで認証後、ログインしてください。');
        navigate('/login', { replace: true });
        return;
      }
      if (!data.access_token) {
        setError('不正なレスポンスです');
        setPassword('');
        return;
      }
      setToken(data.access_token);
      navigate('/rooms', { replace: true });
    } catch {
      setError(connectionFailureHint());
      setPassword('');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="card">
        <h1>アカウント登録</h1>
        <form onSubmit={handleSubmit}>
          <div className="field">
            <label htmlFor="reg-name">表示名</label>
            <input
              id="reg-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoComplete="name"
            />
          </div>
          <div className="field">
            <label htmlFor="reg-email">メールアドレス</label>
            <input
              id="reg-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
            />
          </div>
          <div className="field">
            <label htmlFor="reg-password">パスワード</label>
            <div className="password-input-wrap">
              <input
                id="reg-password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                autoComplete="new-password"
                minLength={8}
              />
              <button
                type="button"
                className="password-toggle"
                aria-label={showPassword ? 'パスワードを隠す' : 'パスワードを表示'}
                aria-pressed={showPassword}
                onClick={() => setShowPassword((v) => !v)}
              >
                {showPassword ? '隠す' : '表示'}
              </button>
            </div>
          </div>
          {error && (
            <p style={{ color: 'tomato', fontSize: 14, marginBottom: 8 }}>
              {error}
            </p>
          )}
          <button className="button" type="submit" disabled={loading}>
            {loading ? '登録中...' : '登録'}
          </button>
        </form>
        <p className="auth-footer-link">
          <Link to="/login">ログインへ</Link>
        </p>
      </div>
    </div>
  );
}
