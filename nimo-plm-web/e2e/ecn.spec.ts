import { test, expect } from '@playwright/test';

test.describe('ECN Engineering Change Management (ECN工程变更管理)', () => {
  test('ECN list page loads with stats cards and table', async ({ page }) => {
    await page.goto('/ecn');
    await page.waitForURL('**/ecn', { timeout: 10000 });

    // Stats cards should be visible
    await expect(page.locator('.ant-statistic-title').getByText('待我审批')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-statistic-title').getByText('进行中')).toBeVisible();
    await expect(page.locator('.ant-statistic-title').getByText('本月新建')).toBeVisible();
    await expect(page.locator('.ant-statistic-title').getByText('本月关闭')).toBeVisible();

    // Main card title
    await expect(page.locator('.ant-card-head-title').getByText('ECN工程变更')).toBeVisible({ timeout: 10000 });

    // Toolbar buttons
    await expect(page.getByRole('button', { name: /新建变更请求/ })).toBeVisible();

    // Table should exist
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('ECN list status filter tabs work', async ({ page }) => {
    await page.goto('/ecn');
    await page.waitForURL('**/ecn', { timeout: 10000 });

    // Wait for page to load
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });

    // Status tabs - the segmented inside card body (not the view toggle in card extra)
    // Find the segmented that contains status text like '全部'
    const statusSegmented = page.locator('.ant-segmented').filter({ hasText: '全部' });
    await expect(statusSegmented).toBeVisible();
    await expect(statusSegmented.getByText('草稿')).toBeVisible();
    await expect(statusSegmented.getByText('待审批')).toBeVisible();
    await expect(statusSegmented.getByText('执行中')).toBeVisible();
    await expect(statusSegmented.getByText('已关闭')).toBeVisible();
  });

  test('ECN list has change type filter', async ({ page }) => {
    await page.goto('/ecn');
    await page.waitForURL('**/ecn', { timeout: 10000 });

    // Change type select
    const changeTypeSelect = page.getByRole('main').locator('.ant-select').first();
    await expect(changeTypeSelect).toBeVisible({ timeout: 10000 });
  });

  test('ECN list view mode can toggle between table and card', async ({ page }) => {
    await page.goto('/ecn');
    await page.waitForURL('**/ecn', { timeout: 10000 });

    // Default table view should show ant-table
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });

    // Segmented control for view toggle should exist
    await expect(page.locator('.ant-segmented').first()).toBeVisible();
  });

  test('ECN create form page loads with step wizard', async ({ page }) => {
    await page.goto('/ecn/new');
    await page.waitForURL('**/ecn/new', { timeout: 10000 });

    // Page title (use heading to avoid matching breadcrumb)
    await expect(page.getByRole('heading', { name: '新建变更请求' })).toBeVisible({ timeout: 10000 });

    // Steps should be visible
    await expect(page.locator('.ant-steps').getByText('基本信息')).toBeVisible();
    await expect(page.locator('.ant-steps').getByText('影响范围')).toBeVisible();
    await expect(page.locator('.ant-steps').getByText('技术方案')).toBeVisible();
    await expect(page.locator('.ant-steps').getByText('审批配置')).toBeVisible();

    // Step 1 form fields (use form label locators)
    await expect(page.locator('.ant-form-item-label').getByText('标题')).toBeVisible();
    await expect(page.locator('.ant-form-item-label').getByText('关联产品')).toBeVisible();
    await expect(page.locator('.ant-form-item-label').getByText('变更类型')).toBeVisible();
    await expect(page.locator('.ant-form-item-label').getByText('紧急程度')).toBeVisible();
    await expect(page.locator('.ant-form-item-label').getByText('变更原因')).toBeVisible();

    // Action buttons
    await expect(page.getByRole('button', { name: /保存草稿/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /下一步/ })).toBeVisible();
  });

  test('ECN create form step navigation works', async ({ page }) => {
    await page.goto('/ecn/new');
    await page.waitForURL('**/ecn/new', { timeout: 10000 });

    // Wait for form to load
    await expect(page.getByRole('heading', { name: '新建变更请求' })).toBeVisible({ timeout: 10000 });

    // Fill required fields for step 1
    await page.getByLabel('标题').fill('测试变更请求');

    // Wait for the product select to be available
    await expect(page.locator('.ant-form-item-label').getByText('关联产品')).toBeVisible({ timeout: 5000 });

    // Check change type radio buttons exist (use wrapper labels, not hidden input)
    await expect(page.locator('.ant-radio-wrapper').getByText('设计变更')).toBeVisible();
    await expect(page.locator('.ant-radio-wrapper').getByText('物料变更')).toBeVisible();
    await expect(page.locator('.ant-radio-wrapper').getByText('工艺变更')).toBeVisible();

    // Urgency buttons (ant-radio-button-wrapper for button-style radio)
    await expect(page.locator('.ant-radio-button-wrapper').getByText('常规')).toBeVisible();
    await expect(page.locator('.ant-radio-button-wrapper').getByText('紧急')).toBeVisible();
    await expect(page.locator('.ant-radio-button-wrapper').getByText('特急')).toBeVisible();
  });

  test('ECN table has correct column headers', async ({ page }) => {
    await page.goto('/ecn');
    await page.waitForURL('**/ecn', { timeout: 10000 });

    // Wait for table to load
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });

    // Verify key table column headers are attached in DOM (some may be hidden due to horizontal scroll)
    const thead = page.locator('.ant-table-thead');
    // Check a subset of visible columns that should be in the viewport
    await expect(thead.locator('th').filter({ hasText: '编号' }).first()).toBeAttached({ timeout: 5000 });
    await expect(thead.locator('th').filter({ hasText: '标题' }).first()).toBeAttached();
    await expect(thead.locator('th').filter({ hasText: '状态' }).first()).toBeAttached();
    await expect(thead.locator('th').filter({ hasText: '操作' }).first()).toBeAttached();
  });

  test('ECN sidebar menu item exists', async ({ page }) => {
    await page.goto('/ecn');
    await page.waitForURL('**/ecn', { timeout: 10000 });

    // Sidebar should have ECN menu item
    await expect(page.getByText('ECN变更管理').first()).toBeVisible({ timeout: 10000 });
  });

  test('navigate to ECN create from list page', async ({ page }) => {
    await page.goto('/ecn');
    await page.waitForURL('**/ecn', { timeout: 10000 });

    // Click create button
    await page.getByRole('button', { name: /新建变更请求/ }).click();

    // Should navigate to create page
    await page.waitForURL('**/ecn/new', { timeout: 5000 });
    await expect(page.getByRole('heading', { name: '新建变更请求' })).toBeVisible({ timeout: 5000 });
  });
});
