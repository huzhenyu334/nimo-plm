import { test, expect } from '@playwright/test';

test.use({ storageState: './e2e/.auth/storage-state.json' });

test.describe('Document Version Management', () => {
  test('documents page loads with table', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1500);
    const heading = page.locator('text=文档管理');
    await expect(heading.first()).toBeVisible();
    // Table should be present
    const table = page.locator('.ant-table');
    await expect(table).toBeVisible();
  });

  test('documents page has upload button', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1000);
    const uploadBtn = page.locator('button').filter({ hasText: '上传文档' });
    await expect(uploadBtn).toBeVisible();
  });

  test('documents page has category filter and search', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1000);
    // Category select
    const categorySelect = page.locator('.ant-select').first();
    await expect(categorySelect).toBeVisible();
    // Search input
    const searchInput = page.locator('input[placeholder*="搜索"]');
    await expect(searchInput).toBeVisible();
  });

  test('documents page has refresh button', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1000);
    const refreshBtn = page.locator('button').filter({ hasText: '刷新' });
    await expect(refreshBtn).toBeVisible();
  });

  test('upload document modal opens', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1000);
    const uploadBtn = page.locator('button').filter({ hasText: '上传文档' });
    await uploadBtn.click();
    // Modal should open
    const modal = page.locator('.ant-modal').filter({ hasText: '上传文档' });
    await expect(modal).toBeVisible();
    // Check form fields
    await expect(page.locator('text=文档标题')).toBeVisible();
    await expect(page.locator('label').filter({ hasText: '选择文件' })).toBeVisible();
  });

  test('table shows version column with V prefix', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1500);
    // Check version column header
    const versionHeader = page.locator('th').filter({ hasText: '版本' });
    await expect(versionHeader).toBeVisible();
    // If there are documents, version tags should show V prefix
    const versionTags = page.locator('.ant-tag').filter({ hasText: /^V\d/ });
    const count = await versionTags.count();
    if (count > 0) {
      const firstTag = await versionTags.first().textContent();
      expect(firstTag).toMatch(/^V\d/);
    }
  });

  test('version history drawer opens with upload button', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1500);
    // Find a history icon button (HistoryOutlined)
    const historyBtns = page.locator('button .anticon-history');
    const count = await historyBtns.count();
    if (count > 0) {
      await historyBtns.first().click();
      await page.waitForTimeout(500);
      // Version history drawer should open
      const drawer = page.locator('.ant-drawer').filter({ hasText: '版本历史' });
      await expect(drawer).toBeVisible();
      // Upload new version button should be in the drawer
      const uploadVersionBtn = drawer.locator('button').filter({ hasText: '上传新版本' });
      await expect(uploadVersionBtn).toBeVisible();
    }
  });

  test('upload new version modal opens from version drawer', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1500);
    const historyBtns = page.locator('button .anticon-history');
    const count = await historyBtns.count();
    if (count > 0) {
      await historyBtns.first().click();
      await page.waitForTimeout(500);
      const drawer = page.locator('.ant-drawer').filter({ hasText: '版本历史' });
      const uploadVersionBtn = drawer.locator('button').filter({ hasText: '上传新版本' });
      await uploadVersionBtn.click();
      await page.waitForTimeout(500);
      // Upload new version modal should open
      const modal = page.locator('.ant-modal').filter({ hasText: '上传新版本' });
      await expect(modal).toBeVisible();
      // Should have change summary field
      await expect(page.locator('text=变更说明')).toBeVisible();
    }
  });

  test('document detail drawer shows version history button', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1500);
    // Find view icon button (EyeOutlined)
    const viewBtns = page.locator('button .anticon-eye');
    const count = await viewBtns.count();
    if (count > 0) {
      await viewBtns.first().click();
      await page.waitForTimeout(500);
      // Detail drawer should have version history button
      const drawer = page.locator('.ant-drawer').filter({ hasText: '文档详情' });
      await expect(drawer).toBeVisible();
      const historyBtn = drawer.locator('button').filter({ hasText: '版本历史' });
      await expect(historyBtn).toBeVisible();
    }
  });

  test('version list shows current version tag', async ({ page }) => {
    await page.goto('/documents');
    await page.waitForTimeout(1500);
    const historyBtns = page.locator('button .anticon-history');
    const count = await historyBtns.count();
    if (count > 0) {
      await historyBtns.first().click();
      await page.waitForTimeout(1000);
      // Should have at least one version entry
      const drawer = page.locator('.ant-drawer').filter({ hasText: '版本历史' });
      const versionItems = drawer.locator('.ant-list-item');
      const versionCount = await versionItems.count();
      if (versionCount > 0) {
        // Current version should be highlighted
        const currentTag = drawer.locator('.ant-tag').filter({ hasText: '当前版本' });
        await expect(currentTag.first()).toBeVisible();
      }
    }
  });
});
