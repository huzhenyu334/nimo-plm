import { test, expect } from '@playwright/test';

// Auth tests need to test unauthenticated scenarios, so clear storageState
test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Authentication', () => {
  test('login page loads', async ({ page }) => {
    await page.goto('/login');
    // Should show the login page with Feishu login button
    await expect(page).toHaveURL(/\/login/);
    // Check that the page contains login-related content
    const body = await page.textContent('body');
    expect(body).toBeTruthy();
  });

  test('unauthenticated user redirects to login', async ({ page }) => {
    // Navigate to a protected page
    await page.goto('/dashboard');

    // Should redirect to login
    await page.waitForURL(/\/login/, { timeout: 5000 });
    await expect(page).toHaveURL(/\/login/);
  });

  test('health check endpoint works', async ({ request }) => {
    const response = await request.get('/health/live');
    expect(response.status()).toBe(200);
  });

  test('API returns 401 without token', async ({ request }) => {
    const response = await request.get('/api/v1/auth/me');
    expect(response.status()).toBe(401);
  });

  test('Feishu login endpoint redirects to Feishu', async ({ request }) => {
    const response = await request.get('/api/v1/auth/feishu/login', {
      maxRedirects: 0,
    });
    // Should be a redirect (302)
    expect(response.status()).toBe(302);
    const location = response.headers()['location'];
    expect(location).toContain('open.feishu.cn');
  });

  // Feishu SSO full flow requires external OAuth - skip in automated tests
  test.skip('full Feishu SSO login flow', async ({ page }) => {
    // This test would require:
    // 1. Navigate to /login
    // 2. Click Feishu login button
    // 3. Complete Feishu OAuth (requires real Feishu account)
    // 4. Verify redirect back with tokens
    // Skipped because it needs external Feishu authentication
    await page.goto('/login');
  });
});
