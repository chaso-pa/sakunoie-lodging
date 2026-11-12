import { expect, test } from '@playwright/test';
import { newApiContext } from './fixtures/api-context';
import { unexpectedErrors, watchPage } from './fixtures/page-watch';
import { type SeededRegistration, seedRegistration } from './fixtures/seed';
import { SYNTHETIC_PASSPORT, uniqueRef } from './fixtures/synthetic';

let seeded: SeededRegistration;

test.beforeAll(async ({ playwright }) => {
  const api = await newApiContext(playwright);
  seeded = await seedRegistration(api, { bookingReference: uniqueRef('manage'), plannedGuestCount: 3 });
  await api.dispose();
});

test.describe('manage (edit link)', () => {
  test('fragment stripped; correct a companion, add a guest, patch the stay; representative read-only', async ({
    page,
    context
  }) => {
    const watch = await watchPage(context, page);
    const summaryResponse = page.waitForResponse(
      (r) => r.url().endsWith('/api/v1/guest-registrations/manage') && r.request().method() === 'GET'
    );
    await page.goto(`/en/registration/manage#token=${encodeURIComponent(seeded.editToken)}`);
    const summary = await summaryResponse;
    expect(summary.status()).toBe(200);
    expect(summary.headers()['cache-control']).toContain('no-store');
    expect(JSON.stringify(await summary.json())).not.toContain(SYNTHETIC_PASSPORT);

    await expect(page.getByRole('heading', { name: 'Guests', exact: true })).toBeVisible();
    expect(await page.evaluate(() => location.href)).not.toContain('#');
    await expect(page.getByText('2/3 registered')).toBeVisible();
    await expect(page.getByText('Passport (last 4): 1234')).toBeVisible();
    await expect(page.getByText("The representative's details cannot be changed")).toBeVisible();
    await expect(page.getByText('000-0000-1234')).toHaveCount(0); // phone is masked

    // Correct the companion's phone (the only "Correct" button: the representative has none).
    await expect(page.getByRole('button', { name: 'Correct', exact: true })).toHaveCount(1);
    await page.getByRole('button', { name: 'Correct', exact: true }).click();
    const guestForm = page.locator('form').filter({ has: page.getByRole('button', { name: 'Save' }) });
    await guestForm.getByLabel('Phone').fill('000-1111-2222');
    await guestForm.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByText('Saved')).toBeVisible();

    // Add the third guest: 3/3.
    await page.locator('#field-guests-0-fullName').fill('Test Companion Two');
    await page.locator('#field-guests-0-address').fill('2 Test Street, Testland');
    await page.locator('section').filter({ hasText: 'Add guests' }).getByRole('radio', { name: 'Yes' }).check();
    await page.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(page.getByText('3/3 registered')).toBeVisible();
    await expect(page.getByText('All planned guests are registered')).toBeVisible();

    // Stay correction (If-Match path).
    await page.getByRole('button', { name: 'Correct stay details' }).click();
    const stayForm = page.locator('form').filter({ has: page.getByRole('button', { name: 'Save' }) });
    await stayForm.getByLabel(/Estimated arrival/).fill('17:30');
    await stayForm.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByText('17:30')).toBeVisible();

    for (const url of watch.requestUrls) expect(url).not.toContain(seeded.editToken);
    expect(watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors)).toEqual([]);
  });

  test('a bad token shows recovery guidance (host reissue)', async ({ page }) => {
    await page.goto('/ja/registration/manage#token=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
    await expect(page.getByText('リンクが無効です')).toBeVisible();
    await expect(page.getByText(/ホストに新しいリンクの発行を依頼してください/)).toBeVisible();
    expect(await page.evaluate(() => location.href)).not.toContain('#');
  });

  test('an expired token (410) shows the expired guidance and drops the token (AC6)', async ({ page }) => {
    // The token cannot expire in real time during a run, so the API answer is replaced with the
    // server's 410 envelope (Go clock-injection tests cover the expiry itself).
    let calls = 0;
    await page.route('**/api/v1/guest-registrations/manage', (route) => {
      calls++;
      return route.fulfill({
        status: 410,
        contentType: 'application/json',
        headers: { 'cache-control': 'no-store' },
        body: JSON.stringify({
          code: 'token_expired',
          message: 'edit token expired',
          details: [],
          request_id: '00000000-0000-4000-8000-000000000410'
        })
      });
    });
    await page.goto(`/en/registration/manage#token=${encodeURIComponent(seeded.editToken)}`);
    const alert = page.getByRole('alert').filter({ hasText: 'This link has expired' });
    await expect(alert).toBeVisible();
    await expect(alert).toContainText('Contact the host if something needs correcting');
    expect(calls).toBeGreaterThan(0);
    expect(await page.evaluate(() => location.href)).not.toContain('#');
    expect(await page.evaluate(() => sessionStorage.getItem('sakunoie.manage.token'))).toBeNull();
    await expect(page.getByRole('heading', { name: 'Guests', exact: true })).toHaveCount(0);
  });

  test('no token in this tab shows the session-lost guidance', async ({ page }) => {
    await page.goto('/en/registration/manage');
    await expect(page.getByText('Edit link not found')).toBeVisible();
    await expect(page.getByText(/ask the host to reissue it/)).toBeVisible();
  });
});
