import { test, expect } from '@playwright/test';

test.use({ storageState: './e2e/.auth/storage-state.json' });

test.describe('Material Search', () => {
  test('page loads with search bar and filters', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1500);
    const searchInput = page.locator('input[placeholder*="搜索物料"]');
    await expect(searchInput.first()).toBeVisible();
    // Category select
    const selects = page.locator('.ant-select');
    expect(await selects.count()).toBeGreaterThanOrEqual(2);
  });

  test('category filter dropdown has options', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1500);
    // First select should be category
    const categorySelect = page.locator('.ant-select').first();
    await categorySelect.click();
    await page.waitForTimeout(300);
    const electronicOption = page.locator('.ant-select-item-option').filter({ hasText: '电子' });
    await expect(electronicOption).toBeVisible();
  });

  test('search input triggers query', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1500);
    const searchInput = page.locator('input[placeholder*="搜索物料"]');
    await searchInput.fill('test');
    await page.waitForTimeout(2000);
    // Table should be visible (even if empty)
    const table = page.locator('.ant-table');
    await expect(table).toBeVisible();
  });

  test('table renders with correct columns', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1500);
    const table = page.locator('.ant-table');
    await expect(table).toBeVisible();
    // Verify key column headers
    const headers = page.locator('.ant-table-thead th');
    const headerTexts = await headers.allTextContents();
    const joined = headerTexts.join(',');
    expect(joined).toContain('名称');
    expect(joined).toContain('分类');
    expect(joined).toContain('MPN');
    expect(joined).toContain('数量');
  });

  test('reset filters link appears when filter active', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1500);
    // Select a category
    const categorySelect = page.locator('.ant-select').first();
    await categorySelect.click();
    await page.waitForTimeout(300);
    const electronicOption = page.locator('.ant-select-item-option').filter({ hasText: '电子' });
    await electronicOption.click();
    await page.waitForTimeout(500);
    // Reset link should appear
    const resetLink = page.locator('text=重置');
    await expect(resetLink).toBeVisible();
  });

  test('pagination controls exist', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1500);
    const pagination = page.locator('.ant-pagination');
    await expect(pagination).toBeVisible();
  });
});

test.describe('Material Search API', () => {
  test('global search API returns valid response', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1000);
    const result = await page.evaluate(async () => {
      const token = localStorage.getItem('access_token');
      const resp = await fetch('/api/v1/bom-items/global?page=1&page_size=5', {
        headers: { Authorization: `Bearer ${token}` },
      });
      return { status: resp.status, body: await resp.json() };
    });
    expect(result.status).toBe(200);
    expect(result.body).toHaveProperty('data');
    expect(result.body).toHaveProperty('total');
  });

  test('global search API supports category filter', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1000);
    const result = await page.evaluate(async () => {
      const token = localStorage.getItem('access_token');
      const resp = await fetch('/api/v1/bom-items/global?category=electronic&page=1&page_size=5', {
        headers: { Authorization: `Bearer ${token}` },
      });
      return { status: resp.status, body: await resp.json() };
    });
    expect(result.status).toBe(200);
    expect(result.body).toHaveProperty('data');
  });

  test('global search API supports text search', async ({ page }) => {
    await page.goto('/material-search');
    await page.waitForTimeout(1000);
    const result = await page.evaluate(async () => {
      const token = localStorage.getItem('access_token');
      const resp = await fetch('/api/v1/bom-items/global?q=test&page=1&page_size=5', {
        headers: { Authorization: `Bearer ${token}` },
      });
      return { status: resp.status, body: await resp.json() };
    });
    expect(result.status).toBe(200);
    expect(result.body).toHaveProperty('data');
  });
});
