import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import path from 'node:path';
import type { FullConfig } from '@playwright/test';

// Runs once per `playwright test` (also for a single spec). Expects `scripts/e2e-stack.sh start`.
const BASE_URL = process.env.E2E_BASE_URL ?? 'http://127.0.0.1:3100';
const API_URL = process.env.E2E_API_URL ?? 'http://127.0.0.1:8180';
const EMULATOR = `http://${process.env.FIREBASE_AUTH_EMULATOR_HOST ?? '127.0.0.1:9099'}`;
const PROJECT_ID = 'demo-sakunoie';
const E2E_DSN = process.env.GORM_DATABASE_URL_E2E ?? 'root:test@tcp(127.0.0.1:33306)/lodging_e2e?parseTime=True';

const assertComposeE2E = (dsn: string) => {
  const m = /@tcp\(([^)]+)\)\/([^?]+)/.exec(dsn);
  const port = process.env.TEST_DB_PORT ?? '33306';
  if (!m || m[1] !== `127.0.0.1:${port}` || m[2] !== 'lodging_e2e') {
    throw new Error('GORM_DATABASE_URL_E2E must point at the compose DB 127.0.0.1:33306/lodging_e2e');
  }
};

const ensureUp = async (url: string, name: string) => {
  try {
    const res = await fetch(url);
    if (!res.ok) throw new Error(`status ${res.status}`);
  } catch (error) {
    throw new Error(
      `${name} is not reachable at ${url} (${String(error)}). Start it with: bash scripts/e2e-stack.sh start`
    );
  }
};

const signUp = async (email: string, password: string): Promise<string> => {
  const res = await fetch(`${EMULATOR}/identitytoolkit.googleapis.com/v1/accounts:signUp?key=demo-api-key`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email, password, returnSecureToken: true })
  });
  if (!res.ok) throw new Error(`emulator signUp failed: ${res.status}`);
  return ((await res.json()) as { localId: string }).localId;
};

export default async function globalSetup(config: FullConfig) {
  const frontendDir = config.configFile ? path.dirname(config.configFile) : process.cwd();
  const backendDir = path.resolve(frontendDir, '../backend');
  assertComposeE2E(E2E_DSN);
  await ensureUp(`${BASE_URL}/ja`, 'Next (web)');
  await ensureUp(`${API_URL}/api/v1/facilities/saku-no-ie/public`, 'API (cross-origin)');
  await ensureUp(`${EMULATOR}/`, 'Auth Emulator');

  // Fresh emulator users each run; the password is random and only kept in env for the workers.
  const clear = await fetch(`${EMULATOR}/emulator/v1/projects/${PROJECT_ID}/accounts`, { method: 'DELETE' });
  if (!clear.ok) throw new Error(`emulator reset failed: ${clear.status}`);
  const password = `e2e-${randomUUID()}`;
  const adminEmail = 'e2e-admin@example.test';
  const nobodyEmail = 'e2e-nobody@example.test';
  const adminUid = await signUp(adminEmail, password);
  await signUp(nobodyEmail, password);

  // Owner grant on compose lodging_e2e (admin-grant refuses any other target). The DSN goes through env.
  execFileSync('go', ['run', './cmd/admin-grant', '--uid', adminUid, '--facility', 'saku-no-ie'], {
    cwd: backendDir,
    env: { ...process.env, SKIP_DOTENV: '1', GORM_DATABASE_URL: E2E_DSN },
    stdio: ['ignore', 'ignore', 'inherit']
  });

  process.env.E2E_ADMIN_EMAIL = adminEmail;
  process.env.E2E_NOBODY_EMAIL = nobodyEmail;
  process.env.E2E_ADMIN_PASSWORD = password;
}
