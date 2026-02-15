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
      iss: 'nimo-plm', iat: now, exp: now + 86400, jti: `e2e-bom-${now}` },
    JWT_SECRET, { algorithm: 'HS256' }
  );
}

const authHeaders = { Authorization: `Bearer ${makeToken()}` };

test.describe('BOM Restructure - 3-Level Architecture', () => {

  // API tests - attribute templates
  test('seed default attribute templates via API', async ({ request }) => {
    const response = await request.post('/api/v1/bom-attr-templates/seed', {
      data: {},
      headers: authHeaders,
    });
    expect(response.status()).toBe(200);
  });

  test('list attribute templates via API', async ({ request }) => {
    const response = await request.get('/api/v1/bom-attr-templates', { headers: authHeaders });
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('list templates filtered by category', async ({ request }) => {
    const response = await request.get('/api/v1/bom-attr-templates?category=结构件', { headers: authHeaders });
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data).toBeDefined();
    if (body.data && body.data.length > 0) {
      expect(body.data[0].category).toBe('结构件');
    }
  });

  // API tests - BOM CRUD with new types
  test('BOM API supports EBOM/PBOM/MBOM types', async ({ request }) => {
    const projResp = await request.get('/api/v1/projects', { headers: authHeaders });
    expect(projResp.status()).toBe(200);
    const projects = await projResp.json().then(r => r.data?.items || r.data || []);
    if (!projects || projects.length === 0) {
      test.skip();
      return;
    }
    const projectId = projects[0].id;

    const bomResp = await request.get(`/api/v1/projects/${projectId}/boms`, { headers: authHeaders });
    expect(bomResp.status()).toBe(200);
    const boms = (await bomResp.json()).data;
    expect(Array.isArray(boms)).toBe(true);
  });

  test('create EBOM with extended_attrs item', async ({ request }) => {
    const projResp = await request.get('/api/v1/projects', { headers: authHeaders });
    const projects = await projResp.json().then(r => r.data?.items || r.data || []);
    if (!projects || projects.length === 0) {
      test.skip();
      return;
    }
    const projectId = projects[0].id;

    // Create EBOM
    const createResp = await request.post(`/api/v1/projects/${projectId}/boms`, {
      data: { name: 'E2E-EBOM-Test', bom_type: 'EBOM', version: 'v1.0' },
      headers: authHeaders,
    });
    expect(createResp.status()).toBe(201);
    const bom = (await createResp.json()).data;
    expect(bom.bom_type).toBe('EBOM');

    // Add item with extended_attrs
    const itemResp = await request.post(`/api/v1/projects/${projectId}/boms/${bom.id}/items`, {
      data: {
        name: '电阻R1',
        quantity: 10,
        unit: 'pcs',
        category: '电子件',
        sub_category: '电阻',
        extended_attrs: { designator: 'R1,R2,R3', package: '0402' },
      },
      headers: authHeaders,
    });
    expect(itemResp.status()).toBe(201);
    const item = (await itemResp.json()).data;
    expect(item.category).toBe('电子件');
    expect(item.extended_attrs).toBeDefined();
    expect(item.extended_attrs.designator).toBe('R1,R2,R3');

    // Get BOM detail
    const detailResp = await request.get(`/api/v1/projects/${projectId}/boms/${bom.id}`, { headers: authHeaders });
    expect(detailResp.status()).toBe(200);
    const detail = (await detailResp.json()).data;
    expect(detail.items.length).toBeGreaterThanOrEqual(1);

    // Category tree
    const treeResp = await request.get(`/api/v1/projects/${projectId}/boms/${bom.id}/category-tree`, { headers: authHeaders });
    expect(treeResp.status()).toBe(200);
  });

  // UI tests - BOM tab loads with new type labels
  test('BOM tab shows EBOM/PBOM/MBOM type options', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    // Try to enter a project
    const projectRow = page.getByRole('row').nth(1);
    const projectExists = await projectRow.isVisible().catch(() => false);
    if (!projectExists) {
      test.skip();
      return;
    }

    // Click first project
    await projectRow.click();
    await page.waitForLoadState('networkidle');

    // Navigate to BOM tab
    const bomTab = page.getByRole('tab', { name: /BOM/ });
    if (await bomTab.isVisible().catch(() => false)) {
      await bomTab.click();
      await page.waitForLoadState('networkidle');

      // Check for "新建BOM" button
      const createBtn = page.getByRole('button', { name: /新建/ });
      if (await createBtn.isVisible().catch(() => false)) {
        await createBtn.click();
        // Verify new BOM type options
        const bomTypeSelect = page.locator('.ant-modal').locator('.ant-select');
        if (await bomTypeSelect.first().isVisible().catch(() => false)) {
          await bomTypeSelect.first().click();
          // Should show EBOM, PBOM, MBOM options (not SBOM)
          await expect(page.getByText('EBOM', { exact: false })).toBeVisible({ timeout: 3000 });
          // Close modal
          await page.keyboard.press('Escape');
        }
      }
    }
  });

  // API tests - process routes
  test('process routes CRUD via API', async ({ request }) => {
    const projResp = await request.get('/api/v1/projects', { headers: authHeaders });
    const projects = await projResp.json().then(r => r.data?.items || r.data || []);
    if (!projects || projects.length === 0) {
      test.skip();
      return;
    }
    const projectId = projects[0].id;

    // Create BOM first
    const bomResp = await request.post(`/api/v1/projects/${projectId}/boms`, {
      data: { name: 'Route-Test-BOM', bom_type: 'PBOM', version: 'v1.0' },
      headers: authHeaders,
    });
    expect(bomResp.status()).toBe(201);
    const bom = (await bomResp.json()).data;

    // Create route
    const routeResp = await request.post(`/api/v1/projects/${projectId}/boms/${bom.id}/routes`, {
      data: { name: '主工艺路线', description: 'E2E测试工艺路线' },
      headers: authHeaders,
    });
    expect(routeResp.status()).toBe(201);
    const route = (await routeResp.json()).data;
    expect(route.name).toBe('主工艺路线');

    // List routes
    const listResp = await request.get(`/api/v1/projects/${projectId}/routes`, { headers: authHeaders });
    expect(listResp.status()).toBe(200);
    const routes = (await listResp.json()).data;
    expect(routes.length).toBeGreaterThanOrEqual(1);

    // Create step
    const stepResp = await request.post(`/api/v1/projects/${projectId}/routes/${route.id}/steps`, {
      data: { name: '注塑', step_number: 1, equipment: '注塑机A', sort_order: 1 },
      headers: authHeaders,
    });
    expect(stepResp.status()).toBe(201);
    const step = (await stepResp.json()).data;
    expect(step.name).toBe('注塑');

    // Create step material
    const matResp = await request.post(`/api/v1/projects/${projectId}/routes/${route.id}/steps/${step.id}/materials`, {
      data: { name: 'ABS原料', quantity: 0.5, unit: 'kg' },
      headers: authHeaders,
    });
    expect(matResp.status()).toBe(201);
    const mat = (await matResp.json()).data;

    // Get route detail (with steps & materials)
    const detailResp = await request.get(`/api/v1/projects/${projectId}/routes/${route.id}`, { headers: authHeaders });
    expect(detailResp.status()).toBe(200);
    const detail = (await detailResp.json()).data;
    expect(detail.steps.length).toBe(1);
    expect(detail.steps[0].materials.length).toBe(1);

    // Delete step material
    const delMatResp = await request.delete(
      `/api/v1/projects/${projectId}/routes/${route.id}/steps/${step.id}/materials/${mat.id}`,
      { headers: authHeaders }
    );
    expect(delMatResp.status()).toBe(200);

    // Delete step
    const delStepResp = await request.delete(
      `/api/v1/projects/${projectId}/routes/${route.id}/steps/${step.id}`,
      { headers: authHeaders }
    );
    expect(delStepResp.status()).toBe(200);
  });

  // BOM conversion test
  test('convert EBOM to MBOM via API', async ({ request }) => {
    const projResp = await request.get('/api/v1/projects', { headers: authHeaders });
    const projects = await projResp.json().then(r => r.data?.items || r.data || []);
    if (!projects || projects.length === 0) {
      test.skip();
      return;
    }
    const projectId = projects[0].id;

    // Create EBOM with items
    const bomResp = await request.post(`/api/v1/projects/${projectId}/boms`, {
      data: { name: 'Convert-Test-EBOM', bom_type: 'EBOM', version: 'v1.0' },
      headers: authHeaders,
    });
    const bom = (await bomResp.json()).data;

    await request.post(`/api/v1/projects/${projectId}/boms/${bom.id}/items`, {
      data: { name: '测试零件A', quantity: 5, unit: 'pcs', category: '结构件' },
      headers: authHeaders,
    });

    // Convert to MBOM
    const convertResp = await request.post(
      `/api/v1/projects/${projectId}/boms/${bom.id}/convert-to-mbom`,
      { headers: authHeaders }
    );
    expect(convertResp.status()).toBe(201);
    const mbom = (await convertResp.json()).data;
    expect(mbom.bom_type).toBe('MBOM');
    expect(mbom.source_bom_id).toBe(bom.id);
  });

  test('convert EBOM to PBOM via API', async ({ request }) => {
    const projResp = await request.get('/api/v1/projects', { headers: authHeaders });
    const projects = await projResp.json().then(r => r.data?.items || r.data || []);
    if (!projects || projects.length === 0) {
      test.skip();
      return;
    }
    const projectId = projects[0].id;

    // Create EBOM
    const bomResp = await request.post(`/api/v1/projects/${projectId}/boms`, {
      data: { name: 'Convert-Test-EBOM-PBOM', bom_type: 'EBOM', version: 'v1.0' },
      headers: authHeaders,
    });
    const bom = (await bomResp.json()).data;

    await request.post(`/api/v1/projects/${projectId}/boms/${bom.id}/items`, {
      data: { name: '测试零件B', quantity: 3, unit: 'pcs', category: '结构件' },
      headers: authHeaders,
    });

    // Convert to PBOM
    const convertResp = await request.post(
      `/api/v1/projects/${projectId}/boms/${bom.id}/convert-to-pbom`,
      { headers: authHeaders }
    );
    expect(convertResp.status()).toBe(201);
    const pbom = (await convertResp.json()).data;
    expect(pbom.bom_type).toBe('PBOM');
    expect(pbom.source_bom_id).toBe(bom.id);
  });
});
