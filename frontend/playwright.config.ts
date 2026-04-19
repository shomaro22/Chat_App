import { defineConfig, devices } from '@playwright/test';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '..');

const databaseURL =
  process.env.DATABASE_URL ?? 'postgres://postgres:postgres@127.0.0.1:5432/chatapp?sslmode=disable';

/** 開発中の :8080 と衝突しないよう E2E 専用ポート */
const e2eApiPort = process.env.E2E_API_PORT ?? '18080';
const e2eApiOrigin = `http://127.0.0.1:${e2eApiPort}`;
process.env.E2E_API_ORIGIN = e2eApiOrigin;

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? 'github' : 'list',
  globalSetup: path.join(__dirname, 'e2e/global-setup.ts'),
  globalTeardown: path.join(__dirname, 'e2e/global-teardown.ts'),
  use: {
    baseURL: 'http://127.0.0.1:5173',
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      command: 'go run .',
      cwd: path.join(repoRoot, 'backend'),
      env: {
        ...process.env,
        DATABASE_URL: databaseURL,
        JWT_SECRET: 'e2e-test-jwt-secret-32-bytes-min!!',
        PORT: e2eApiPort,
        REDIS_ADDR: '',
      },
      url: `${e2eApiOrigin}/health`,
      reuseExistingServer: !process.env.CI,
      timeout: 180_000,
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 5173',
      cwd: __dirname,
      env: {
        ...process.env,
        VITE_API_ORIGIN: e2eApiOrigin,
      },
      url: 'http://127.0.0.1:5173',
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
});
