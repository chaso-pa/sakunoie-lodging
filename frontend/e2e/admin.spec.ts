import fs from 'node:fs';
import { expect, test } from '@playwright/test';
import { adminLogin, fillLogin, unauthorizedCredentials } from './fixtures/admin-login';
import { newApiContext } from './fixtures/api-context';
import { unexpectedErrors, watchPage } from './fixtures/page-watch';
import { type SeededRegistration, seedRegistration } from './fixtures/seed';
import { SYNTHETIC_PASSPORT, uniqueRef, uniqueStay } from './fixtures/synthetic';

let rowA: SeededRegistration;
let rowB: SeededRegistration;

test.beforeAll(async ({ playwright }, testInfo) => {
  const api = await newApiContext(playwright, testInfo);
  const stay = uniqueStay();
  // A: non-Japan representative with a synthetic passport; 2 of 3 guests registered.
  rowA = await seedRegistration(api, { bookingReference: uniqueRef('admin'), ...stay, plannedGuestCount: 3 });
  // B: formula-like booking reference that the CSV must neutralize.
  rowB = await seedRegistration(api, {
    bookingReference: '=SUM(1)',
    ...stay,
    plannedGuestCount: 1,
    guests: [
      { is_representative: true, full_name: 'Test Formula Rep', address: '3 Test Street', has_japan_address: true }
    ]
  });
  await api.dispose();
});

const slash = (date: string) => date.replace(/-/g, '/');

