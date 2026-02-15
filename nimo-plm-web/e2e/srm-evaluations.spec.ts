import { test, expect } from '@playwright/test';

test.describe('SRM Evaluations (供应商评价)', () => {
  test('evaluations page loads with table and toolbar', async ({ page }) => {
    await page.goto('/srm/evaluations');
    await page.waitForURL('**/srm/evaluations', { timeout: 10000 });

    // Card title in main content area
    await expect(page.getByRole('main').locator('.ant-card-head-title')).toContainText('供应商评价', { timeout: 10000 });

    // Toolbar buttons
    await expect(page.getByRole('button', { name: /新建评价/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /刷新/ })).toBeVisible();

    // Table should exist
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('supplier filter dropdown works', async ({ page }) => {
    await page.goto('/srm/evaluations');
    await page.waitForURL('**/srm/evaluations', { timeout: 10000 });

    // Find the supplier select within main content
    const supplierSelect = page.getByRole('main').locator('.ant-select').first();
    await expect(supplierSelect).toBeVisible({ timeout: 10000 });
    await supplierSelect.click();

    // Dropdown should appear
    await expect(page.locator('.ant-select-dropdown')).toBeVisible({ timeout: 5000 });
  });

  test('status filter dropdown shows correct options', async ({ page }) => {
    await page.goto('/srm/evaluations');
    await page.waitForURL('**/srm/evaluations', { timeout: 10000 });

    // Second select is status filter
    const statusSelects = page.getByRole('main').locator('.ant-select');
    await expect(statusSelects.nth(1)).toBeVisible({ timeout: 10000 });
    await statusSelects.nth(1).click();

    // Should show status options
    await expect(page.locator('.ant-select-dropdown')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.ant-select-item-option').filter({ hasText: '草稿' })).toBeVisible();
    await expect(page.locator('.ant-select-item-option').filter({ hasText: '已提交' })).toBeVisible();
    await expect(page.locator('.ant-select-item-option').filter({ hasText: '已审批' })).toBeVisible();
  });

  test('create evaluation modal opens and has all form fields', async ({ page }) => {
    await page.goto('/srm/evaluations');
    await page.waitForURL('**/srm/evaluations', { timeout: 10000 });

    // Click create button
    await page.getByRole('button', { name: /新建评价/ }).click();

    // Modal should appear
    const modal = page.locator('.ant-modal-content');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await expect(modal.locator('.ant-modal-title')).toContainText('新建供应商评价');

    // Form fields inside modal
    await expect(modal.getByText('评价周期')).toBeVisible();
    await expect(modal.getByText('评价类型')).toBeVisible();
    await expect(modal.getByText('质量评分')).toBeVisible();
    await expect(modal.getByText('交期评分')).toBeVisible();
    await expect(modal.getByText('价格竞争力')).toBeVisible();
    await expect(modal.getByText('服务响应')).toBeVisible();
    await expect(modal.getByText('评价备注')).toBeVisible();

    // Cancel via close icon
    await page.locator('.ant-modal-close').click();
    await expect(page.locator('.ant-modal')).not.toBeVisible({ timeout: 3000 });
  });

  test('table has correct column headers', async ({ page }) => {
    await page.goto('/srm/evaluations');
    await page.waitForURL('**/srm/evaluations', { timeout: 10000 });

    // Verify table column headers
    const headerTexts = ['供应商', '评估周期', '类型', '质量', '交期', '价格', '服务', '综合评分', '等级', '状态', '操作'];
    for (const header of headerTexts) {
      await expect(page.locator('.ant-table-thead').getByText(header)).toBeVisible({ timeout: 5000 });
    }
  });

  test('evaluation type select defaults to quarterly', async ({ page }) => {
    await page.goto('/srm/evaluations');
    await page.waitForURL('**/srm/evaluations', { timeout: 10000 });

    // Open create modal
    await page.getByRole('button', { name: /新建评价/ }).click();
    const modal = page.locator('.ant-modal-content');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // The eval_type field should default to '季度评估'
    await expect(modal.locator('.ant-select-selection-item').filter({ hasText: '季度评估' })).toBeVisible();
  });
});
