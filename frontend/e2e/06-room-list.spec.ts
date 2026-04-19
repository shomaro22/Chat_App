import { test, expect } from '@playwright/test';
import { E2E_ADMIN, E2E_BOB, loginAs } from './helpers';

test.describe('ルーム一覧（サイドバー）', () => {
  test('DM 作成後、サイドバーにルームが並ぶ', async ({ page }) => {
    await loginAs(page, E2E_BOB.email, E2E_BOB.password);
    await expect(page.getByTestId('sidebar-room-list')).toBeVisible({ timeout: 15_000 });

    await page.getByTestId('sidebar-create-room').click();
    await page.getByRole('button', { name: '1対1' }).click();
    await page.getByTestId('room-modal-direct-peer').fill(E2E_ADMIN.email);
    await page.getByTestId('room-modal-submit').click();
    await page.waitForURL(/\/rooms\/[0-9a-f-]{36}/i, { timeout: 30_000 });
    const idMatch = page.url().match(/\/rooms\/([0-9a-f-]{36})/i);
    expect(idMatch).toBeTruthy();
    const roomId = idMatch![1];

    await page.goto('/rooms');
    await expect(page.getByTestId(`sidebar-room-link-${roomId}`)).toBeVisible({ timeout: 15_000 });
  });
});
