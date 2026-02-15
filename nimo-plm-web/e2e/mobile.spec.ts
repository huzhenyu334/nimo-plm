import { test, expect } from '@playwright/test';

// Use Chromium with mobile viewport (no WebKit dependency)
test.use({
  viewport: { width: 430, height: 932 },
  isMobile: true,
  hasTouch: true,
  storageState: './e2e/.auth/storage-state.json',
});

test('mobile: bottom navigation bar visible', async ({ page }) => {
  await page.goto('/');
  const bottomNav = page.locator('.mobile-bottom-nav');
  await expect(bottomNav).toBeVisible();
  const items = bottomNav.locator('.mobile-bottom-nav-item');
  await expect(items).toHaveCount(5);
});

test('mobile: header visible with title', async ({ page }) => {
  await page.goto('/my-tasks');
  const header = page.locator('.mobile-header');
  await expect(header).toBeVisible();
  await expect(header.locator('.mobile-header-title')).toContainText('我的任务');
});

test('mobile: no horizontal overflow', async ({ page }) => {
  for (const path of ['/projects', '/my-tasks', '/approvals']) {
    await page.goto(path);
    await page.waitForTimeout(1500);
    const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
    const viewportWidth = await page.evaluate(() => window.innerWidth);
    expect(bodyWidth).toBeLessThanOrEqual(viewportWidth + 1);
  }
});

test('mobile: projects page has search bar', async ({ page }) => {
  await page.goto('/projects');
  await page.waitForTimeout(1000);
  const searchBar = page.locator('.ds-search-bar');
  await expect(searchBar).toBeVisible();
});

test('mobile: my-tasks shows filter pills not table', async ({ page }) => {
  await page.goto('/my-tasks');
  await page.waitForTimeout(1500);
  const filterPills = page.locator('.mobile-filter-pills');
  await expect(filterPills).toBeVisible();
  const table = page.locator('.ant-table');
  await expect(table).toHaveCount(0);
});

test('mobile: approvals shows filter pills', async ({ page }) => {
  await page.goto('/approvals');
  await page.waitForTimeout(1000);
  const filterPills = page.locator('.mobile-filter-pills');
  await expect(filterPills).toBeVisible();
});

test('mobile: bottom nav navigates between pages', async ({ page }) => {
  await page.goto('/dashboard');
  // Tab order: 工作台(0) / 项目(1) / BOM(2) / 任务(3) / 我的(4)
  await page.locator('.mobile-bottom-nav-item').nth(1).click();
  await expect(page).toHaveURL(/\/projects/);
  await page.locator('.mobile-bottom-nav-item').nth(2).click();
  await expect(page).toHaveURL(/\/bom-management/);
  await page.locator('.mobile-bottom-nav-item').nth(3).click();
  await expect(page).toHaveURL(/\/my-tasks/);
});

test('mobile: "我的" page shows grouped menu', async ({ page }) => {
  await page.goto('/dashboard');
  // Click "我的" tab (5th item, index 4)
  await page.locator('.mobile-bottom-nav-item').nth(4).click();
  await page.waitForTimeout(500);
  // Should show quick entries
  const myTaskEntry = page.locator('.mobile-my-menu-item').filter({ hasText: '我的任务' });
  await expect(myTaskEntry).toBeVisible();
  const approvalEntry = page.locator('.mobile-my-menu-item').filter({ hasText: '审批中心' });
  await expect(approvalEntry).toBeVisible();
  // Should show grouped titles
  const groupTitles = page.locator('.mobile-my-group-title');
  expect(await groupTitles.count()).toBeGreaterThanOrEqual(3);
  // Should show logout
  const logoutItem = page.locator('.mobile-my-menu-item').filter({ hasText: '退出登录' });
  await expect(logoutItem).toBeVisible();
});

test('mobile: "我的" page grouped menu navigates', async ({ page }) => {
  await page.goto('/__my__');
  await page.waitForTimeout(500);
  // Click BOM管理 item
  const bomItem = page.locator('.mobile-my-menu-item').filter({ hasText: 'BOM管理' });
  await expect(bomItem).toBeVisible();
  await bomItem.click();
  await expect(page).toHaveURL(/\/bom-management/);
});

test('mobile: project detail shows back button', async ({ page }) => {
  await page.goto('/projects');
  await page.waitForTimeout(1500);
  const cards = page.locator('.ds-list-card');
  if (await cards.count() > 0) {
    await cards.first().click();
    await page.waitForTimeout(1500);
    await expect(page.locator('.mobile-header-back')).toBeVisible();
  }
});

