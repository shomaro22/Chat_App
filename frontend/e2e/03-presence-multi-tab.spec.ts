import { test, expect } from '@playwright/test';
import { E2E_ADMIN, E2E_BOB, loginAs } from './helpers';

test.describe('多端末プレゼンス（同一ユーザー・複数タブ）', () => {
  test('Bob が2タブでルームにいる間はオンライン、両方閉じるとオフラインへ', async ({ browser }) => {
    const adminCtx = await browser.newContext();
    const adminPage = await adminCtx.newPage();
    await loginAs(adminPage, E2E_ADMIN.email, E2E_ADMIN.password);

    const bob1 = await browser.newContext();
    const p1 = await bob1.newPage();
    await loginAs(p1, E2E_BOB.email, E2E_BOB.password);
    await p1.getByTestId('sidebar-create-room').click();
    await p1.getByRole('button', { name: '1対1' }).click();
    await p1.getByTestId('room-modal-direct-peer').fill(E2E_ADMIN.email);
    await p1.getByTestId('room-modal-submit').click();
    await p1.waitForURL(/\/rooms\/[0-9a-f-]{36}/i, { timeout: 30_000 });
    const roomUrl = p1.url();

    const state = await bob1.storageState();
    const bob2 = await browser.newContext({ storageState: state });
    const p2 = await bob2.newPage();
    await p2.goto(roomUrl);

    await adminPage.goto(roomUrl);
    const bobRow = adminPage.locator('.member-presence-row').filter({ hasText: /Bob/ });
    await expect(bobRow.getByText('オンライン')).toBeVisible({ timeout: 25_000 });

    await p2.close();
    await bob2.close();
    await expect(bobRow.getByText('オンライン')).toBeVisible({ timeout: 15_000 });

    await p1.close();
    await bob1.close();

    await adminPage.reload();
    await expect
      .poll(
        async () => {
          const row = adminPage.locator('.member-presence-row').filter({ hasText: /Bob/ });
          return (await row.getByText('オフライン').count()) > 0;
        },
        { timeout: 25_000, intervals: [600, 1200, 2000] },
      )
      .toBe(true);

    await adminCtx.close();
  });
});
