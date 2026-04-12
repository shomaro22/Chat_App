# Chat_App

## 起動方法（バックエンド）

1. **依存**: [Go](https://go.dev/dl/)（`backend/go.mod` のバージョン）、[Docker](https://docs.docker.com/get-docker/)（Postgres / Redis 用）

2. **バックエンドディレクトリへ移動**（以降の手順は `backend` で実行）

   ```bash
   cd backend
   ```

3. **DB と Redis を起動**

   ```bash
   docker compose up -d
   ```

   Postgres（`localhost:5432`）と Redis（`localhost:6379`）が立ち上がります。

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

詳細・トラブル時の確認は [backend/README.md](backend/README.md) を参照。
