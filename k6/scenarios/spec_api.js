/**
 * API 負荷テスト（単一シナリオ）
 * - setup: GET /health → ユーザー2名登録 → ログイン → グループ作成
 * - VU 毎: GET /me, GET /rooms, POST メッセージ, GET メッセージ一覧, POST 既読
 *
 * 環境変数: K6_API_BASE_URL, K6_VUS, K6_DURATION, K6_SLEEP_SEC
 * （旧 K6_SPEC_VUS / K6_SPEC_DURATION も互換で受け付け）
 */
import http from 'k6/http';
import { check, sleep } from 'k6';

function baseURL() {
  return (__ENV.K6_API_BASE_URL || 'http://127.0.0.1:8080').replace(/\/$/, '');
}

function vus() {
  const v = __ENV.K6_VUS || __ENV.K6_SPEC_VUS;
  return Number(v || 4);
}

function duration() {
  return __ENV.K6_DURATION || __ENV.K6_SPEC_DURATION || '35s';
}

export const options = {
  scenarios: {
    api: {
      executor: 'constant-vus',
      vus: vus(),
      duration: duration(),
    },
  },
  thresholds: {
    checks: ['rate>0.95'],
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<8000'],
  },
};

export function setup() {
  const base = baseURL();
  const health = http.get(`${base}/health`);
  if (!check(health, { 'GET /health ok': (r) => r.status === 200 && String(r.body).trim() === 'ok' })) {
    throw new Error(`GET /health failed: ${health.status} ${String(health.body).slice(0, 200)}`);
  }

  const suffix = Date.now();
  const ownerEmail = `k6own${suffix}@local.test`;
  const peerEmail = `k6peer${suffix}@local.test`;
  const pass = 'TestPassw0rd!12';

  const regPeer = http.post(
    `${base}/auth/register`,
    JSON.stringify({ name: 'Peer', email: peerEmail, password: pass }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  if (!check(regPeer, { 'peer reg 201': (r) => r.status === 201 })) {
    throw new Error(`peer register: ${regPeer.status} ${String(regPeer.body).slice(0, 300)}`);
  }
  const peerBody = regPeer.json();
  const peerId = peerBody.user && peerBody.user.id;
  if (!peerId) {
    throw new Error('no peer user id');
  }

  const regOwn = http.post(
    `${base}/auth/register`,
    JSON.stringify({ name: 'Owner', email: ownerEmail, password: pass }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  if (!check(regOwn, { 'owner reg 201': (r) => r.status === 201 })) {
    throw new Error(`owner register: ${regOwn.status}`);
  }

  const lr = http.post(
    `${base}/auth/login`,
    JSON.stringify({ email: ownerEmail, password: pass }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  if (!check(lr, { 'owner login 200': (r) => r.status === 200 })) {
    throw new Error(`owner login: ${lr.status} ${lr.body}`);
  }
  const token = lr.json('access_token');

  const gr = http.post(
    `${base}/rooms/groups`,
    JSON.stringify({ name: `k6-${suffix}`, member_ids: [peerId] }),
    {
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    },
  );
  if (!check(gr, { 'group 201': (r) => r.status === 201 })) {
    throw new Error(`group: ${gr.status} ${gr.body}`);
  }
  const roomId = gr.json('id');
  return { base, token, roomId, pass, peerEmail };
}

export default function (data) {
  const auth = {
    headers: {
      Authorization: `Bearer ${data.token}`,
      'Content-Type': 'application/json',
    },
  };

  const me = http.get(`${data.base}/me`, { headers: { Authorization: `Bearer ${data.token}` } });
  check(me, { 'GET /me 200': (r) => r.status === 200 });

  const rooms = http.get(`${data.base}/rooms`, { headers: { Authorization: `Bearer ${data.token}` } });
  check(rooms, { 'GET /rooms 200': (r) => r.status === 200 });

  const send = http.post(
    `${data.base}/rooms/${data.roomId}/messages`,
    JSON.stringify({ content: `k6 iter ${__ITER}` }),
    auth,
  );
  check(send, { 'POST message 201': (r) => r.status === 201 });

  const list = http.get(`${data.base}/rooms/${data.roomId}/messages?limit=10`, {
    headers: { Authorization: `Bearer ${data.token}` },
  });
  check(list, { 'GET messages 200': (r) => r.status === 200 });
  let mid = null;
  try {
    const j = list.json();
    if (j.messages && j.messages.length > 0) {
      mid = j.messages[0].id;
    }
  } catch (_) {
    /* ignore */
  }
  if (mid) {
    const rd = http.post(
      `${data.base}/rooms/${data.roomId}/read`,
      JSON.stringify({ message_id: mid }),
      auth,
    );
    check(rd, { 'POST read 204': (r) => r.status === 204 });
  }

  sleep(Number(__ENV.K6_SLEEP_SEC || 0.2));
}
