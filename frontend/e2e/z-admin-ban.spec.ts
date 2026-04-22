import { test, expect } from '@playwright/test';
import { E2E_ADMIN, E2E_API_ORIGIN, E2E_BOB, loginAs } from './helpers';

test.describe('管理者: ユーザー一覧 → BAN → 再ログイン不可・トークン無効', () => {
  test('Bob を BAN 後、ログインと JWT の /me が拒否される', async ({ page, browser }) => {
    const bobCtx = await browser.newContext();
    const bobPage = await bobCtx.newPage();
    await loginAs(bobPage, E2E_BOB.email, E2E_BOB.password);
    const bobToken = await bobPage.evaluate(() => localStorage.getItem('chat_app_access_token'));
    expect(bobToken).toBeTruthy();

    await loginAs(page, E2E_ADMIN.email, E2E_ADMIN.password);
    await page.goto('/admin');
    await expect(page.getByRole('heading', { name: '管理者画面' })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText('このアカウントは管理者ではありません')).toHaveCount(0);
    await page.locator('table.admin-table').waitFor({ state: 'visible', timeout: 30_000 });
    await page.locator('table.admin-table tbody tr').first().waitFor({ state: 'visible', timeout: 15_000 });
    await page.getByPlaceholder('名前・メール・UUID の一部で検索').fill(E2E_BOB.email);
    await page.getByRole('button', { name: '検索' }).click();
    const bobRow = page.locator('tr').filter({ hasText: E2E_BOB.email }).first();
    await expect(bobRow).toBeVisible({ timeout: 20_000 });
    await bobRow.getByRole('button', { name: 'BAN' }).click();

    await bobCtx.close();

    const victimCtx = await browser.newContext();
    const victimPage = await victimCtx.newPage();
    await victimPage.goto('/login');
    await victimPage.getByLabel('メールアドレス').fill(E2E_BOB.email);
    await victimPage.locator('#password').fill(E2E_BOB.password);
    await victimPage.getByRole('button', { name: 'ログイン' }).click();
    await expect(victimPage.getByText(/無効化されています/)).toBeVisible({ timeout: 10_000 });
    await victimCtx.close();

    const res = await page.request.get(`${E2E_API_ORIGIN}/me`, {
      headers: { Authorization: `Bearer ${bobToken}` },
    });
    expect(res.status()).toBe(401);
  });
});
