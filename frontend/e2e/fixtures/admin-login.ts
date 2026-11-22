import { expect, type Page } from '@playwright/test';

// Created by global-setup.ts in the Auth Emulator; the password is random per run and lives only in env.
export const adminCredentials = () => {
  const email = process.env.E2E_ADMIN_EMAIL;
  const password = process.env.E2E_ADMIN_PASSWORD;
  if (!email || !password) throw new Error('E2E admin user missing: global-setup.ts did not run');
  return { email, password };
};

export const unauthorizedCredentials = () => {
  const email = process.env.E2E_NOBODY_EMAIL;
  const password = process.env.E2E_ADMIN_PASSWORD;
  if (!email || !password) throw new Error('E2E users missing: global-setup.ts did not run');
  return { email, password };
};

export const fillLogin = async (page: Page, email: string, password: string) => {
  await page.goto('/admin/login');
  await page.getByLabel('メールアドレス').fill(email);
  await page.getByLabel('パスワード').fill(password);
  await page.getByRole('button', { name: 'ログイン' }).click();
};

/** Signs in as the granted owner and waits for the ledger. */
export const adminLogin = async (page: Page) => {
  const { email, password } = adminCredentials();
  await fillLogin(page, email, password);
  await page.waitForURL('**/admin/guest-registrations', { waitUntil: 'commit' });
  await expect(page.getByRole('heading', { name: '宿泊者名簿' })).toBeVisible();
};
