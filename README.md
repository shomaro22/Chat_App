# Chat_App

## 起動方法（バックエンド）

1. **依存**: [Go](https://go.dev/dl/)（`backend/go.mod` のバージョン）、[Docker](https://docs.docker.com/get-docker/)（Postgres / Redis 用）

2. **DB と Redis を起動**（リポジトリ**ルート**の `docker-compose.yml` のみ使用）

   ```bash
   docker compose up -d postgres redis
   ```

   Postgres（`localhost:5432`）と Redis（`localhost:6379`）が立ち上がります。API 〜 フロントまでコンテナで動かす場合は `docker compose up -d`。

3. **バックエンドディレクトリへ移動**（以降の手順は `backend` で実行）

   ```bash
   cd backend
   ```

4. **環境変数**

   ```bash
   cp .env.example .env
   ```

   - **`DATABASE_URL`**: 上記 compose のままなら `.env.example` の値でよい
   - **`REDIS_ADDR`**: ホストで `go run .` する場合は `localhost:6379`。空にすると Redis レート制限は無効（スキップ）
   - **`JWT_SECRET`**: 本番では 32 文字以上を設定（未設定時は開発用の弱い既定値）

5. **API サーバー起動**

   **`backend` をカレントディレクトリにしたまま** `go run .` すると、同じディレクトリに `.env` があれば起動時に読み込みます（`github.com/joho/godotenv`）。すでにシェルや IDE で入っている環境変数は上書きしません。

   ```bash
   go run .
   ```

   既定は `http://localhost:8080`（`PORT` で変更可）。`DATABASE_URL` 未設定時は compose 用のローカル Postgres URL がコード側の既定で使われます。レート制限を有効にするには `.env` に `REDIS_ADDR=127.0.0.1:6379`（など）を書き、ログに `redis rate limit: connected` が出ることを確認してください。

## フロント E2E（Playwright）

前提: 上記どおり **Postgres が起動**していること（`DATABASE_URL` は既定で `localhost:5432` の `chatapp` を参照）。

```bash
cd frontend
npm ci
npx playwright install chromium
npm run test:e2e
```

- テスト実行前に `go run ./cmd/seed e2e`（`global-setup` から自動）で `e2e-admin@chat.test` / `e2e-bob@chat.test` を用意します。終了時に再度シードして BAN 状態を戻します。
- API はホストの **:8080 ではなく :18080** で起動し（`E2E_API_PORT` で変更可）、既存の `go run .` とポートがぶつからないようにしています。
- `SKIP_E2E_SEED=1` を付けると DB シードのみスキップします（自分でユーザーを用意する場合）。

## k6（API 負荷）

リリース前・CI で **必ず**通す想定の **単一シナリオ**（`k6/scenarios/spec_api.js`）。`/health`・登録・グループ・`/me`・`/rooms`・メッセージ・既読をまとめて叩きます。DB シードは不要です。

1. Postgres（＋任意で Redis）と API を起動する（上記「起動方法」）。
2. リポジトリ**ルート**で [k6](https://grafana.com/docs/k6/latest/) を使う:

   **k6 本体がある場合**

   ```bash
   k6 run k6/scenarios/spec_api.js
   ```

   **Docker のみの場合**（API がホストの `8080` のとき。Mac の Docker Desktop では `K6_API_BASE_URL=http://host.docker.internal:8080` が必要なことがあります）

   ```bash
   docker run --rm -i --network=host \
     -e K6_API_BASE_URL="${K6_API_BASE_URL:-http://127.0.0.1:8080}" \
     -e K6_VUS="${K6_VUS:-4}" \
     -e K6_DURATION="${K6_DURATION:-35s}" \
     -v "$(pwd)/k6:/k6:ro" \
     grafana/k6:0.55.0 run /k6/scenarios/spec_api.js
   ```

環境変数の一覧は [k6/README.md](k6/README.md) に書いています。

## CI（GitHub Actions）

[.github/workflows/ci.yml](.github/workflows/ci.yml) で次を順に実行します。

- `go vet` とバックエンド単体テスト（短縮モード）
- Postgres 上の **integration** テスト（`//go:build integration` 含む）
- フロントの **lint / build**
- **Playwright E2E**（DB シード込み）
- **k6**（`spec_api.js` のみ）

ローカルで近いことをする場合は、Postgres を起動したうえで `backend` / `frontend` の各コマンドと、上記 **k6** のコマンドを実行してください。

詳細・トラブル時の確認は [backend/README.md](backend/README.md) を参照。
