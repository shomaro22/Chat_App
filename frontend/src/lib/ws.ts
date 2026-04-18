import { getApiOrigin } from './http';

/** JWT 付きで `/ws` に接続する URL（開発時は API オリジンへ）。 */
export function buildWsUrl(accessToken: string): string {
  const base = getApiOrigin();
  if (base) {
    let u: URL;
    try {
      u = new URL(base);
    } catch {
      u = new URL(window.location.origin);
    }
    const wsProto = u.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${wsProto}//${u.host}/ws?access_token=${encodeURIComponent(accessToken)}`;
  }
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
  return `${protocol}://${window.location.host}/ws?access_token=${encodeURIComponent(accessToken)}`;
}
