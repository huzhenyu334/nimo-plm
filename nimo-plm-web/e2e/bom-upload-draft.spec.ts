import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const STORAGE_STATE_PATH = path.join(__dirname, '.auth', 'storage-state.json');

// Helper: extract JWT token from storageState
function getAuthToken(): string {
  const state = JSON.parse(fs.readFileSync(STORAGE_STATE_PATH, 'utf-8'));
  for (const origin of state.origins || []) {
    for (const item of origin.localStorage || []) {
      if (item.name === 'access_token') return item.value;
    }
  }
  throw new Error('No access_token found in storageState');
}

test.describe('BOM Upload & Draft Save', () => {
  // ========== Upload API Tests ==========

  test('upload API accepts file and returns correct format', async ({ request }) => {
    const token = getAuthToken();
    const response = await request.post('/api/v1/upload', {
      headers: { Authorization: `Bearer ${token}` },
      multipart: {
        files: {
          name: 'test-drawing.pdf',
          mimeType: 'application/pdf',
          buffer: Buffer.from('test file content for upload'),
        },
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();

    // Backend returns: { code: 0, message: "success", data: [{id, url, filename, size, content_type}] }
    expect(body.code).toBe(0);
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBe(1);

    const file = body.data[0];
    expect(file.id).toBeTruthy();
    expect(file.filename).toBe('test-drawing.pdf');
    expect(file.url).toBeTruthy();
    expect(file.size).toBeGreaterThan(0);
  });

  test('upload API response.data is array not object with files key', async ({ request }) => {
    const token = getAuthToken();
    const response = await request.post('/api/v1/upload', {
      headers: { Authorization: `Bearer ${token}` },
      multipart: {
        files: {
          name: 'test.step',
          mimeType: 'application/octet-stream',
          buffer: Buffer.from('3d model data'),
        },
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);

    // data IS the array directly (not an object with a files property)
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.data.files).toBeUndefined();
    expect(body.data[0].id).toBeTruthy();
    expect(body.data[0].filename).toBe('test.step');
  });

  // ========== Draft Save API Tests ==========

  test('draft save API requires auth', async ({ request }) => {
    const response = await request.put('/api/v1/my/tasks/test-task/form-draft', {
      data: { form_data: { test_field: 'value' } },
    });
    expect(response.status()).toBe(401);
  });

  test('draft save API returns error for task without form', async ({ request }) => {
    const token = getAuthToken();
    const response = await request.put('/api/v1/my/tasks/nonexistent-task/form-draft', {
      headers: { Authorization: `Bearer ${token}` },
      data: { form_data: { test_field: 'value' } },
    });

    const body = await response.json();
    // Should get error (task has no form) rather than 500
    expect(body.code).not.toBe(0);
    expect(response.status()).not.toBe(500);
  });

  test('draft load API returns null for non-existent task', async ({ request }) => {
    const token = getAuthToken();
    const response = await request.get('/api/v1/my/tasks/nonexistent-task/form-draft', {
      headers: { Authorization: `Bearer ${token}` },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toBeNull();
  });

  // ========== MyTasks Page UI Tests ==========

  test('MyTasks page loads with task table', async ({ page }) => {
    await page.goto('/my-tasks');
    await page.waitForLoadState('networkidle');

    // Should show the page heading
    await expect(page.getByRole('heading', { name: '我的任务' })).toBeVisible();

    // Should show filter buttons
    await expect(page.getByRole('button', { name: '全部任务' })).toBeVisible();
    await expect(page.getByRole('button', { name: '进行中' })).toBeVisible();
  });

  test('MyTasks filter buttons work', async ({ page }) => {
    await page.goto('/my-tasks');
    await page.waitForLoadState('networkidle');

    // Click "进行中" filter
    await page.getByRole('button', { name: '进行中' }).click();
    await page.waitForTimeout(500);

    // The button should be highlighted (type=primary)
    const btn = page.getByRole('button', { name: '进行中' });
    await expect(btn).toBeVisible();

    // Click back to "全部任务"
    await page.getByRole('button', { name: '全部任务' }).click();
    await page.waitForTimeout(500);
  });

  test('MyTasks task table has correct columns', async ({ page }) => {
    await page.goto('/my-tasks');
    await page.waitForLoadState('networkidle');

    // Check that the task table has the expected column headers
    await expect(page.getByRole('columnheader', { name: '任务标题' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: '负责人' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: '剩余天数' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: '创建人' })).toBeVisible();
  });

  test('clicking task row opens detail view', async ({ page }) => {
    await page.goto('/my-tasks');
    await page.waitForLoadState('networkidle');

    // If there are tasks, click the first row
    const firstRow = page.locator('table tbody tr').first();
    if (await firstRow.isVisible()) {
      await firstRow.click();
      await page.waitForTimeout(500);

      // Should show the "返回任务列表" back button
      await expect(page.getByRole('button', { name: /返回任务列表/ })).toBeVisible();

      // Click back
      await page.getByRole('button', { name: /返回任务列表/ }).click();
      await page.waitForTimeout(500);

      // Should be back to the task list
      await expect(page.getByRole('heading', { name: '我的任务' })).toBeVisible();
    }
  });
});
