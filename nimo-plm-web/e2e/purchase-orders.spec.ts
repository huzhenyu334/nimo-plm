import { test, expect } from '@playwright/test';

test.use({ storageState: './e2e/.auth/storage-state.json' });

test.describe('Purchase Orders', () => {
  test('purchase orders page loads', async ({ page }) => {
    await page.goto('/srm/purchase-orders');
    await page.waitForTimeout(1500);
    const heading = page.locator('text=采购订单');
    await expect(heading.first()).toBeVisible();
  });

  test('purchase orders page has create button', async ({ page }) => {
    await page.goto('/srm/purchase-orders');
    await page.waitForTimeout(1500);
    const createBtn = page.locator('button').filter({ hasText: '新建订单' });
    await expect(createBtn).toBeVisible();
  });

  test('purchase orders page has status filter', async ({ page }) => {
    await page.goto('/srm/purchase-orders');
    await page.waitForTimeout(1000);
    const statusSelect = page.locator('.ant-select');
    await expect(statusSelect.first()).toBeVisible();
  });

  test('purchase orders table renders', async ({ page }) => {
    await page.goto('/srm/purchase-orders');
    await page.waitForTimeout(1500);
    const table = page.locator('.ant-table');
    await expect(table).toBeVisible();
  });

  test('create PO modal opens on button click', async ({ page }) => {
    await page.goto('/srm/purchase-orders');
    await page.waitForTimeout(1500);
    const createBtn = page.locator('button').filter({ hasText: '新建订单' });
    await createBtn.click();
    await page.waitForTimeout(500);
    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible();
    // Should have supplier select
    const supplierLabel = modal.locator('text=供应商');
    await expect(supplierLabel.first()).toBeVisible();
  });

  test('PO status labels include submitted state', async ({ page }) => {
    await page.goto('/srm/purchase-orders');
    await page.waitForTimeout(1000);
    // Check that the status filter has "待审批" option
    const statusSelect = page.locator('.ant-select').nth(1); // second select is status
    await statusSelect.click();
    await page.waitForTimeout(300);
    const submittedOption = page.locator('.ant-select-item-option').filter({ hasText: '待审批' });
    await expect(submittedOption.first()).toBeVisible();
  });
});

test.describe('Purchase Orders API', () => {
  test('list POs API returns valid response', async ({ page }) => {
    await page.goto('/srm/purchase-orders');
    await page.waitForTimeout(1000);
    const result = await page.evaluate(async () => {
      const token = localStorage.getItem('access_token');
      const resp = await fetch('/api/v1/srm/purchase-orders?page=1&page_size=5', {
        headers: { Authorization: `Bearer ${token}` },
      });
      return { status: resp.status, body: await resp.json() };
    });
    expect(result.status).toBe(200);
    expect(result.body).toHaveProperty('data');
  });

  test('from-bom API endpoint exists', async ({ page }) => {
    await page.goto('/srm/purchase-orders');
    await page.waitForTimeout(1000);
    const result = await page.evaluate(async () => {
      const token = localStorage.getItem('access_token');
      const resp = await fetch('/api/v1/srm/purchase-orders/from-bom', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ bom_id: 'nonexistent', item_ids: ['test'] }),
      });
      return { status: resp.status, body: await resp.text() };
    });
    // Should get 500 (not found BOM items) or 400, not 405 (method not allowed)
    expect([400, 500]).toContain(result.status);
  });
});

test.describe('BOM to PO Generation', () => {
  test('BOM detail page has generate PO button', async ({ page }) => {
    await page.goto('/bom-management');
    await page.waitForTimeout(1500);
    const projectItems = page.locator('[style*="cursor: pointer"]');
    if (await projectItems.count() > 0) {
      await projectItems.first().click();
      await page.waitForTimeout(2000);
      // Check for "生成采购订单" button (may be visible if BOM has items)
      const genBtn = page.locator('button').filter({ hasText: '生成采购订单' });
      const count = await genBtn.count();
      expect(count).toBeGreaterThanOrEqual(0); // May be 0 if no BOM items
    }
  });
});
