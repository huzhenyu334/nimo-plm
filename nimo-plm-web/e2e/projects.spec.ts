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

  test('project list shows card layout with search and create', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    // Verify search input exists
    await expect(page.getByPlaceholder('搜索项目名称或编号')).toBeVisible();
    // Verify create button exists
    await expect(page.getByRole('button', { name: '创建项目' })).toBeVisible();
  });

  test('create project button navigates to templates', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    // Click the create button which navigates to templates
    await page.getByRole('button', { name: '创建项目' }).click();

    // Should navigate to templates page
    await expect(page).toHaveURL(/\/templates/);
  });

  test('click project card navigates to detail', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    // Click the first project card (if any exist)
    const card = page.locator('.ant-card-hoverable').first();
    if (await card.isVisible()) {
      await card.click();
      // Should navigate to detail page
      await expect(page).toHaveURL(/\/projects\/.+/);
    }
  });
});
