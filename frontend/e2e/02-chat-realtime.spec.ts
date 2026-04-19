import { test, expect } from '@playwright/test';
import { E2E_ADMIN, E2E_BOB, TINY_PNG, loginAs } from './helpers';

test.describe.configure({ mode: 'serial' });

test.describe('チャット: ルーム作成・送信・相手リアルタイム・画像・削除・既読・再接続', () => {
  test('シード済み Bob / Admin で DM フロー', async ({ browser }) => {
    const bob = await browser.newContext();
    const admin = await browser.newContext();
    const bobPage = await bob.newPage();
    const adminPage = await admin.newPage();

    await loginAs(bobPage, E2E_BOB.email, E2E_BOB.password);
    await bobPage.getByTestId('sidebar-create-room').click();
    await bobPage.getByRole('button', { name: '1対1' }).click();
    await bobPage.getByTestId('room-modal-direct-peer').fill(E2E_ADMIN.email);
    await bobPage.getByTestId('room-modal-submit').click();
    await bobPage.waitForURL(/\/rooms\/[0-9a-f-]{36}/i, { timeout: 30_000 });
    const roomUrl = bobPage.url();

    await bobPage.getByTestId('chat-message-input').fill('hello e2e');
    await bobPage.getByTestId('chat-send-button').click();
    await expect(bobPage.getByTestId('message-list').getByText('hello e2e').first()).toBeVisible({ timeout: 15_000 });

    await loginAs(adminPage, E2E_ADMIN.email, E2E_ADMIN.password);
    await adminPage.goto(roomUrl);
    await expect(adminPage.getByTestId('message-list').getByText('hello e2e').first()).toBeVisible({ timeout: 20_000 });

    await bobPage.getByTestId('chat-message-input').fill('img probe');
    const fileInput = bobPage.locator('label[aria-label="画像を送信"] input[type="file"]');
    await fileInput.setInputFiles({ name: 't.png', mimeType: 'image/png', buffer: TINY_PNG });
    await expect(bobPage.locator('.message-row img[alt="添付画像"]').first()).toBeVisible({ timeout: 25_000 });

    await bobPage.once('dialog', (d) => d.accept());
    const helloRow = bobPage.locator('.message-row').filter({ hasText: 'hello e2e' }).first();
    await helloRow.getByRole('button', { name: '削除' }).click();
    await expect(bobPage.getByText('このメッセージは削除されました').first()).toBeVisible({ timeout: 10_000 });

    await adminPage.getByTestId('chat-message-input').fill('from admin read-receipt');
    await adminPage.getByTestId('chat-send-button').click();
    await expect(bobPage.getByText('from admin read-receipt').first()).toBeVisible({ timeout: 20_000 });
    await bobPage.locator('[data-testid="message-list"]').evaluate((el) => {
      el.scrollTop = el.scrollHeight;
    });
    await expect
      .poll(
        async () => {
          const t = await adminPage.content();
          return t.includes('既読');
        },
        { timeout: 25_000, intervals: [400, 800, 1200] },
      )
      .toBe(true);

    await bobPage.reload();
    await expect(bobPage.getByText('from admin read-receipt').first()).toBeVisible({ timeout: 15_000 });

    await bob.close();
    await admin.close();
  });
});
