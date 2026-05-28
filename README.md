# ポートフォリオサイト

## 概要

Next.js / TypeScript / Tailwind CSS で作成したシンプルなポートフォリオサイトです。
まずは静的サイトとして公開し、プロフィール、スキル、制作物、連絡先を見やすく整理する構成にしています。

## 技術スタック

### Frontend

- Next.js
- TypeScript
- Tailwind CSS

### Infra

- Vercel
- Docker / nginx（静的エクスポート確認用）

### CI/CD

- GitHub Actions

## ページ構成

- Hero: 名前、職種、概要
- About: 強み、開発で大切にしていること
- Skills: 使用技術
- Projects: 制作物一覧
- Contact: メール、GitHubリンク

## ローカル起動

```bash
cd frontend
npm ci
npm run dev
```

開発サーバー: `http://localhost:3000`

## 編集する主なファイル

- `frontend/src/app/page.tsx`: 表示内容、制作物、スキル
- `frontend/src/app/globals.css`: 全体スタイル
- `frontend/src/app/layout.tsx`: メタデータ

`frontend/src/app/page.tsx` の `profile`, `skills`, `projects`, `strengths` を自分の内容に差し替えると、ポートフォリオとして使えます。

## lint / build

```bash
cd frontend
npm run lint
npm run build
```

## デプロイ

Vercel では `frontend` ディレクトリをプロジェクトルートに設定します。

- Install Command: `npm ci`
- Build Command: `npm run build`
- Framework Preset: Next.js

## 今後の案2について

問い合わせ保存、ブログ管理、制作物の管理画面などが必要になったら、バックエンドを追加します。
GinではなくEchoで作る構成も相性が良く、既存のGoバックエンド資産を活かしやすいです。
