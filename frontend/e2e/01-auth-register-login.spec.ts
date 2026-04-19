import { test, expect } from '@playwright/test';
import { loginAs, registerAs } from './helpers';

test.describe('新規登録 → ログイン', () => {
  test('ユニークメールで登録後、同じ資格情報でログインできる', async ({ page }) => {
    const email = `e2e-fresh-${Date.now()}@chat.test`;
    const password = 'E2ePassw0rd!';
    await registerAs(page, 'Fresh User', email, password);
    await page.waitForURL(/\/rooms/, { timeout: 30_000 });
    await page.evaluate(() => localStorage.removeItem('chat_app_access_token'));
    await loginAs(page, email, password);
    await expect(page.getByTestId('sidebar-create-room')).toBeVisible();
  });
});
