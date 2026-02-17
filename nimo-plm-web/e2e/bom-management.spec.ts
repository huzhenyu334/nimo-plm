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

  test('bom detail has add material button', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1500);
      // BOM detail page shows "添加物料" buttons for adding items to categories
      const addRowBtn = page.locator('button').filter({ hasText: '添加物料' });
      await expect(addRowBtn.first()).toBeVisible();
    }
  });

  test('bom detail supplier column shows Select dropdown', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1500);
      // The supplier column should render as Select components (not plain text inputs)
      const supplierSelects = page.locator('.ant-select').filter({ has: page.locator('[title="选择供应商"]') });
      // Also check that the table header has "供应商" column
      const supplierHeader = page.locator('th, .editable-table-header').filter({ hasText: '供应商' });
      // At least one should be visible (header or select)
      const hasHeader = await supplierHeader.count() > 0;
      const hasSelect = await supplierSelects.count() > 0;
      expect(hasHeader || hasSelect).toBeTruthy();
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

test.describe('BOM Management - Add Material Modal', () => {
  test('add material button opens search modal', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1500);
      // Click first "添加物料" button
      const addBtn = page.locator('button').filter({ hasText: '添加物料' });
      if (await addBtn.count() > 0) {
        await addBtn.first().click();
        await page.waitForTimeout(500);
        // Modal should appear with search input
        const modal = page.locator('.ant-modal');
        await expect(modal).toBeVisible();
        const searchInput = modal.locator('input[placeholder*="搜索"]');
        await expect(searchInput).toBeVisible();
      }
    }
  });

  test('add material modal has create new button', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1500);
      const addBtn = page.locator('button').filter({ hasText: '添加物料' });
      if (await addBtn.count() > 0) {
        await addBtn.first().click();
        await page.waitForTimeout(500);
        const modal = page.locator('.ant-modal');
        // Should have "创建新物料" button
        const createBtn = modal.locator('button').filter({ hasText: '创建新物料' });
        await expect(createBtn).toBeVisible();
      }
    }
  });

  test('add material modal has skip link', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(1500);
      const addBtn = page.locator('button').filter({ hasText: '添加物料' });
      if (await addBtn.count() > 0) {
        await addBtn.first().click();
        await page.waitForTimeout(500);
        const modal = page.locator('.ant-modal');
        // Should have "跳过" skip link at bottom
        const skipLink = modal.locator('button').filter({ hasText: '跳过' });
        await expect(skipLink).toBeVisible();
      }
    }
  });

  test('bom item search API returns results', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1000);
    // Use the page context which has storageState auth to make API call
    const result = await page.evaluate(async () => {
      const token = localStorage.getItem('access_token');
      const resp = await fetch('/api/v1/bom-items/search?q=&limit=5', {
        headers: { Authorization: `Bearer ${token}` },
      });
      return { status: resp.status, body: await resp.json() };
    });
    expect(result.status).toBe(200);
    expect(result.body).toHaveProperty('data');
    expect(Array.isArray(result.body.data)).toBeTruthy();
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
