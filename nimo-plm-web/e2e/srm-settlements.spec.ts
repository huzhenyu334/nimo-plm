import { test, expect } from '@playwright/test';

test.describe('SRM Settlements (对账结算)', () => {
  test('settlements page loads with table and toolbar', async ({ page }) => {
    await page.goto('/srm/settlements');
    await page.waitForURL('**/srm/settlements', { timeout: 10000 });

    // Card title in main content area
    await expect(page.getByRole('main').getByText('对账结算')).toBeVisible({ timeout: 10000 });

    // Toolbar buttons
    await expect(page.getByRole('button', { name: /新建对账单/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /自动生成/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /刷新/ })).toBeVisible();

    // Table should exist
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('supplier filter dropdown works', async ({ page }) => {
    await page.goto('/srm/settlements');
    await page.waitForURL('**/srm/settlements', { timeout: 10000 });

    // Find the supplier select within main content
    const supplierSelect = page.getByRole('main').locator('.ant-select').first();
    await expect(supplierSelect).toBeVisible({ timeout: 10000 });
    await supplierSelect.click();

    // Dropdown should appear
    await expect(page.locator('.ant-select-dropdown')).toBeVisible({ timeout: 5000 });
  });

  test('status filter dropdown works', async ({ page }) => {
    await page.goto('/srm/settlements');
    await page.waitForURL('**/srm/settlements', { timeout: 10000 });

    // Second select is status filter
    const statusSelects = page.getByRole('main').locator('.ant-select');
    await expect(statusSelects.nth(1)).toBeVisible({ timeout: 10000 });
    await statusSelects.nth(1).click();

    // Should show status options
    await expect(page.locator('.ant-select-dropdown')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.ant-select-item-option').filter({ hasText: '草稿' })).toBeVisible();
    await expect(page.locator('.ant-select-item-option').filter({ hasText: '已确认' })).toBeVisible();
    await expect(page.locator('.ant-select-item-option').filter({ hasText: '已开票' })).toBeVisible();
    await expect(page.locator('.ant-select-item-option').filter({ hasText: '已付款' })).toBeVisible();
  });

  test('create settlement modal opens and has form fields', async ({ page }) => {
    await page.goto('/srm/settlements');
    await page.waitForURL('**/srm/settlements', { timeout: 10000 });

    // Click create button
    await page.getByRole('button', { name: /新建对账单/ }).click();

    // Modal should appear
    const modal = page.locator('.ant-modal-content');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await expect(modal.locator('.ant-modal-title')).toContainText('新建对账单');

    // Form fields inside modal
    await expect(modal.getByText('对账周期')).toBeVisible();
    await expect(modal.getByText('备注')).toBeVisible();

    // Cancel via close icon
    await page.locator('.ant-modal-close').click();
    await expect(page.locator('.ant-modal')).not.toBeVisible({ timeout: 3000 });
  });

  test('auto-generate modal opens and has required fields', async ({ page }) => {
    await page.goto('/srm/settlements');
    await page.waitForURL('**/srm/settlements', { timeout: 10000 });

    // Click auto-generate button
    await page.getByRole('button', { name: /自动生成/ }).click();

    // Modal should appear
    const modal = page.locator('.ant-modal-content');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await expect(modal.locator('.ant-modal-title')).toContainText('自动生成对账单');

    // Form fields inside modal
    await expect(modal.getByText('对账周期')).toBeVisible();

    // Cancel via close icon
    await page.locator('.ant-modal-close').click();
  });

  test('table has correct column headers', async ({ page }) => {
    await page.goto('/srm/settlements');
    await page.waitForURL('**/srm/settlements', { timeout: 10000 });

    // Verify table column headers
    const headerTexts = ['对账单号', '供应商', '对账周期', 'PO金额', '扣款', '结算金额', '状态', '创建时间', '操作'];
    for (const header of headerTexts) {
      await expect(page.locator('.ant-table-thead').getByText(header)).toBeVisible({ timeout: 5000 });
    }
  });
});
