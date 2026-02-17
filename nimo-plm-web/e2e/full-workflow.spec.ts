import { test, expect } from '@playwright/test';
import jwt from 'jsonwebtoken';

// ========== Auth setup ==========

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
    {
      sub: TEST_USER.id, uid: TEST_USER.id, name: TEST_USER.name, email: TEST_USER.email,
      feishu_uid: TEST_USER.feishu_uid, roles: ['plm_admin'], perms: ['*'],
      iss: 'nimo-plm', iat: now, exp: now + 86400, jti: `e2e-workflow-${now}`,
    },
    JWT_SECRET, { algorithm: 'HS256' },
  );
}

const authHeaders = { Authorization: `Bearer ${makeToken()}` };

// Helper: extract items from API response (handles both { data: [...] } and { data: { items: [...] } })
function extractItems(body: any): any[] {
  const d = body.data;
  if (Array.isArray(d)) return d;
  if (d?.items && Array.isArray(d.items)) return d.items;
  return [];
}

// Helper: fetch project tasks
async function fetchTasks(request: any, pid: string): Promise<any[]> {
  const resp = await request.get(`/api/v1/projects/${pid}/tasks`, { headers: authHeaders });
  return extractItems(await resp.json());
}

// ========== Shared state ==========

let projectId = '';
let projectName = '';

// ========== Test suite ==========

