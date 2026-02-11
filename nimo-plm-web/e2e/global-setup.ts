import { chromium } from '@playwright/test';
import jwt from 'jsonwebtoken';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const JWT_SECRET = 'nimo-plm-jwt-secret-key-2024';
const STORAGE_STATE_PATH = path.join(__dirname, '.auth', 'storage-state.json');

const TEST_USER = {
  id: 'd406707b-f9bb-4e9f-8cc5-0cab38e4',
  feishu_user_id: 'c99c9eb3',
  username: 'zebin',
  name: '陈泽斌',
  email: 'zebin@bitfantasy.io',
  status: 'active',
};

async function globalSetup() {
  // Generate a valid JWT token matching backend's expected format (using real user from DB)
  const now = Math.floor(Date.now() / 1000);
  const token = jwt.sign(
    {
      sub: TEST_USER.id,
      uid: TEST_USER.id,
      name: TEST_USER.name,
      email: TEST_USER.email,
      feishu_uid: TEST_USER.feishu_user_id,
      roles: ['plm_admin'],
      perms: ['*'],
      iss: 'nimo-plm',
      iat: now,
      exp: now + 86400, // 24 hours
      jti: `e2e-jti-${now}`,
    },
    JWT_SECRET,
    { algorithm: 'HS256' }
  );

  // Ensure .auth directory exists
  fs.mkdirSync(path.dirname(STORAGE_STATE_PATH), { recursive: true });

  // Launch browser, set localStorage, save storageState
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();

  await page.goto('http://localhost:8080/login');
  await page.evaluate((t) => {
    localStorage.setItem('access_token', t);
  }, token);

  await context.storageState({ path: STORAGE_STATE_PATH });
  await browser.close();
}

export default globalSetup;
