import { useEffect, useState, type FormEvent } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { connectionFailureHint, fetchApi, getToken, setToken } from '../lib/api';

function loginErrorMessage(status: number, apiError: string): string {
  if (status === 401 && apiError === 'invalid email or password') {
    return 'メールアドレスまたはパスワードが正しくありません。';
  }
  if (status === 429) {
    return '試行回数が多すぎます。しばらく待ってから再度お試しください。';
  }
  return 'ログインに失敗しました。';
}

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const fromPath =
    (location.state as { from?: { pathname?: string } } | null)?.from?.pathname ?? '/rooms';

  useEffect(() => {
    if (getToken()) {
      navigate('/rooms', { replace: true });
    }
  }, [navigate]);
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
      const res = await fetchApi('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        const raw = typeof data.error === 'string' ? data.error : '';
        setError(loginErrorMessage(res.status, raw));
        setPassword('');
        return;
      }
      // backend returns { access_token: string }
      if (!data.access_token) {
        setError('不正なレスポンスです');
        setPassword('');
        return;
      }
      setToken(data.access_token);
      navigate(fromPath.startsWith('/') ? fromPath : '/rooms', { replace: true });
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
        <h1>チャットにログイン</h1>
        <form onSubmit={handleSubmit}>
          <div className="field">
            <label htmlFor="email">メールアドレス</label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
            />
          </div>
          <div className="field">
            <label htmlFor="password">パスワード</label>
            <div className="password-input-wrap">
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                autoComplete="current-password"
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
            {loading ? 'ログイン中...' : 'ログイン'}
          </button>
        </form>
        <p className="auth-footer-link">
          <Link to="/register">アカウント作成</Link>
        </p>
      </div>
    </div>
  );
}

