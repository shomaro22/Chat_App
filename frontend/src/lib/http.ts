import { getToken, setToken } from './token';

export function getApiOrigin(): string {
  return (import.meta.env.VITE_API_ORIGIN as string | undefined)?.trim().replace(/\/$/, '') ?? '';
}

/** ネットワーク失敗時に UI に出す文言（内部構成や手順は書かない）。 */
export function connectionFailureHint(): string {
  return 'サーバーに接続できませんでした。時間をおいて再度お試しください。';
}

/** 本番は同一オリジン、開発は `VITE_API_ORIGIN`（例: http://127.0.0.1:8080）へ。 */
export function resolveApiUrl(path: string): string {
  if (path.startsWith('http://') || path.startsWith('https://')) {
    return path;
  }
  const p = path.startsWith('/') ? path : `/${path}`;
  const base = getApiOrigin();
  return base ? `${base}${p}` : p;
}

/** `/static/...` を API オリジン付きで解決（Vite と API が別ポートのとき用）。 */
export function resolvePublicAssetUrl(path: string | null | undefined): string {
  if (!path) return '';
  if (path.startsWith('http://') || path.startsWith('https://')) return path;
  const base = getApiOrigin();
  return base ? `${base.replace(/\/$/, '')}${path.startsWith('/') ? path : `/${path}`}` : path;
}

type UnauthorizedHandler = () => void;

let unauthorizedHandler: UnauthorizedHandler | null = null;

/** 認証付き API で 401 が返ったときに一度だけ呼ばれる（ログアウト・ログインへ誘導など）。 */
export function setUnauthorizedHandler(handler: UnauthorizedHandler | null) {
  unauthorizedHandler = handler;
}

function notifyUnauthorized() {
  setToken(null);
  unauthorizedHandler?.();
}

export async function fetchApi(path: string, options: RequestInit = {}): Promise<Response> {
  const url = resolveApiUrl(path);
  return fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers ?? {}),
    },
  });
}

export async function fetchWithAuth(path: string, options: RequestInit = {}): Promise<Response> {
  const token = getToken();
  const url = resolveApiUrl(path);
  const isFormData = typeof FormData !== 'undefined' && options.body instanceof FormData;

  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string> | undefined),
  };
  if (!isFormData) {
    headers['Content-Type'] = 'application/json';
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const res = await fetch(url, {
    ...options,
    headers,
  });

  if (res.status === 401) {
    notifyUnauthorized();
  }

  return res;
}
