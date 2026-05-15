import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { fetchWithAuth } from '../lib/api';
import { toErrorMessage } from '../chat/errors';
import { ErrorNotice } from '../components/ErrorNotice';
import type { Message } from '../types/chat';

const VOICE_SEARCH_LANG = 'ja-JP';

type SpeechRecognitionAlternativeLike = {
  transcript: string;
};

type SpeechRecognitionResultLike = {
  isFinal: boolean;
  0?: SpeechRecognitionAlternativeLike;
};

type SpeechRecognitionResultListLike = {
  length: number;
  [index: number]: SpeechRecognitionResultLike;
};

type SpeechRecognitionEventLike = Event & {
  resultIndex: number;
  results: SpeechRecognitionResultListLike;
};

type SpeechRecognitionErrorEventLike = Event & {
  error: string;
};

type SpeechRecognitionLike = {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  maxAlternatives: number;
  onend: (() => void) | null;
  onerror: ((event: SpeechRecognitionErrorEventLike) => void) | null;
  onresult: ((event: SpeechRecognitionEventLike) => void) | null;
  abort: () => void;
  start: () => void;
  stop: () => void;
};

type SpeechRecognitionConstructorLike = new () => SpeechRecognitionLike;

type SpeechRecognitionWindow = Window &
  typeof globalThis & {
    SpeechRecognition?: SpeechRecognitionConstructorLike;
    webkitSpeechRecognition?: SpeechRecognitionConstructorLike;
  };

function getSpeechRecognitionConstructor() {
  if (typeof window === 'undefined') return null;
  const speechWindow = window as SpeechRecognitionWindow;
  return speechWindow.SpeechRecognition ?? speechWindow.webkitSpeechRecognition ?? null;
}

function extractTranscript(event: SpeechRecognitionEventLike) {
  const fragments: string[] = [];
  for (let i = event.resultIndex; i < event.results.length; i += 1) {
    const result = event.results[i];
    const transcript = result?.[0]?.transcript.trim();
    if (result?.isFinal && transcript) {
      fragments.push(transcript);
    }
  }
  return fragments.join(' ').trim();
}

function toVoiceSearchErrorMessage(error: string) {
  switch (error) {
    case 'not-allowed':
    case 'service-not-allowed':
      return '音声検索にはマイクの許可が必要です。ブラウザの設定を確認してください。';
    case 'no-speech':
      return '音声を認識できませんでした。もう一度日本語で話してください。';
    case 'audio-capture':
      return 'マイクを利用できません。接続や入力デバイスを確認してください。';
    case 'network':
      return '音声認識サービスに接続できませんでした。通信状況を確認してください。';
    default:
      return '音声検索に失敗しました。もう一度お試しください。';
  }
}

export function SearchPage() {
  const navigate = useNavigate();
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [voiceErr, setVoiceErr] = useState<string | null>(null);
  const [listening, setListening] = useState(false);
  const [rows, setRows] = useState<Message[]>([]);
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null);
  const supportsVoiceSearch = getSpeechRecognitionConstructor() !== null;

  useEffect(() => {
    return () => {
      recognitionRef.current?.abort();
    };
  }, []);

  const performSearch = async (query = q) => {
    const trimmedQuery = query.trim();
    if (trimmedQuery.length < 2) {
      setRows([]);
      return;
    }
    setLoading(true);
    setErr(null);
    setRows([]);
    try {
      const params = new URLSearchParams({ q: trimmedQuery, limit: '30' });
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

  const startVoiceSearch = () => {
    if (listening) {
      recognitionRef.current?.stop();
      return;
    }

    const SpeechRecognitionConstructor = getSpeechRecognitionConstructor();
    if (!SpeechRecognitionConstructor) {
      setVoiceErr('このブラウザは音声検索に対応していません。');
      return;
    }

    recognitionRef.current?.abort();

    const recognition = new SpeechRecognitionConstructor();
    recognition.lang = VOICE_SEARCH_LANG;
    recognition.continuous = false;
    recognition.interimResults = false;
    recognition.maxAlternatives = 1;

    recognition.onresult = (event) => {
      const transcript = extractTranscript(event);
      if (!transcript) return;
      setQ(transcript);
      setVoiceErr(null);
      void performSearch(transcript);
    };
    recognition.onerror = (event) => {
      setVoiceErr(toVoiceSearchErrorMessage(event.error));
    };
    recognition.onend = () => {
      if (recognitionRef.current === recognition) {
        recognitionRef.current = null;
      }
      setListening(false);
    };

    recognitionRef.current = recognition;
    setVoiceErr(null);
    setListening(true);

    try {
      recognition.start();
    } catch {
      recognitionRef.current = null;
      setListening(false);
      setVoiceErr('音声検索を開始できませんでした。もう一度お試しください。');
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
          <button
            className={`pill-button${listening ? ' active' : ''}`}
            type="button"
            onClick={startVoiceSearch}
            disabled={!supportsVoiceSearch}
            aria-pressed={listening}
          >
            {listening ? '聞き取り中…' : '音声検索'}
          </button>
          <button className="button" type="submit" disabled={loading || q.trim().length < 2}>
            {loading ? '検索中…' : '検索'}
          </button>
        </form>
        <p className="muted voice-search-note">
          音声検索は日本語（{VOICE_SEARCH_LANG}）で認識します。
        </p>
        {voiceErr && <ErrorNotice message={voiceErr} onRetry={supportsVoiceSearch ? startVoiceSearch : undefined} />}
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
