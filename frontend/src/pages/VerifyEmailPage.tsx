import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { fetchApi } from '../lib/api';

export function VerifyEmailPage() {
  const [params] = useSearchParams();
  const token = params.get('token');
  const [msg, setMsg] = useState(token ? '確認中…' : 'トークンがありません。');

  useEffect(() => {
    if (!token) {
      return;
    }
    void (async () => {
      const res = await fetchApi('/auth/verify-email', {
        method: 'POST',
        body: JSON.stringify({ token }),
      });
      if (res.ok) setMsg('メール確認が完了しました。ログインしてください。');
      else setMsg('確認に失敗しました。リンクの期限切れの可能性があります。');
    })();
  }, [token]);

  return (
    <div className="auth-page">
      <div className="card">
        <h1>メール確認</h1>
        <p>{msg}</p>
        <Link to="/login">ログインへ</Link>
      </div>
    </div>
  );
}
