import fs from 'node:fs';
import { expect, test } from '@playwright/test';
import { adminLogin, fillLogin, unauthorizedCredentials } from './fixtures/admin-login';
import { API_URL, newApiContext } from './fixtures/api-context';
import { copiedTexts, simulateInsecureClipboard } from './fixtures/insecure-clipboard';
import { unexpectedErrors, waitForHydrated, watchPage } from './fixtures/page-watch';
import { defaultGuests, type SeededRegistration, seedRegistration } from './fixtures/seed';
import { SYNTHETIC_PASSPORT, uniqueRef, uniqueStay } from './fixtures/synthetic';

let rowA: SeededRegistration;
let rowB: SeededRegistration;

test.beforeAll(async ({ playwright }) => {
  const api = await newApiContext(playwright);
  const stay = uniqueStay();
  // A: non-Japan representative with a synthetic passport; 1 of 2 guests registered (capacity 2).
  rowA = await seedRegistration(api, {
    bookingReference: uniqueRef('admin'),
    ...stay,
    plannedGuestCount: 2,
    guests: defaultGuests().slice(0, 1)
  });
  // B: formula-like booking reference that the CSV must neutralize.
  rowB = await seedRegistration(api, {
    bookingReference: '=SUM(1)',
    ...stay,
    plannedGuestCount: 1,
    guests: [
      {
        is_representative: true,
        full_name: 'Test Formula Rep',
        name_kana: 'テスト フォーミュラ',
        address: '3 Test Street',
        has_japan_address: true,
        phone: '000-0000-5678',
        email: 'test-formula-rep@example.test'
      }
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
    expect(csp).toContain(`connect-src 'self' ${API_URL} http://127.0.0.1:9099;`);
    expect(csp).toContain("frame-ancestors 'none'");
    expect(csp).toContain("object-src 'none'");
    expect(response?.headers()['referrer-policy']).toBe('no-referrer');
    expect(response?.headers()['x-robots-tag']).toContain('noindex');
    await expect(page.locator('html')).toHaveAttribute('lang', 'ja');
    await expect(page.getByRole('button', { name: 'ログイン' })).toBeVisible();
    // The login form has hydrated (Firebase Auth initialized); violations are reported asynchronously.
    await waitForHydrated(page, 'form button[type=submit]');
    await expect.poll(() => watch.cspViolations).toEqual([]);
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
    await expect(page.getByRole('main').getByText('確認済み', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '本人確認済み', exact: true })).toBeDisabled();

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
    const oldRes = await page.request.get(`${API_URL}/api/v1/guest-registrations/manage`, {
      headers: { Authorization: `Bearer ${rowA.editToken}` }
    });
    const newRes = await page.request.get(`${API_URL}/api/v1/guest-registrations/manage`, {
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

  test('invitations: create shows the URL once, the list shows the status, revoke with confirmation', async ({
    page,
    context
  }) => {
    const watch = await watchPage(context, page);
    await adminLogin(page);
    await page.getByRole('navigation', { name: '管理メニュー' }).getByRole('link', { name: '招待リンク' }).click();
    await page.waitForURL('**/admin/invitations', { waitUntil: 'commit' });
    await expect(page.getByRole('heading', { name: '招待リンク', level: 1 })).toBeVisible();

    const stay = uniqueStay();
    const note = `E2E guest ${uniqueRef('inv')}`;
    const form = page.locator('form').filter({ has: page.getByRole('button', { name: '招待リンクを作成' }) });
    await form.getByLabel('チェックイン日').fill(stay.checkIn);
    await form.getByLabel('チェックアウト日').fill(stay.checkOut);
    await form.getByLabel('予約コード（任意）').fill(uniqueRef('inv-code'));
    await form.getByLabel('メモ（任意）').fill(note);
    const [created] = await Promise.all([
      page.waitForResponse((r) => r.url().endsWith('/api/v1/admin/invitations') && r.request().method() === 'POST'),
      form.getByRole('button', { name: '招待リンクを作成' }).click()
    ]);
    expect(created.status()).toBe(201);
    expect(created.headers()['cache-control']).toContain('no-store');

    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText('このリンクは今だけ表示されます')).toBeVisible();
    await expect(dialog.getByText(/Airbnb のメッセージ/)).toBeVisible();
    const url = await dialog.getByLabel('招待リンク').inputValue();
    expect(url).toMatch(/\/ja\/stays\/saku-no-ie\/registration#invite=[A-Za-z0-9_-]{40,}$/);
    await dialog.getByLabel('リンクの表示言語').selectOption('en');
    const enUrl = await dialog.getByLabel('招待リンク').inputValue();
    expect(enUrl).toMatch(/\/en\/stays\/saku-no-ie\/registration#invite=/);
    // Copy works without a secure context too (plain http on the LAN): the execCommand fallback.
    await simulateInsecureClipboard(page);
    await dialog.getByRole('button', { name: 'リンクをコピー' }).click();
    await expect(dialog.getByRole('button', { name: 'コピーしました' })).toBeVisible();
    expect(await copiedTexts(page)).toEqual([enUrl]);
    const token = decodeURIComponent(url.split('#invite=')[1]);
    await dialog.getByRole('button', { name: '閉じる' }).click();
    await expect(dialog).toBeHidden();
    // Shown once: the page no longer has the token anywhere.
    expect(await page.content()).not.toContain(token);

    // The list shows it as active; the guest-facing check sees only the dates.
    const row = page.getByRole('row').filter({ hasText: note });
    await expect(row.getByText('有効')).toBeVisible();
    const check = await page.request.get(`${API_URL}/api/v1/invitations/current`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(check.status()).toBe(200);
    const current = await check.json();
    expect(current).toMatchObject({
      facility_code: 'saku-no-ie',
      check_in_date: stay.checkIn,
      check_out_date: stay.checkOut
    });
    expect(JSON.stringify(current)).not.toContain(note);

    // Revoke: confirmation first, then the guest gets 401 invitation_invalid.
    await row.getByRole('button', { name: /取り消す/ }).click();
    const confirm = page.getByRole('dialog');
    await expect(confirm.getByText('招待リンクを取り消しますか')).toBeVisible();
    await confirm.getByRole('button', { name: '取り消す', exact: true }).click();
    await expect(page.getByText('招待リンクを取り消しました。')).toBeVisible();
    await expect(row.getByText('取り消し済み')).toBeVisible();
    await expect(row.getByRole('button', { name: /取り消す/ })).toHaveCount(0);
    const revoked = await page.request.get(`${API_URL}/api/v1/invitations/current`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(revoked.status()).toBe(401);
    expect((await revoked.json()).code).toBe('invitation_invalid');

    for (const u of watch.requestUrls) expect(u).not.toContain(token);
    expect(watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors)).toEqual([]);
  });
});
