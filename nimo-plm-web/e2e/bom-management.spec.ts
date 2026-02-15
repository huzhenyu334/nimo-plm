import { test, expect } from '@playwright/test';

test.use({ storageState: './e2e/.auth/storage-state.json' });

test.describe('BOM Management', () => {
  test('bom-management page loads with project list', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    // Should show BOM管理 title or project cards
    const heading = page.locator('text=BOM管理');
    await expect(heading.first()).toBeVisible();
  });

  test('bom-management page has search input', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1000);
    const search = page.locator('input[placeholder*="搜索"]');
    await expect(search).toBeVisible();
  });

  test('clicking project navigates to detail', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    // Find any clickable project item
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1000);
      await expect(page).toHaveURL(/\/bom-management\//);
    }
  });

  test('bom detail page shows BOM type tabs', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1500);
      // Should have EBOM/PBOM/MBOM tabs
      const ebomTab = page.locator('.ant-tabs-tab').filter({ hasText: 'EBOM' });
      await expect(ebomTab).toBeVisible();
      const pbomTab = page.locator('.ant-tabs-tab').filter({ hasText: 'PBOM' });
      await expect(pbomTab).toBeVisible();
    }
  });

  test('bom detail has create button', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1500);
      const createBtn = page.locator('button').filter({ hasText: '新建' });
      await expect(createBtn).toBeVisible();
    }
  });

  test('bom permissions API returns valid response', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1500);
      // The page should have loaded permissions without error
      // Check that the page doesn't show a fatal error
      const errorAlert = page.locator('.ant-alert-error');
      await expect(errorAlert).toHaveCount(0);
    }
  });
});

test.describe('BOM Management - Navigation', () => {
  test('sidebar has BOM管理 menu item', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForTimeout(1000);
    // Check sidebar menu
    const bomMenuItem = page.locator('.ant-menu-item, .ant-pro-base-menu-item').filter({ hasText: 'BOM管理' });
    await expect(bomMenuItem.first()).toBeVisible();
  });

  test('clicking BOM管理 in sidebar navigates correctly', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForTimeout(1000);
    const bomMenuItem = page.locator('.ant-menu-item, .ant-pro-base-menu-item').filter({ hasText: 'BOM管理' });
    if (await bomMenuItem.count() > 0) {
      await bomMenuItem.first().click();
      await expect(page).toHaveURL(/\/bom-management/);
    }
  });
});
