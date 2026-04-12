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

   アプリを **別コンテナ** で動かし同一 compose ネットワークに載せる場合は、`REDIS_ADDR=redis:6379` のようにサービス名を使う。

3. **サーバー起動**

   このディレクトリをカレントにして `go run .` すると、**`.env` が存在すれば起動時に読み込みます**（未設定の変数のみ。既にエクスポート済みの値はそのまま）。

   ```bash
   go run .
   ```

### 動作確認の目安

- マイグレーションは起動時に自動実行
- Redis に接続できればログに `redis rate limit: connected (<REDIS_ADDR>)` が出る。`.env` に `REDIS_ADDR` がなく空のままなら `redis rate limit: disabled` となる（`.env` を置いたのに disabled のときは **`backend` で `go run .` しているか**、`cat .env` で `REDIS_ADDR` があるかを確認）
- Redis の生存確認: `redis-cli -h 127.0.0.1 -p 6379 ping` → `PONG`
