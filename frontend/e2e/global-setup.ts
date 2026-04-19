import { execSync } from 'node:child_process';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../..');

/**
 * Postgres が起動している前提で E2E 用ユーザーを投入する。
 * DB が無い場合は失敗するので、先に `docker compose up -d postgres redis` を実行してください。
 */
export default function globalSetup() {
  if (process.env.SKIP_E2E_SEED === '1') {
    console.warn('[e2e] SKIP_E2E_SEED=1 — `go run ./cmd/seed e2e` をスキップしました');
    return;
  }
  execSync('go run ./cmd/seed e2e', {
    cwd: path.join(repoRoot, 'backend'),
    stdio: 'inherit',
    env: { ...process.env },
  });
}