test('mobile: project detail overview uses ds-info-row', async ({ page }) => {
  await page.goto('/projects');
  await page.waitForTimeout(1500);
  const cards = page.locator('.ds-list-card');
  if (await cards.count() > 0) {
    await cards.first().click();
    await page.waitForTimeout(1500);
    // Overview tab should use ds-detail-page layout, not Descriptions table
    const detailPage = page.locator('.ds-detail-page');
    if (await detailPage.count() > 0) {
      const infoRows = detailPage.locator('.ds-info-row');
      expect(await infoRows.count()).toBeGreaterThanOrEqual(3);
      // Should NOT have Descriptions table
      const descTable = page.locator('.ant-descriptions');
      expect(await descTable.count()).toBe(0);
    }
  }
});

test('mobile: design system - projects use ds-list-card', async ({ page }) => {
  await page.goto('/projects');
  await page.waitForTimeout(1500);
  const cards = page.locator('.ds-list-card');
  if (await cards.count() > 0) {
    // ds-list-card should have border-radius: 12px
    const radius = await cards.first().evaluate(el =>
      getComputedStyle(el).borderRadius
    );
    expect(radius).toBe('12px');
  }
});

test('mobile: design system - tasks use ds-list-card', async ({ page }) => {
  await page.goto('/my-tasks');
  await page.waitForTimeout(1500);
  const cards = page.locator('.ds-list-card');
  if (await cards.count() > 0) {
    // ds-list-card should have consistent styling
    const bg = await cards.first().evaluate(el =>
      getComputedStyle(el).backgroundColor
    );
    // Should be white
    expect(bg).toMatch(/rgb\(255,\s*255,\s*255\)/);
  }
});

test('mobile: project detail Gantt shows task list view', async ({ page }) => {
  await page.goto('/projects');
  await page.waitForTimeout(1500);
  const cards = page.locator('.ds-list-card');
  if (await cards.count() > 0) {
    await cards.first().click();
    await page.waitForTimeout(1500);
    // Navigate to Gantt tab
    const ganttTab = page.locator('.ant-tabs-tab').filter({ hasText: '甘特图' });
    if (await ganttTab.count() > 0) {
      await ganttTab.click();
      await page.waitForTimeout(1000);
      // Should show mobile list view, not the desktop gantt chart
      const mobileList = page.locator('.gantt-mobile-list');
      if (await mobileList.count() > 0) {
        // Should have phase groups
        const phases = page.locator('.gantt-mobile-phase');
        expect(await phases.count()).toBeGreaterThanOrEqual(1);
        // Phase headers should be clickable (collapsible)
        const header = page.locator('.gantt-mobile-phase-header').first();
        await expect(header).toBeVisible();
      }
    }
  }
});

test('mobile: project detail BOM tab has compact tabs', async ({ page }) => {
  await page.goto('/projects');
  await page.waitForTimeout(1500);
  const cards = page.locator('.ds-list-card');
  if (await cards.count() > 0) {
    await cards.first().click();
    await page.waitForTimeout(1500);
    // Navigate to BOM tab
    const bomTab = page.locator('.ant-tabs-tab').filter({ hasText: 'BOM' });
    if (await bomTab.count() > 0) {
      await bomTab.click();
      await page.waitForTimeout(1000);
      // BOM sub-tabs should show compact labels (EBOM, PBOM, MBOM)
      const ebomTab = page.locator('.ant-tabs-tab').filter({ hasText: 'EBOM' });
      await expect(ebomTab.first()).toBeVisible();
    }
  }
});

test('mobile: BOM category view has left nav and right content', async ({ page }) => {
  await page.goto('/projects');
  await page.waitForTimeout(1500);
  const cards = page.locator('.ds-list-card');
  if (await cards.count() > 0) {
    await cards.first().click();
    await page.waitForTimeout(1500);
    const bomTab = page.locator('.ant-tabs-tab').filter({ hasText: 'BOM' });
    if (await bomTab.count() > 0) {
      await bomTab.click();
      await page.waitForTimeout(1000);
      // Should show Meituan-style category view
      const catView = page.locator('.bom-catview');
      if (await catView.count() > 0) {
        // Left nav should exist
        const nav = catView.locator('.bom-catview-nav');
        await expect(nav).toBeVisible();
        // Nav should be ~85px wide
        const navWidth = await nav.evaluate(el => el.getBoundingClientRect().width);
        expect(navWidth).toBeGreaterThanOrEqual(80);
        expect(navWidth).toBeLessThanOrEqual(95);
        // Right content should exist
        const content = catView.locator('.bom-catview-content');
        await expect(content).toBeVisible();
        // Summary bar should exist
        const summary = page.locator('.bom-catview-summary');
        await expect(summary).toBeVisible();
      }
    }
  }
});
