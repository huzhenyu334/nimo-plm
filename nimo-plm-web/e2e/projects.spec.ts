import { test, expect } from '@playwright/test';

test.describe('Project Management', () => {
  test('projects API returns 401 without auth', async ({ request }) => {
    const response = await request.get('/api/v1/projects');
    expect(response.status()).toBe(401);
  });

  test('project list page loads', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    // Should show the project list heading
    await expect(page.getByRole('heading', { name: '研发项目' })).toBeVisible();
  });

  test('project list shows table columns', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    // Verify table column headers are visible
    await expect(page.getByRole('columnheader', { name: '项目编码' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: '项目名称' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: '操作' })).toBeVisible();
  });

  test('create project button navigates to templates', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    // Click the create button which navigates to templates
    await page.getByRole('button', { name: '从研发流程创建项目' }).click();

    // Should navigate to templates page
    await expect(page).toHaveURL(/\/templates/);
  });

  test('view project detail', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    // Click "详情" link on the first project row
    const detailLink = page.getByRole('button', { name: '详情' }).first();
    if (await detailLink.isVisible()) {
      await detailLink.click();
      // Should navigate to detail page
      await expect(page).toHaveURL(/\/projects\/.+/);
    }
  });
});
