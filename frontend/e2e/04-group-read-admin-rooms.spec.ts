import { test, expect } from '@playwright/test';
import { E2E_ADMIN, E2E_BOB, loginAs } from './helpers';

test.describe.configure({ mode: 'serial' });

test.describe('グループ・既読人数・管理者ルーム一覧', () => {
  test('グループ作成→送信→相手で既読人数→管理者でルーム一覧', async ({ browser }) => {
    const bobCtx = await browser.newContext();
    const adminCtx = await browser.newContext();
    const bobPage = await bobCtx.newPage();
    const adminPage = await adminCtx.newPage();

    await loginAs(bobPage, E2E_BOB.email, E2E_BOB.password);
    await expect(bobPage.getByTestId('sidebar-room-list')).toBeVisible({ timeout: 15_000 });

    await bobPage.getByTestId('sidebar-create-room').click();
    await bobPage.getByRole('button', { name: 'グループ' }).click();
    await bobPage.getByPlaceholder('グループの表示名').fill('E2Eグループ既読');
    await bobPage.getByPlaceholder(/参加メンバー/).fill(E2E_ADMIN.email);
    await bobPage.getByTestId('room-modal-submit').click();
    await bobPage.waitForURL(/\/rooms\/[0-9a-f-]{36}/i, { timeout: 30_000 });
    const roomUrl = bobPage.url();

    await bobPage.getByTestId('chat-message-input').fill('group read probe');
    await bobPage.getByTestId('chat-send-button').click();
    await expect(bobPage.getByTestId('message-list').getByText('group read probe').first()).toBeVisible({ timeout: 15_000 });

    await loginAs(adminPage, E2E_ADMIN.email, E2E_ADMIN.password);
    await adminPage.goto(roomUrl);
    await expect(adminPage.getByTestId('message-list').getByText('group read probe').first()).toBeVisible({ timeout: 20_000 });
    await adminPage.locator('[data-testid="message-list"]').evaluate((el) => {
      el.scrollTop = el.scrollHeight;
    });
    await expect(adminPage.getByText(/既読.*人/).first()).toBeVisible({ timeout: 25_000 });

    await adminPage.goto('/admin');
    await expect(adminPage.getByRole('heading', { name: '管理者画面' })).toBeVisible({ timeout: 15_000 });
    await adminPage.getByTestId('admin-tab-rooms').click();
    await expect(adminPage.getByTestId('admin-room-table')).toBeVisible({ timeout: 20_000 });
    await expect(adminPage.getByTestId('admin-room-table').getByText('group', { exact: true }).first()).toBeVisible({ timeout: 15_000 });

    await bobCtx.close();
    await adminCtx.close();
  });
});
