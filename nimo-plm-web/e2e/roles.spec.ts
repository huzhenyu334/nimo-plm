import { test, expect } from '@playwright/test';

// Use a unique role name to avoid conflicts across test runs
const TEST_ROLE_NAME = 'E2E测试角色-' + Date.now().toString(36);
const TEST_ROLE_RENAMED = TEST_ROLE_NAME + '-已修改';

test.describe('Role Management', () => {
  // CRUD tests must run in order: create → edit → delete
  test.describe.configure({ mode: 'serial' });
  test('roles API returns 401 without auth header', async ({ request }) => {
    const response = await request.get('/api/v1/roles', {
      headers: { 'Authorization': 'Bearer invalid-token' },
    });
    expect(response.status()).toBe(401);
  });

  test('roles list page loads', async ({ page }) => {
    await page.goto('/roles');
    await page.waitForLoadState('networkidle');

    // Should show role management page with role list
    await expect(page.getByText('角色管理').first()).toBeVisible();
  });

  test('create role flow', async ({ page }) => {
    await page.goto('/roles');
    await page.waitForLoadState('networkidle');

    // Click "新增角色" button at the bottom of the role list
    await page.getByRole('button', { name: '新增角色' }).click();

    // Fill in role name in the modal
    await page.getByPlaceholder('如 项目经理').fill(TEST_ROLE_NAME);

    // Submit
    await page.getByRole('button', { name: '确 定' }).click();

    // Wait for modal to close and verify role appears
    await expect(page.getByText(TEST_ROLE_NAME).first()).toBeVisible({ timeout: 5000 });
  });

  test('edit role flow', async ({ page }) => {
    await page.goto('/roles');
    await page.waitForLoadState('networkidle');

    // Click on the role in the list to select it
    await page.getByText(TEST_ROLE_NAME).first().click();

    // Click the MoreOutlined (⋮) icon on the role item to open dropdown
    const roleItem = page.locator('div[style*="cursor: pointer"]', { hasText: TEST_ROLE_NAME }).first();
    await roleItem.locator('.ant-dropdown-trigger').click();

    // Click edit option in the dropdown menu
    await page.locator('.ant-dropdown-menu-item').getByText('编辑').click();

    // Update name
    await page.getByPlaceholder('如 项目经理').fill(TEST_ROLE_RENAMED);

    // Submit
    await page.getByRole('button', { name: '确 定' }).click();

    // Verify updated name
    await expect(page.getByText(TEST_ROLE_RENAMED).first()).toBeVisible({ timeout: 5000 });
  });

  test('delete role flow', async ({ page }) => {
    await page.goto('/roles');
    await page.waitForLoadState('networkidle');

    // Click the MoreOutlined icon on the role item to open dropdown
    const roleItem = page.locator('div[style*="cursor: pointer"]', { hasText: TEST_ROLE_RENAMED }).first();
    await roleItem.locator('.ant-dropdown-trigger').click();

    // Click delete option in the dropdown
    await page.locator('.ant-dropdown-menu-item').getByText('删除').click();

    // Wait for the confirm modal to appear, then click confirm
    const confirmModal = page.locator('.ant-modal-confirm');
    await expect(confirmModal).toBeVisible();
    await confirmModal.getByRole('button', { name: '确 定' }).click();

    // Wait for modal to close and verify role is removed
    await expect(confirmModal).not.toBeVisible({ timeout: 5000 });
    await expect(page.getByText(TEST_ROLE_RENAMED, { exact: true })).not.toBeVisible({ timeout: 5000 });
  });
});
