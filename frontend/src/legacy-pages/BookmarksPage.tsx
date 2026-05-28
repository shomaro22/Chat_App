import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { fetchWithAuth, resolvePublicAssetUrl } from '../lib/api';
import { toErrorMessage } from '../chat/errors';
import { ErrorNotice } from '../components/ErrorNotice';
import type { Message } from '../types/chat';

type BookmarkRow = {
  room_id: string;
  message_id: string;
  message: Message;
};

export function BookmarksPage() {
  const navigate = useNavigate();
  const [rows, setRows] = useState<BookmarkRow[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [reloadSeq, setReloadSeq] = useState(0);

  useEffect(() => {
    let alive = true;
    void (async () => {
      const res = await fetchWithAuth('/me/bookmarks?limit=100');
      if (!alive) return;
      if (res.status === 401) {
        navigate('/login');
        return;
      }
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setErr(toErrorMessage(res.status, (data as { error?: string }).error ?? '取得に失敗しました。'));
        return;
      }
      const data = await res.json().catch(() => ({}));
      setErr(null);
      setRows((data.bookmarks ?? []) as BookmarkRow[]);
    })();
    return () => {
      alive = false;
    };
  }, [navigate, reloadSeq]);

  return (
    <div className="auth-page">
      <div className="card admin-card-wide">
        <h1>ブックマーク</h1>
        <p className="back-link-emphasis">
          <Link to="/rooms">← チャットに戻る</Link>
        </p>
        {err && <ErrorNotice message={err} onRetry={() => setReloadSeq((v) => v + 1)} />}
        <ul style={{ listStyle: 'none', padding: 0 }}>
          {rows.map((b) => {
            const m = b.message;
            return (
              <li key={`${b.room_id}-${b.message_id}`} style={{ marginBottom: 14, borderBottom: '1px solid #eee', paddingBottom: 10 }}>
                <button
                  type="button"
                  className="pill-button"
                  onClick={() => navigate(`/rooms/${b.room_id}?message_id=${encodeURIComponent(b.message_id)}`)}
                >
                  ルームへ
                </button>{' '}
                <span className="muted">{m.sender_name ?? m.sender_id}</span>
                {m.content && <div>{m.content}</div>}
                {m.image_url && (
                  <div className="bubble image" style={{ maxWidth: 200 }}>
                    <img src={resolvePublicAssetUrl(m.image_url)} alt="" />
                  </div>
                )}
                {m.file_url && (
                  <div>
                    <a href={resolvePublicAssetUrl(m.file_url)} target="_blank" rel="noreferrer">
                      {m.file_name ?? '添付'}
                    </a>
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}
