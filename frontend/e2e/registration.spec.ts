import { expect, type Page, test } from '@playwright/test';
import { unexpectedErrors, watchPage } from './fixtures/page-watch';
import { SYNTHETIC_PASSPORT, uniqueRef, uniqueStay } from './fixtures/synthetic';

const CREATE_URL = '**/api/v1/guest-registrations';
const PARTIAL = 'You are registering 2 of 3 planned guests';

/** Fills the 4-step wizard (en): planned 3, a non-Japan representative and one companion. Stops on the review step. */
const fillWizard = async (page: Page, bookingReference: string) => {
  const stay = uniqueStay();
  await page.goto('/en/stays/saku-no-ie/registration');
  await page.locator('#field-stay-checkInDate').fill(stay.checkIn);
  await page.locator('#field-stay-checkOutDate').fill(stay.checkOut);
  await page.locator('#field-stay-plannedGuestCount').fill('3');
  await page.locator('#field-stay-bookingReference').fill(bookingReference);
  await page.getByRole('button', { name: 'Continue' }).click();

  // Representative without a Japanese address: client validation requires nationality + passport.
  await page.locator('#field-representative-fullName').fill('Test Rep');
  await page.locator('#field-representative-address').fill('1 Test Street, Testland');
  await page.getByRole('radio', { name: 'No' }).check();
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.getByRole('alert').filter({ hasText: 'Please review your entries' })).toBeVisible();
  await expect(page.locator('#field-representative-passportNumber')).toBeVisible();
  await page.locator('#field-representative-nationality').fill('Testland');
  await page.locator('#field-representative-passportNumber').fill(SYNTHETIC_PASSPORT);
  await page.getByRole('button', { name: 'Continue' }).click();

  // One companion (planned 3): partial-registration warning.
  await page.getByRole('button', { name: 'Add a guest' }).click();
  await page.locator('#field-companions-0-fullName').fill('Test Companion');
  await page.getByRole('button', { name: "Use the representative's address and phone" }).click();
  await expect(page.locator('#field-companions-0-address')).toHaveValue('1 Test Street, Testland');
  await page.locator('section').filter({ hasText: 'Guest 1' }).getByRole('radio', { name: 'Yes' }).check();
  await expect(page.getByText(PARTIAL)).toBeVisible();
  await page.getByRole('button', { name: 'Continue' }).click();

  // Review: warning again, passport never echoed, consents.
  await expect(page.getByText(PARTIAL)).toBeVisible();
  await expect(page.getByText(SYNTHETIC_PASSPORT)).toHaveCount(0);
  await page.getByLabel(/I agree to the Terms/).check();
  await page.getByLabel(/I agree to the Privacy Policy/).check();
};

test.describe('registration wizard', () => {
  test('lost success response: retry reuses the key; completion shows 2/3 and a fragment-only edit link', async ({
    page,
    context
  }) => {
    const watch = await watchPage(context, page);
    const keys: string[] = [];
    let first = true;
    await page.route(CREATE_URL, async (route) => {
      if (route.request().method() !== 'POST') return route.continue();
      keys.push(route.request().headers()['idempotency-key'] ?? '');
      if (first) {
        // The server commits, but the browser never sees the response.
        first = false;
        await route.fetch();
        await route.abort('failed');
        return;
      }
      await route.continue();
    });

    await fillWizard(page, uniqueRef('reg'));
    await page.getByRole('button', { name: 'Register' }).click();
    await page.waitForURL('**/en/registration/complete', { waitUntil: 'commit' });

    expect(keys.length).toBeGreaterThanOrEqual(2);
    expect(new Set(keys).size).toBe(1);
    await expect(page.getByText('2/3 registered')).toBeVisible();
    const editUrl = await page.getByLabel('Edit link (URL)').inputValue();
    expect(editUrl).toMatch(/\/en\/registration\/manage#token=[A-Za-z0-9_-]{20,}$/);
    const token = decodeURIComponent(editUrl.split('#token=')[1]);

    // The replayed link works: manage loads with it and strips the fragment.
    const manage = await context.newPage();
    const manageRequests: string[] = [];
    manage.on('request', (r) => manageRequests.push(r.url()));
    await manage.goto(editUrl);
    await expect(manage.getByRole('heading', { name: 'Guests', exact: true })).toBeVisible();
    expect(manage.url()).not.toContain('#');
    await expect(manage.getByText('2/3 registered')).toBeVisible();

    // The token only ever travels in the fragment / Authorization header, never in a request URL.
    for (const url of [...watch.requestUrls, ...manageRequests]) {
      expect(url).not.toContain(token);
      expect(url).not.toContain('token=');
    }
    expect(watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors).filter((e) => !/ERR_FAILED|net::/.test(e))).toEqual([]);
  });

  test('409 after a lost success and an edit: "already submitted", no new key', async ({ page }) => {
    const keys: string[] = [];
    let attempts = 0;
    await page.route(CREATE_URL, async (route) => {
      if (route.request().method() !== 'POST') return route.continue();
      keys.push(route.request().headers()['idempotency-key'] ?? '');
      attempts++;
      if (attempts === 1) {
        await route.fetch(); // committed server-side, response lost
        await route.abort('failed');
        return;
      }
      if (attempts <= 3) {
        await route.abort('failed'); // the client's automatic retries fail too
        return;
      }
      await route.continue();
    });

    await fillWizard(page, uniqueRef('reg'));
    await page.getByRole('button', { name: 'Register' }).click();
    await expect(page.getByText(/Could not connect/)).toBeVisible({ timeout: 20_000 });
    expect(attempts).toBe(3);

    // The guest changes something and submits again: same key, different body.
    for (let i = 0; i < 3; i++) await page.getByRole('button', { name: 'Back' }).click();
    await page.locator('#field-stay-bookingReference').fill(uniqueRef('reg-edited'));
    for (let i = 0; i < 3; i++) await page.getByRole('button', { name: 'Continue' }).click();
    const [response] = await Promise.all([
      page.waitForResponse((r) => r.url().endsWith('/api/v1/guest-registrations') && r.request().method() === 'POST'),
      page.getByRole('button', { name: 'Register' }).click()
    ]);
    expect(response.status()).toBe(409);
    expect((await response.json()).code).toBe('idempotency_conflict');
    await expect(page.getByText('This registration was already submitted')).toBeVisible();
    await expect(page.getByText(/please contact the host/)).toBeVisible();
    expect(keys).toHaveLength(4);
    expect(new Set(keys).size).toBe(1);
  });

  test('server 422 (client validation bypassed) renders a localized summary', async ({ page }) => {
    await page.route(CREATE_URL, async (route) => {
      if (route.request().method() !== 'POST') return route.continue();
      // Drop the representative's passport the client required.
      const body = route.request().postDataJSON();
      delete body.guests[0].passport_number;
      await route.continue({ postData: JSON.stringify(body) });
    });
    await fillWizard(page, uniqueRef('reg-422'));
    const [response] = await Promise.all([
      page.waitForResponse((r) => r.url().endsWith('/api/v1/guest-registrations') && r.request().method() === 'POST'),
      page.getByRole('button', { name: 'Register' }).click()
    ]);
    expect(response.status()).toBe(422);
    const envelope = await response.json();
    expect(envelope.code).toBe('validation_error');
    expect(JSON.stringify(envelope)).not.toContain(SYNTHETIC_PASSPORT);
    await expect(page.getByRole('alert').filter({ hasText: 'Please review your entries' })).toBeVisible();
    await expect(page.locator('#field-representative-passportNumber')).toBeVisible();
    await expect(page.getByText('Please check this field').first()).toBeVisible();
  });
});