test.describe('admin console', () => {
  test('CSP on /admin/login allows the Auth Emulator; zero CSP violations; signed-out redirect', async ({
    page,
    context
  }) => {
    const watch = await watchPage(context, page);
    await page.goto('/admin');
    await page.waitForURL('**/admin/login', { waitUntil: 'commit' });
    const response = await page.goto('/admin/login');
    const csp = response?.headers()['content-security-policy'] ?? '';
    expect(csp).toMatch(/connect-src 'self' http:\/\/127\.0\.0\.1:9099(;|$)/);
    expect(csp).toContain("frame-ancestors 'none'");
    expect(csp).toContain("object-src 'none'");
    expect(response?.headers()['referrer-policy']).toBe('no-referrer');
    expect(response?.headers()['x-robots-tag']).toContain('noindex');
    await expect(page.locator('html')).toHaveAttribute('lang', 'ja');
    await expect(page.getByRole('button', { name: 'ログイン' })).toBeVisible();
    await page.waitForLoadState('networkidle');
    expect(watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors)).toEqual([]);
  });

  test('an emulator user without a grant sees "not authorized"', async ({ page }) => {
    const { email, password } = unauthorizedCredentials();
    await fillLogin(page, email, password);
    await expect(page.getByRole('heading', { name: '権限がありません' })).toBeVisible();
  });

  test('ledger, detail (last4 only), verify, correct, reissue and CSV export', async ({ page, context }) => {
    const watch = await watchPage(context, page);
    await adminLogin(page);

    // Ledger search by booking reference.
    await page.getByLabel('氏名・予約番号で検索').fill(rowA.bookingReference);
    await page.getByRole('button', { name: '検索' }).click();
    await page.waitForURL(/q=E2E-admin-/, { waitUntil: 'commit' });
    const row = page.getByRole('row').filter({ hasText: rowA.bookingReference });
    await expect(row).toBeVisible();
    await expect(row.getByText('未登録あり 残り1名')).toBeVisible();
    await expect(row.getByText('本人確認待ち')).toBeVisible();
    await expect(row.getByRole('cell', { name: 'Test Rep', exact: true })).toBeVisible();

    // Detail: passport last4 only, contact data masked.
    await row.getByRole('link', { name: /詳細/ }).click();
    await page.waitForURL(`**/admin/guest-registrations/${rowA.publicId}`, { waitUntil: 'commit' });
    await expect(page.getByRole('heading', { name: `${slash(rowA.checkIn)} – ${slash(rowA.checkOut)}` })).toBeVisible();
    await expect(page.getByText('****1234')).toBeVisible();
    await expect(page.getByText(SYNTHETIC_PASSPORT)).toHaveCount(0);
    await expect(page.getByText('000-0000-1234')).toHaveCount(0);

    // Verify identity.
    await page.getByRole('button', { name: '本人確認済みにする' }).click();
    await page.getByRole('dialog').getByRole('button', { name: '本人確認済みにする' }).click();
    await expect(page.getByText('本人確認を記録しました。')).toBeVisible();
    await expect(page.getByText('確認済み', { exact: true }).first()).toBeVisible();

    // Correction (audited by field name server-side).
    await page.getByRole('button', { name: '訂正する' }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByLabel('予約経路').fill('direct');
    await dialog.getByRole('button', { name: '訂正を保存' }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText('direct', { exact: true })).toBeVisible();

    // Reissue: the seeded token is revoked (401), the new one works.
    await page.getByRole('button', { name: '編集リンクを再発行' }).click();
    await page.getByRole('button', { name: '再発行する' }).click();
    const linkInput = page.getByLabel('編集リンク', { exact: true });
    await expect(linkInput).toBeVisible();
    const newUrl = await linkInput.inputValue();
    expect(newUrl).toMatch(/\/ja\/registration\/manage#token=/);
    const newToken = decodeURIComponent(newUrl.split('#token=')[1]);
    const oldRes = await page.request.get('/api/v1/guest-registrations/manage', {
      headers: { Authorization: `Bearer ${rowA.editToken}` }
    });
    const newRes = await page.request.get('/api/v1/guest-registrations/manage', {
      headers: { Authorization: `Bearer ${newToken}` }
    });
    expect(oldRes.status()).toBe(401);
    expect(newRes.status()).toBe(200);
    await page.getByRole('button', { name: '閉じる' }).click();

    // CSV filtered to this run's check-in date.
    await page.getByRole('link', { name: '宿泊者名簿に戻る' }).click();
    await page.waitForURL('**/admin/guest-registrations**', { waitUntil: 'commit' });
    await page.getByLabel('チェックイン（から）').fill(rowA.checkIn);
    await page.getByLabel('チェックイン（まで）').fill(rowA.checkIn);
    await page.getByLabel('氏名・予約番号で検索').fill('');
    await page.getByRole('button', { name: '検索' }).click();
    await page.waitForURL(new RegExp(`from=${rowA.checkIn}`), { waitUntil: 'commit' });
    // Wait for the filtered result (older runs' =SUM(1) rows are on other dates).
    await expect(page.getByRole('row').filter({ hasText: rowB.bookingReference })).toHaveCount(1);
    await expect(page.getByRole('row').filter({ hasText: rowA.bookingReference })).toHaveCount(1);
    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByRole('button', { name: 'CSV出力' }).click()
    ]);
    expect(download.suggestedFilename()).toBe(`registrations-saku-no-ie-${rowA.checkIn}-${rowA.checkIn}.csv`);
    const buf = fs.readFileSync(await download.path());
    expect([...buf.subarray(0, 3)]).toEqual([0xef, 0xbb, 0xbf]);
    const csv = buf.toString('utf8').slice(1);
    expect(csv.startsWith('facility_code,check_in_date')).toBe(true);
    expect(csv).toContain(rowA.bookingReference);
    expect(csv).toContain(",'=SUM(1),");
    expect(csv).not.toContain(',=SUM(1),');
    expect(csv).not.toContain(SYNTHETIC_PASSPORT);
    expect(csv).toContain(',1234\r\n');
    await expect(page.getByText(/をダウンロードしました/)).toBeVisible();

    for (const url of watch.requestUrls) {
      expect(url).not.toContain(rowA.editToken);
      expect(url).not.toContain(newToken);
    }
    expect(watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors)).toEqual([]);
  });
});
