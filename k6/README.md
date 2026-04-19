# k6（API 負荷テスト）

**シナリオは `scenarios/spec_api.js` のみ**です（`/health` → 登録 → グループ → 各 VU で `/me`・`/rooms`・メッセージ POST・一覧 GET・既読 POST）。事前の DB シードは不要です。

ルートの [README.md](../README.md) にも同じ実行例を載せています。

## 前提

- API が起動していること（例: `cd backend && go run .`）。
- Postgres がマイグレーション済みであること。

## 実行（リポジトリルートで）

### k6 本体がある場合

```bash
k6 run k6/scenarios/spec_api.js
```

### Docker のみの場合

```bash
docker run --rm -i --network=host \
  -e K6_API_BASE_URL="${K6_API_BASE_URL:-http://127.0.0.1:8080}" \
  -e K6_VUS="${K6_VUS:-4}" \
  -e K6_DURATION="${K6_DURATION:-35s}" \
  -v "$(pwd)/k6:/k6:ro" \
  grafana/k6:0.55.0 run /k6/scenarios/spec_api.js
```

Docker Desktop（Mac）で API がホストの `localhost:8080` のときは、`K6_API_BASE_URL=http://host.docker.internal:8080` を付けることがあります。

## 環境変数

| 変数 | 既定 | 説明 |
|------|------|------|
| `K6_API_BASE_URL` | `http://127.0.0.1:8080` | API オリジン |
| `K6_VUS` | `4` | 仮想ユーザー数（`K6_SPEC_VUS` も可） |
| `K6_DURATION` | `35s` | 実行時間（`K6_SPEC_DURATION` も可） |
| `K6_SLEEP_SEC` | `0.2` | 1 イテレーションあたりの sleep（秒） |

## CI

`.github/workflows/ci.yml` の **k6-spec** ジョブで上記と同じ `spec_api.js` を 1 回だけ実行します。
