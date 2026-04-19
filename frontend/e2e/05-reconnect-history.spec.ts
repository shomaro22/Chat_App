import { test, expect } from '@playwright/test';
import { E2E_ADMIN, E2E_BOB, loginAs } from './helpers';

/** 仕様 §11.5: 再接続後に HTTP で履歴再取得し表示が復元される */
test.describe('再接続後の履歴復元', () => {
  test('ページリロード後もメッセージが残る', async ({ page }) => {
    await loginAs(page, E2E_BOB.email, E2E_BOB.password);
    await page.getByTestId('sidebar-create-room').click();
    await page.getByRole('button', { name: '1対1' }).click();
    await page.getByTestId('room-modal-direct-peer').fill(E2E_ADMIN.email);
    await page.getByTestId('room-modal-submit').click();
    await page.waitForURL(/\/rooms\/[0-9a-f-]{36}/i, { timeout: 30_000 });

    const marker = `reconnect-${Date.now()}`;
    await page.getByTestId('chat-message-input').fill(marker);
    await page.getByTestId('chat-send-button').click();
    await expect(page.getByTestId('message-list').getByText(marker).first()).toBeVisible({ timeout: 15_000 });

    await page.reload();
    await expect(page.getByTestId('message-list').getByText(marker).first()).toBeVisible({ timeout: 20_000 });
  });
});
