# リアルタイムチャットアプリ

## 概要

本アプリケーションは、1対1チャット / グループチャット / 既読 / リアクション / ルーム管理を一貫して扱うリアルタイムチャットシステムです。

主な特徴:

- 1対1ルーム・グループルームの作成
- WebSocket によるリアルタイムメッセージ配信
- 既読管理（既読更新・既読数表示）
- リアクション・ブックマーク・ピン留め
- 管理者機能（凍結・強制退出など）
- Rate Limit / 権限分離によるセキュリティ対策

## 想定ユーザー

- 一般ユーザー（チャット利用）
- 管理者（ユーザー・ルームの管理）

## 主なフロー

1. ユーザーがログイン
2. ルームに参加 / 作成
3. メッセージ送信（画像・ファイル添付含む）
4. 既読・リアクション・ブックマークを操作
5. 必要に応じて管理者がルームやユーザーを制御

## 技術スタック

### Backend

- Go (Echo)
- PostgreSQL
- Redis
- GORM

### Frontend

- React
- TypeScript
- Vite

### その他

- Docker / Docker Compose
- Playwright（E2E）
- k6（負荷試験）
- GitHub Actions（CI）

## アーキテクチャ

バックエンドはレイヤ分離を意識した構成です。

- `entity`
- `usecase`
- `repository`
- `controller`

## デプロイ先

- Frontend (Vercel): `https://chat-app-xi-ashen-92.vercel.app`
- Backend API (Render): `https://chat-app-zr9i.onrender.com`
- Health Check: `https://chat-app-zr9i.onrender.com/health`

補足:

- Vercel の環境変数: `VITE_API_ORIGIN=https://chat-app-zr9i.onrender.com`
- Render の環境変数: `CORS_ALLOW_ORIGINS=https://chat-app-xi-ashen-92.vercel.app`

## 起動（全体）

```bash
docker compose up -d
```

DB・Redis・バックエンド・フロントエンドを一括起動します。
（コンテナビルドはルート直下の `Dockerfile` を共通利用）

- フロントエンド: `http://localhost`
- バックエンド API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

## バックエンド起動（個別）

```bash
docker compose up -d postgres redis
cd backend
cp .env.example .env
go run .
```

主な環境変数（`backend/.env.example`）:

- `DATABASE_URL`: `postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable`
- `REDIS_ADDR`: `localhost:6379`（空にするとRedisレート制限は無効）
- `JWT_SECRET`: 任意

## フロントエンド起動（個別）

```bash
cd frontend
npm ci
npm run dev
```

開発サーバー: `http://localhost:5173`

## テストアカウント作成（seed）

`backend` の seed コマンドでテストアカウントを作成できます。

### E2E用（Playwright）

```bash
cd backend
go run ./cmd/seed e2e
```

作成（または更新）されるアカウント:

- 管理者: `e2e-admin@chat.test`
- 一般ユーザー: `e2e-bob@chat.test`
- パスワード（共通）: `E2ePassw0rd!`

### ローカル開発用（1件）

```bash
cd backend
go run ./cmd/seed dev
```

既定値:

- メール: `test@example.com`
- 名前: `Test`
- パスワード: `password123`

### ローカル開発用（複数）

```bash
cd backend
go run ./cmd/seed dev -team
```

追加されるユーザー:

- `alice@example.com`（user）
- `bob@example.com`（user）
- `admin@example.com`（admin）

`-reset` を付けると既存ユーザーの名前・ロール・パスワードを更新します。

## テスト

以下のテストを実施可能です。

- 静的解析（frontend lint / `go vet`）
- 単体・結合テスト（Go）
- E2Eテスト（Playwright）
- API負荷試験（k6）

### Backend テスト

```bash
cd backend
go test ./...
go test -tags=integration ./repository ./usecase ./e2e -count=1 -timeout 20m
```

### Frontend lint / build

```bash
cd frontend
npm ci
npm run lint
npm run build
```

### E2E（Playwright）

```bash
cd frontend
npx playwright install chromium
npm run test:e2e
```

補足:

- `global-setup` から `go run ./cmd/seed e2e` が実行され、E2E用データを投入します。
- E2EのAPIポートは既定で `:18080`（通常の `:8080` と衝突回避）。

### k6

ローカルに `k6` バイナリが無い場合は、以下の Docker コマンドで実行します。

```bash
docker run --rm -i --network=host \
  -e K6_API_BASE_URL="${K6_API_BASE_URL:-http://127.0.0.1:8080}" \
  -e K6_VUS="${K6_VUS:-4}" \
  -e K6_DURATION="${K6_DURATION:-35s}" \
  -v "$(pwd)/k6:/k6:ro" \
  grafana/k6:0.55.0 run /k6/scenarios/spec_api.js
```

## CI（GitHub Actions）

`.github/workflows/ci.yml` で主に以下を実行します。

- backend: `go vet` / test
- frontend: lint / build
- Playwright E2E
- k6 シナリオ

補足:

- Integration テストは `repository` / `usecase` / `e2e` を分離し、各パッケージで専用DBを使って干渉を回避しています。

## API / 仕様

- OpenAPI定義: [openapi.yaml](openapi.yaml)

## 提出物としての補足

本リポジトリは実運用ではなく成果物提出を目的に整備しています。  
そのため、運用監視よりも次を重視しています。

- 再現可能な起動手順
- 再現可能なテスト手順
- 仕様と実装の整合

## 注意点

- プロトコル文字列（例: `SCHEMA_VERSION_MISMATCH`）やJSONキー（例: `my_last_read_message_id`）は互換性のため固定しています。
- コマンドは実行ディレクトリ（ルート / `backend` / `frontend`）を正しく選んで実行してください。
