import type { Page } from '@playwright/test';

/** playwright.config.ts が process.env に入れる（既定 18080 でホストの :8080 と分離） */
export const E2E_API_ORIGIN = (process.env.E2E_API_ORIGIN ?? 'http://127.0.0.1:18080').replace(/\/$/, '');

export const E2E_ADMIN = { email: 'e2e-admin@chat.test', password: 'E2ePassw0rd!', name: 'E2E Admin' };
export const E2E_BOB = { email: 'e2e-bob@chat.test', password: 'E2ePassw0rd!', name: 'E2E Bob' };

/** 1x1 透明 PNG */
export const TINY_PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
  'base64',
);

export async function loginAs(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.getByLabel('メールアドレス').fill(email);
  await page.locator('#password').fill(password);
  await page.getByRole('button', { name: 'ログイン' }).click();
  await page.waitForURL(/\/rooms(\/|$)/, { timeout: 30_000 });
}

export async function registerAs(page: Page, name: string, email: string, password: string) {
  await page.goto('/register');
  await page.locator('#reg-name').fill(name);
  await page.locator('#reg-email').fill(email);
  await page.locator('#reg-password').fill(password);
  await page.getByRole('button', { name: '登録' }).click();
}
