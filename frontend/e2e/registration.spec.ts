import { type APIRequestContext, expect, type Page, type Request, test } from '@playwright/test';
import { type IssuedInvitation, invitationPath, issueInvitation, revokeInvitation } from './fixtures/invitations';
import { unexpectedErrors, watchPage } from './fixtures/page-watch';
import { SYNTHETIC_PASSPORT, uniqueRef } from './fixtures/synthetic';

const CREATE_URL = '**/api/v1/guest-registrations';
const PARTIAL = 'You are registering 1 of 2 planned guests';

const INVITE_KEY = 'sakunoie.registration.invite';

/**
 * Opens the host's invitation (en) and fills the 4-step wizard: the stay dates come from the invitation and
 * are read-only; planned 2 (saku-no-ie capacity), a non-Japan representative, and a companion that is added
 * (copying the representative's address) and removed again, so 1 of 2 guests is registered. Stops on review.
 */
const fillWizard = async (page: Page, request: APIRequestContext, bookingReference: string) => {
  const invite = await issueInvitation(request, { bookingReference, note: 'Test Host Note' });
  await page.goto(invitationPath('en', invite.token));
  const checkIn = page.locator('#field-stay-checkInDate');
  const checkOut = page.locator('#field-stay-checkOutDate');
  await expect(checkIn).toHaveValue(invite.checkIn);
  await expect(checkOut).toHaveValue(invite.checkOut);
  await expect(checkIn).toHaveAttribute('readonly', '');
  await expect(checkOut).toHaveAttribute('readonly', '');
  await expect(page.getByText('Your host has set these dates.')).toBeVisible();
  // The fragment is gone from the address bar; the host's note is never shown to the guest.
  expect(await page.evaluate(() => location.href)).not.toContain('#');
  await expect(page.getByText('Test Host Note')).toHaveCount(0);
  await page.locator('#field-stay-plannedGuestCount').fill('2');
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

  // A companion fills the plan (no warning); removing it leaves 1 of 2: partial-registration warning.
  await page.getByRole('button', { name: 'Add a guest' }).click();
  await page.locator('#field-companions-0-fullName').fill('Test Companion');
  await page.getByRole('button', { name: "Use the representative's address and phone" }).click();
  await expect(page.locator('#field-companions-0-address')).toHaveValue('1 Test Street, Testland');
  await page.locator('section').filter({ hasText: 'Guest 1' }).getByRole('radio', { name: 'Yes' }).check();
  await expect(page.getByText(/You are registering/)).toHaveCount(0);
  await page.getByRole('button', { name: 'Remove guest 1' }).click();
  await expect(page.locator('#field-companions-0-fullName')).toHaveCount(0);
  await expect(page.getByText(PARTIAL)).toBeVisible();
  await page.getByRole('button', { name: 'Continue' }).click();

  // Review: warning again, passport never echoed, consents.
  await expect(page.getByText(PARTIAL)).toBeVisible();
  await expect(page.getByText(SYNTHETIC_PASSPORT)).toHaveCount(0);
  await page.getByLabel(/I agree to the Terms/).check();
  await page.getByLabel(/I agree to the Privacy Policy/).check();
  return invite;
};

/** Every API request of the page (read their headers with `allHeaders()` afterwards). */
const apiRequestsOf = (page: Page) => {
  const seen: Request[] = [];
  page.on('request', (r) => {
    if (r.url().includes('/api/v1/')) seen.push(r);
  });
  return seen;
};

