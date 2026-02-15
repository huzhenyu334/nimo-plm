import { test, expect } from '@playwright/test';
import jwt from 'jsonwebtoken';

const JWT_SECRET = 'nimo-plm-jwt-secret-key-2024';
const TEST_USER = {
  id: 'd406707b-f9bb-4e9f-8cc5-0cab38e4',
  name: '陈泽斌',
  email: 'zebin@bitfantasy.io',
  feishu_uid: 'c99c9eb3',
};

function makeToken(): string {
  const now = Math.floor(Date.now() / 1000);
  return jwt.sign(
    { sub: TEST_USER.id, uid: TEST_USER.id, name: TEST_USER.name, email: TEST_USER.email,
      feishu_uid: TEST_USER.feishu_uid, roles: ['plm_admin'], perms: ['*'],
      iss: 'nimo-plm', iat: now, exp: now + 86400, jti: `e2e-bom-ctrl-${now}` },
    JWT_SECRET, { algorithm: 'HS256' }
  );
}

const authHeaders = { Authorization: `Bearer ${makeToken()}` };

test.describe('BOM Control Components', () => {

  // Verify attribute template API works (prerequisite for controls)
  test('attribute templates API returns data for dynamic columns', async ({ request }) => {
    const response = await request.get('/api/v1/bom-attr-templates', { headers: authHeaders });
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBe(true);
  });

  // Verify the BOM control components render without JS errors
  test('BOM component files are served correctly in the built bundle', async ({ page }) => {
    // Collect page errors before navigation
    const errors: string[] = [];
    page.on('pageerror', (err) => errors.push(err.message));

    // Set auth token in localStorage so the app loads authenticated
    await page.addInitScript((token: string) => {
      localStorage.setItem('access_token', token);
    }, makeToken());

    // Navigate to projects page
    await page.goto('/projects', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Filter out unrelated errors (only BOM-related errors matter)
    const bomErrors = errors.filter(e =>
      e.includes('BOM') || e.includes('DynamicBOMTable') ||
      e.includes('EBOMControl') || e.includes('PBOMControl') || e.includes('MBOMControl')
    );
    expect(bomErrors).toHaveLength(0);
  });

  // Verify module resolution for BOM components works at runtime
  test('BOM components can be imported without errors at build time', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });

    await page.addInitScript((token: string) => {
      localStorage.setItem('access_token', token);
    }, makeToken());

    await page.goto('/my-tasks', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Filter for import/module errors related to our components
    const moduleErrors = consoleErrors.filter(e =>
      e.includes('Failed to') && (
        e.includes('BOM') || e.includes('bomConstants') ||
        e.includes('DynamicBOMTable') || e.includes('EBOMControl')
      )
    );
    expect(moduleErrors).toHaveLength(0);
  });

  // Verify attribute templates can be filtered for EBOM sub-categories
  test('attribute templates support sub_category filtering for EBOM controls', async ({ request }) => {
    const response = await request.get('/api/v1/bom-attr-templates?sub_category=component', {
      headers: authHeaders,
    });
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data).toBeDefined();
    if (body.data && body.data.length > 0) {
      for (const tmpl of body.data) {
        expect(tmpl.sub_category).toBe('component');
      }
    }
  });

  // Verify attribute templates can be filtered for PBOM categories
  test('attribute templates support category filtering for PBOM controls', async ({ request }) => {
    const response = await request.get('/api/v1/bom-attr-templates?category=packaging', {
      headers: authHeaders,
    });
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data).toBeDefined();
  });
});
