import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { fetchWithAuth } from '../lib/api';
import { toErrorMessage } from '../chat/errors';
import { ErrorNotice } from '../components/ErrorNotice';
import type { Message } from '../types/chat';

export function SearchPage() {
  const navigate = useNavigate();
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [rows, setRows] = useState<Message[]>([]);

  const performSearch = async () => {
    setLoading(true);
    setErr(null);
    setRows([]);
    try {
      const params = new URLSearchParams({ q: q.trim(), limit: '30' });
      const res = await fetchWithAuth(`/search/messages?${params.toString()}`);
      if (res.status === 401) {
        navigate('/login');
        return;
      }
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setErr(toErrorMessage(res.status, (data as { error?: string }).error ?? '検索に失敗しました。'));
        return;
      }
      const data = await res.json().catch(() => ({}));
      const list = (data.messages ?? []) as Message[];
      setRows(list);
    } finally {
      setLoading(false);
    }
  };

  const run = async (e: FormEvent) => {
    e.preventDefault();
    await performSearch();
  };

  return (
    <div className="auth-page">
      <div className="card admin-card">
        <h1>メッセージ検索</h1>
        <p className="back-link-emphasis">
          <Link to="/rooms">← チャットに戻る</Link>
        </p>
        <form onSubmit={run} className="admin-toolbar">
          <input
            className="room-create-input admin-search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="2文字以上"
          />
          <button className="button" type="submit" disabled={loading || q.trim().length < 2}>
            {loading ? '検索中…' : '検索'}
          </button>
        </form>
        {err && <ErrorNotice message={err} onRetry={() => void performSearch()} />}
        <ul style={{ listStyle: 'none', padding: 0 }}>
          {rows.map((m) => (
            <li key={m.id} style={{ marginBottom: 12, borderBottom: '1px solid #eee', paddingBottom: 8 }}>
              <button type="button" className="pill-button" onClick={() => navigate(`/rooms/${m.room_id}?message_id=${encodeURIComponent(m.id)}`)}>
                ルームへ
              </button>{' '}
              <span className="muted">{m.sender_name ?? m.sender_id}</span>
              <div>{m.content}</div>
              <div className="muted" style={{ fontSize: 12 }}>
                {m.file_name ?? ''}
              </div>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