test.describe('registration wizard', () => {
  test('lost success response: retry reuses the key; completion shows 1/2 and a fragment-only edit link', async ({
    page,
    context,
    request
  }) => {
    const watch = await watchPage(context, page);
    const apiRequests = apiRequestsOf(page);
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

    const invite = await fillWizard(page, request, uniqueRef('reg'));
    await page.getByRole('button', { name: 'Register' }).click();
    await page.waitForURL('**/en/registration/complete', { waitUntil: 'commit' });
    // The invitation is consumed: it is dropped from this tab.
    expect(await page.evaluate((k) => sessionStorage.getItem(k), INVITE_KEY)).toBeNull();

    expect(keys.length).toBeGreaterThanOrEqual(2);
    expect(new Set(keys).size).toBe(1);
    await expect(page.getByText('1/2 registered')).toBeVisible();
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
    await expect(manage.getByText('1/2 registered')).toBeVisible();

    // The tokens only ever travel in the fragment / Authorization header, never in a request URL.
    for (const url of [...watch.requestUrls, ...manageRequests]) {
      expect(url).not.toContain(token);
      expect(url).not.toContain('token=');
      expect(url).not.toContain(invite.token);
      expect(url).not.toContain('invite=');
    }
    // The invitation goes only to the invitation check and the create; never to another endpoint.
    for (const r of apiRequests) {
      const path = new URL(r.url()).pathname;
      const { authorization } = await r.allHeaders();
      if (path === '/api/v1/invitations/current' || path === '/api/v1/guest-registrations') {
        expect(authorization, path).toBe(`Bearer ${invite.token}`);
      } else {
        expect(authorization, path).toBeUndefined();
      }
    }
    expect(apiRequests.some((r) => new URL(r.url()).pathname === '/api/v1/invitations/current')).toBe(true);

    // Reopening the same invitation: already used, with the edit-link guidance.
    const again = await context.newPage();
    await again.goto(invitationPath('en', invite.token));
    await expect(again.getByText('This invitation link has already been used')).toBeVisible();
    await expect(again.getByText(/use the edit link shown when you registered/)).toBeVisible();
    await expect(again.locator('#field-stay-checkInDate')).toHaveCount(0);
    expect(await again.evaluate((k) => sessionStorage.getItem(k), INVITE_KEY)).toBeNull();
    expect(watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors).filter((e) => !/ERR_FAILED|net::/.test(e))).toEqual([]);
  });

  test('409 after a lost success and an edit: "already submitted", no new key', async ({ page, request }) => {
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

    await fillWizard(page, request, uniqueRef('reg'));
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

  test('server 422 (client validation bypassed) renders a localized summary', async ({ page, request }) => {
    await page.route(CREATE_URL, async (route) => {
      if (route.request().method() !== 'POST') return route.continue();
      // Drop the representative's passport the client required.
      const body = route.request().postDataJSON();
      delete body.guests[0].passport_number;
      await route.continue({ postData: JSON.stringify(body) });
    });
    await fillWizard(page, request, uniqueRef('reg-422'));
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

  test('without an invitation the wizard shows the guidance (ja and en)', async ({ page }) => {
    await page.goto('/ja/stays/saku-no-ie/registration');
    await expect(page.getByText('招待リンクから開いてください', { exact: true })).toBeVisible();
    await expect(page.getByText(/このページは、ホストからお送りした招待リンクから開いてください/)).toBeVisible();
    await expect(page.locator('#field-stay-checkInDate')).toHaveCount(0);

    // A malformed fragment is stripped and counts as no invitation.
    await page.goto('/en/stays/saku-no-ie/registration#invite=short');
    await expect(page.getByText('Please open your invitation link')).toBeVisible();
    expect(await page.evaluate(() => location.href)).not.toContain('#');
  });

  test('a revoked invitation is rejected and dropped from the tab', async ({ page, request }) => {
    const invite: IssuedInvitation = await issueInvitation(request);
    await revokeInvitation(request, invite.publicId);
    const check = page.waitForResponse((r) => r.url().endsWith('/api/v1/invitations/current'));
    await page.goto(invitationPath('en', invite.token));
    const response = await check;
    expect(response.status()).toBe(401);
    expect(response.headers()['cache-control']).toContain('no-store');
    expect((await response.json()).code).toBe('invitation_invalid');
    await expect(page.getByText('This invitation link is not valid')).toBeVisible();
    await expect(page.getByText(/incorrect or has been cancelled/)).toBeVisible();
    expect(await page.evaluate((k) => sessionStorage.getItem(k), INVITE_KEY)).toBeNull();
    await expect(page.locator('#field-stay-checkInDate')).toHaveCount(0);
  });
});
