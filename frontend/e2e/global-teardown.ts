import { execSync } from 'node:child_process';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../..');

/** BAN などで変わった状態を次回用に戻す */
export default function globalTeardown() {
  if (process.env.SKIP_E2E_SEED === '1') return;
  try {
    execSync('go run ./cmd/seed e2e', {
      cwd: path.join(repoRoot, 'backend'),
      stdio: 'inherit',
      env: { ...process.env },
    });
  } catch {
    // DB が落ちている場合などは無視（ローカルで compose を止めた直後など）
  }
}
