import { defineConfig } from '@playwright/test';

// The stack (compose MariaDB, Auth Emulator, API :8180, Next standalone :3100) is started by
// `scripts/e2e-stack.sh start`, not by a Playwright webServer, so its PIDs are tracked explicitly.
// Specs share one DB and one emulator: run serially. Each spec seeds its own data and passes alone.
export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',
  workers: 1,
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  timeout: 60_000,
  expect: { timeout: 15_000 },
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://127.0.0.1:3100',
    browserName: 'chromium',
    // Traces and screenshots may hold synthetic tokens; /test-results and /playwright-report are git-ignored.
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure'
  },
  projects: [
    { name: 'desktop-chromium', use: { viewport: { width: 1280, height: 900 } } },
    {
      name: 'mobile-chromium',
      testMatch: /registration\.spec\.ts/,
      use: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 3 }
    }
  ]
});
