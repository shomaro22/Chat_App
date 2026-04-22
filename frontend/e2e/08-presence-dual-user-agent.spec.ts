import { test, expect } from '@playwright/test';
import { E2E_ADMIN, E2E_BOB, loginAs } from './helpers';

/**
 * 仕様の「PC とスマートフォン相当」を、同一ブラウザ内で User-Agent 異なる 2 コンテキストで近似する。
 */
test.describe('多端末プレゼンス（User-Agent 切り替え）', () => {
  test('モバイル UA とデスクトップ UA の 2 接続でオンライン維持→両閉じでオフライン', async ({ browser }) => {
    const adminCtx = await browser.newContext();
    const adminPage = await adminCtx.newPage();
    await loginAs(adminPage, E2E_ADMIN.email, E2E_ADMIN.password);

    const mobileBob = await browser.newContext({
      userAgent:
        'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
    });
    const pMobile = await mobileBob.newPage();
    await loginAs(pMobile, E2E_BOB.email, E2E_BOB.password);
    await pMobile.getByTestId('sidebar-create-room').click();
    await pMobile.getByRole('button', { name: '1対1' }).click();
    await pMobile.getByTestId('room-modal-direct-peer').fill(E2E_ADMIN.email);
    await pMobile.getByTestId('room-modal-submit').click();
    await pMobile.waitForURL(/\/rooms\/[0-9a-f-]{36}/i, { timeout: 30_000 });
    const roomUrl = pMobile.url();

    const state = await mobileBob.storageState();
    const desktopBob = await browser.newContext({ storageState: state });
    const pDesktop = await desktopBob.newPage();
    await pDesktop.goto(roomUrl);

    await adminPage.goto(roomUrl);
    const bobRow = adminPage.locator('.member-presence-row').filter({ hasText: /Bob/ });
    await expect(bobRow.getByText('オンライン')).toBeVisible({ timeout: 25_000 });

    await pDesktop.close();
    await desktopBob.close();
    await expect(bobRow.getByText('オンライン')).toBeVisible({ timeout: 15_000 });

    await pMobile.close();
    await mobileBob.close();

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