test.describe.serial('Full Workflow: Template → Project → Tasks → Complete', () => {

  // Safety net: always cleanup even if tests fail mid-way
  test.afterAll(async ({ request }) => {
    if (!projectId) return;
    await request.delete(`/api/v1/projects/${projectId}`, { headers: authHeaders }).catch(() => {});
  });

  // ========== Step 1: Find template and create project via API ==========

  test('Step 1: create project from "眼镜盒开发流程" template', async ({ request }) => {
    // Find the latest published "眼镜盒开发流程" template
    const tplResp = await request.get('/api/v1/templates', { headers: authHeaders });
    expect(tplResp.status()).toBe(200);
    const templates = extractItems(await tplResp.json());

    const candidates = templates
      .filter(t => t.name.includes('眼镜盒') && t.status === 'published')
      .sort((a, b) => parseFloat(b.version) - parseFloat(a.version));
    expect(candidates.length).toBeGreaterThan(0);
    const template = candidates[0];

    // Find an available product codename
    const cnResp = await request.get('/api/v1/codenames?type=product&available=true', { headers: authHeaders });
    expect(cnResp.status()).toBe(200);
    const codenames = extractItems(await cnResp.json());
    // Pick a codename that isn't BlackHole (that's the real project)
    const codename = codenames.find(c => !c.is_used && c.codename !== 'BlackHole');
    expect(codename).toBeTruthy();

    // Create project
    const ts = Date.now();
    projectName = `E2E-Workflow-${ts}`;
    const createResp = await request.post('/api/v1/projects/create-from-template', {
      headers: authHeaders,
      data: {
        template_id: template.id,
        project_name: projectName,
        project_code: `E2E-${ts}`,
        start_date: new Date().toISOString().split('T')[0],
        pm_user_id: TEST_USER.id,
        skip_weekends: true,
        codename_id: codename.id,
      },
    });
    expect(createResp.status()).toBe(201);
    const project = (await createResp.json()).data;
    projectId = project.id;
    expect(projectId).toBeTruthy();
  });

  // ========== Step 2: Verify project in UI ==========

  test('Step 2: project appears in project list UI', async ({ page }) => {
    expect(projectId).toBeTruthy();
    await page.goto('/projects');
    await page.waitForLoadState('domcontentloaded');
    const searchInput = page.getByPlaceholder('搜索项目名称或编号');
    await expect(searchInput).toBeVisible({ timeout: 5000 });
    await searchInput.fill(projectName);
    await page.waitForTimeout(1000);
    await expect(page.getByText(projectName)).toBeVisible({ timeout: 5000 });
  });

  // ========== Step 3: Navigate to project detail and verify tabs ==========

  test('Step 3: project detail page shows correct tabs', async ({ page }) => {
    expect(projectId).toBeTruthy();
    await page.goto(`/projects/${projectId}`);
    await page.waitForLoadState('domcontentloaded');

    await expect(page.getByText(projectName).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('tab', { name: /甘特图/ })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('tab', { name: /BOM管理/ })).toBeVisible();
    await expect(page.getByRole('tab', { name: /角色指派/ })).toBeVisible();
  });

  // ========== Step 4: Assign all roles to self ==========

  test('Step 4: assign all task roles to current user', async ({ request }) => {
    expect(projectId).toBeTruthy();

    const tasks = await fetchTasks(request, projectId);
    expect(tasks.length).toBeGreaterThan(0);

    // Collect unique roles
    const roles = [...new Set(tasks.map(t => t.default_assignee_role).filter(Boolean))] as string[];
    expect(roles.length).toBeGreaterThan(0);

    // Assign all roles to self
    const assignments = roles.map(role => ({ role, user_id: TEST_USER.id }));
    const assignResp = await request.post(`/api/v1/projects/${projectId}/assign-roles`, {
      headers: authHeaders,
      data: { assignments },
    });
    expect(assignResp.status()).toBe(200);

    // Verify tasks now have assignee
    const updatedTasks = await fetchTasks(request, projectId);
    const assignedCount = updatedTasks.filter(t => t.assignee_id === TEST_USER.id).length;
    expect(assignedCount).toBe(updatedTasks.length);
  });

  // ========== Step 5: Verify role assignment in UI ==========

  test('Step 5: role assignment tab shows all roles assigned', async ({ page }) => {
    expect(projectId).toBeTruthy();
    await page.goto(`/projects/${projectId}`);
    await page.waitForLoadState('domcontentloaded');

    const rolesTab = page.locator('.ant-tabs-tab').filter({ hasText: '角色指派' });
    await expect(rolesTab).toBeVisible({ timeout: 5000 });
    await rolesTab.click();
    await page.waitForTimeout(1000);

    await expect(page.getByText('陈泽斌').first()).toBeVisible({ timeout: 5000 });
  });

  // ========== Step 6: Complete first task (项目立项) ==========

  test('Step 6: complete first task "项目立项" via API and verify', async ({ request }) => {
    expect(projectId).toBeTruthy();

    const tasks = await fetchTasks(request, projectId);
    const firstTask = tasks.find(t => t.status === 'in_progress');
    expect(firstTask).toBeTruthy();
    expect(firstTask.title).toContain('项目立项');

    // Complete the task (as the assignee via my/tasks)
    const completeResp = await request.post(`/api/v1/my/tasks/${firstTask.id}/complete`, {
      headers: authHeaders,
      data: { form_data: {} },
    });
    expect(completeResp.status()).toBe(200);

    // Confirm the task as PM
    const confirmResp = await request.post(`/api/v1/projects/${projectId}/tasks/${firstTask.id}/confirm`, {
      headers: authHeaders,
    });
    expect(confirmResp.status()).toBe(200);

    // Verify task is now completed and downstream tasks activated
    const updated = await fetchTasks(request, projectId);
    const completedTask = updated.find(t => t.id === firstTask.id);
    expect(completedTask.status).toBe('completed');

    const activeTasks = updated.filter(t => t.status === 'in_progress');
    expect(activeTasks.length).toBeGreaterThanOrEqual(2);
  });

  // ========== Step 7: Complete all remaining tasks in dependency order ==========

  test('Step 7: complete all remaining tasks through dependency chain', async ({ request }) => {
    expect(projectId).toBeTruthy();

    // Process tasks in waves until all are completed
    for (let wave = 0; wave < 10; wave++) {
      const tasks = await fetchTasks(request, projectId);
      const inProgress = tasks.filter(t => t.status === 'in_progress');

      if (inProgress.length === 0) {
        const allDone = tasks.every(t => t.status === 'completed');
        expect(allDone).toBe(true);
        break;
      }

      for (const task of inProgress) {
        const completeResp = await request.post(`/api/v1/my/tasks/${task.id}/complete`, {
          headers: authHeaders,
          data: { form_data: {} },
        });
        expect(completeResp.status()).toBe(200);

        const confirmResp = await request.post(`/api/v1/projects/${projectId}/tasks/${task.id}/confirm`, {
          headers: authHeaders,
        });
        expect(confirmResp.status()).toBe(200);
      }
    }

    // Final verification: all tasks completed
    const finalTasks = await fetchTasks(request, projectId);
    for (const task of finalTasks) {
      expect(task.status).toBe('completed');
    }
  });

  // ========== Step 8: Verify completed state in UI ==========

  test('Step 8: project detail shows all tasks completed in Gantt view', async ({ page }) => {
    expect(projectId).toBeTruthy();
    await page.goto(`/projects/${projectId}`);
    await page.waitForLoadState('domcontentloaded');

    const ganttTab = page.locator('.ant-tabs-tab').filter({ hasText: '甘特图' });
    await expect(ganttTab).toBeVisible({ timeout: 5000 });
    await ganttTab.click();
    await page.waitForTimeout(2000);

    // All task rows should show "已完成" tag
    const completedTags = page.locator('.ant-tag').filter({ hasText: '已完成' });
    const count = await completedTags.count();
    expect(count).toBeGreaterThanOrEqual(8);
  });

  // ========== Step 9: Verify project overview progress ==========

  test('Step 9: project overview shows 100% progress', async ({ page }) => {
    expect(projectId).toBeTruthy();
    await page.goto(`/projects/${projectId}`);
    await page.waitForLoadState('domcontentloaded');

    // Wait for overview to load
    await page.waitForTimeout(2000);

    // Look for 100% progress text
    const progressText = page.getByText('100%');
    if (await progressText.count() > 0) {
      await expect(progressText.first()).toBeVisible();
    } else {
      // At minimum verify no tasks are "in_progress" in overview
      const inProgressTag = page.locator('.ant-tag').filter({ hasText: '进行中' });
      expect(await inProgressTag.count()).toBe(0);
    }
  });

  // ========== Cleanup: delete the test project ==========

  test('Cleanup: delete test project', async ({ request }) => {
    if (!projectId) return;

    const deleteResp = await request.delete(`/api/v1/projects/${projectId}`, { headers: authHeaders });
    expect(deleteResp.status()).toBe(200);

    // Verify project is gone
    const listResp = await request.get('/api/v1/projects', { headers: authHeaders });
    const items = extractItems(await listResp.json());
    const found = items.find((p: any) => p.id === projectId);
    expect(found).toBeUndefined();

    // Clear projectId so afterAll doesn't try to delete again
    projectId = '';
  });
});
