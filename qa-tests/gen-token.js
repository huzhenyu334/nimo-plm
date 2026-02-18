#!/usr/bin/env node
/**
 * Generate JWT auth state for web-qa-bot tests.
 * Outputs a Playwright-compatible storage state JSON to stdout.
 *
 * Usage:
 *   node gen-token.js > .auth-state.json
 *   AGENT_BROWSER_STATE=.auth-state.json web-qa-bot run plm-full.yaml
 */
const crypto = require('crypto');

const JWT_SECRET = 'nimo-plm-jwt-secret-key-2024';
const BASE_URL = process.env.PLM_BASE_URL || 'http://43.134.86.237:8080';

const TEST_USER = {
  id: 'd406707b-f9bb-4e9f-8cc5-0cab38e4',
  feishu_user_id: 'c99c9eb3',
  name: '陈泽斌',
  email: 'zebin@bitfantasy.io',
};

function base64url(buf) {
  return buf.toString('base64').replace(/=/g, '').replace(/\+/g, '-').replace(/\//g, '_');
}

const now = Math.floor(Date.now() / 1000);
const header = base64url(Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })));
const payload = base64url(Buffer.from(JSON.stringify({
  sub: TEST_USER.id,
  uid: TEST_USER.id,
  name: TEST_USER.name,
  email: TEST_USER.email,
  feishu_uid: TEST_USER.feishu_user_id,
  roles: ['plm_admin'],
  perms: ['*'],
  iss: 'nimo-plm',
  iat: now,
  exp: now + 86400,
  jti: `qa-${now}`,
})));
const sig = base64url(
  crypto.createHmac('sha256', JWT_SECRET).update(header + '.' + payload).digest()
);
const token = `${header}.${payload}.${sig}`;

const state = {
  cookies: [],
  origins: [{
    origin: BASE_URL,
    localStorage: [{ name: 'access_token', value: token }],
  }],
};

console.log(JSON.stringify(state, null, 2));
