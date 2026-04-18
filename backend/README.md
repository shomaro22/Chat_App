# Backend（chat-app）

## 起動方法

### 前提

- Go（`go.mod` のバージョン）
- Docker / Docker Compose（Postgres・Redis）

### 手順

1. **コンテナ起動**（このディレクトリで実行）

   ```bash
   docker compose up -d
   ```

   | サービス   | ホストからの接続例 |
   | ---------- | ------------------ |
   | Postgres   | `localhost:5432`（DB: `chatapp`, ユーザー/パス: `postgres`） |
   | Redis      | `localhost:6379` |

2. **`.env`**

   ```bash
   cp .env.example .env
   ```

   | 変数 | 説明 |
   | ---- | ---- |
   | `DATABASE_URL` | Postgres 接続文字列 |
   | `REDIS_ADDR` | 例: `localhost:6379`。**空**のときは Redis レート制限を使わない |
   | `REDIS_PASSWORD` | 省略可 |
   | `REDIS_DB` | 省略時は `0` |
   | `JWT_SECRET` | 32 文字未満だと警告のうえ開発用既定値を使う |
   | `PORT` | 省略時 `8080` |
   | `CORS_ALLOW_ORIGINS` | 省略可。`DATABASE_URL` が `localhost` / `127.0.0.1` を含むとき、未設定なら Vite 用（5173 / 4173 の localhost と 127.0.0.1）を自動許可する。本番や上書き時はカンマ区切りで明示 |
   | `CORS_DISABLE` | `1` のとき上記の自動 CORS も含め CORS ミドルウェアを付けない |

   アプリを **別コンテナ** で動かし同一 compose ネットワークに載せる場合は、`REDIS_ADDR=redis:6379` のようにサービス名を使う。

3. **サーバー起動**

   このディレクトリをカレントにして `go run .` すると、**`.env` が存在すれば起動時に読み込みます**（未設定の変数のみ。既にエクスポート済みの値はそのまま）。

   ```bash
   go run .
   ```

### ローカル用テストアカウント

Postgres が起動した状態で `go run ./cmd/seedtestuser` を実行する。同じメールが既にいればスキップされ、パスワードだけ既定に戻したいときは `-reset`。

```bash
go run ./cmd/seedtestuser                 # 1件だけ（既定 test@example.com）
go run ./cmd/seedtestuser -team          # DM 用に複数（test + alice + bob）
go run ./cmd/seedtestuser -team -reset   # 上記3件すべてパスワード等を上書き
```

`-team` で作るアカウント（表示名は括弧内）。**パスワードは全員共通**（既定 `password123`、環境変数 `TEST_USER_PASSWORD` で変更可）。

| メール | 表示名 |
| --- | --- |
| `test@example.com` | Test（`TEST_USER_NAME` で変更可） |
| `alice@example.com` | Alice |
| `bob@example.com` | Bob |

1件だけのときは第1行のみ。`TEST_USER_EMAIL` で第1アカウントのメールを変えた場合、`alice` / `bob` はそのまま追加される（メールの重複は除く）。

メール・パスワード・表示名（第1件のみ）は `TEST_USER_EMAIL` / `TEST_USER_PASSWORD` / `TEST_USER_NAME` で上書きできます（パスワードは8文字以上）。

### フロント（Vite）でログイン試す

`frontend` は既定で `VITE_API_ORIGIN=http://127.0.0.1:8080`（`.env.development`）により API と別ポートになる。**`DATABASE_URL` がローカル向けのときは CORS を自動で有効**にするので、まずは `CORS_ALLOW_ORIGINS` なしでよい。

手順の例: `docker compose up -d` → `go run ./cmd/seedtestuser -team` → backend `go run .` → `frontend` で `npm run dev` → 別ブラウザ／シークレットで `alice` と `bob` にログインして DM を試す、など。

### 動作確認の目安

- マイグレーションは起動時に自動実行
- Redis に接続できればログに `redis rate limit: connected (<REDIS_ADDR>)` が出る。`.env` に `REDIS_ADDR` がなく空のままなら `redis rate limit: disabled` となる（`.env` を置いたのに disabled のときは **`backend` で `go run .` しているか**、`cat .env` で `REDIS_ADDR` があるかを確認）
- Redis の生存確認: `redis-cli -h 127.0.0.1 -p 6379 ping` → `PONG`
