import { expect, type Request, test } from '@playwright/test';
import { API_URL, newApiContext, WEB_URL } from './fixtures/api-context';
import { type SeededRegistration, seedRegistration } from './fixtures/seed';
import { uniqueRef } from './fixtures/synthetic';

// The E2E build calls the API origin (:8180) from the web origin (:3100), like production behind
// Cloudflare Tunnel, so CORS on the API is exercised by every spec. These checks pin it down.
const WEB_ORIGIN = new URL(WEB_URL).origin;
const API_ORIGIN = new URL(API_URL).origin;

let seeded: SeededRegistration;

test.beforeAll(async ({ playwright }) => {
  const api = await newApiContext(playwright);
  seeded = await seedRegistration(api, { bookingReference: uniqueRef('cors'), plannedGuestCount: 2 });
  await api.dispose();
});

test.describe('cross-origin API', () => {
  test('preflights from the web origin succeed; another origin gets no CORS headers', async ({ request }) => {
    for (const [path, method, headers] of [
      ['/api/v1/guest-registrations', 'POST', 'content-type,idempotency-key'],
      ['/api/v1/guest-registrations/manage', 'PATCH', 'authorization,content-type,if-match'],
      ['/api/v1/admin/facilities', 'GET', 'authorization']
    ]) {
      const res = await request.fetch(`${API_URL}${path}`, {
        method: 'OPTIONS',
        headers: {
          Origin: WEB_ORIGIN,
          'Access-Control-Request-Method': method,
          'Access-Control-Request-Headers': headers
        }
      });
      expect(res.status(), `${method} ${path}`).toBe(204);
      const h = res.headers();
      expect(h['access-control-allow-origin']).toBe(WEB_ORIGIN);
      expect(h['access-control-allow-methods']).toBe('GET, POST, PATCH, OPTIONS');
      expect(h['access-control-allow-headers']).toBe(
        'Authorization, Content-Type, Idempotency-Key, If-Match, X-Request-ID'
      );
      expect(h['access-control-max-age']).toBe('600');
      expect(h['access-control-allow-credentials']).toBeUndefined();
    }

    const foreign = await request.fetch(`${API_URL}/api/v1/guest-registrations/manage`, {
      method: 'OPTIONS',
      headers: { Origin: 'https://evil.example', 'Access-Control-Request-Method': 'PATCH' }
    });
    expect(foreign.status()).toBe(204);
    expect(foreign.headers()['access-control-allow-origin']).toBeUndefined();
    const foreignGet = await request.get(`${API_URL}/api/v1/facilities/saku-no-ie/public`, {
      headers: { Origin: 'https://evil.example' }
    });
    expect(foreignGet.headers()['access-control-allow-origin']).toBeUndefined();
  });

  test('the web origin has no /api rewrite in this build', async ({ request }) => {
    const res = await request.get(`${WEB_URL}/api/v1/facilities/saku-no-ie/public`, { maxRedirects: 0 });
    expect(res.status()).not.toBe(200);
  });

  test('manage calls the API origin with a Bearer token and no Referer; preflighted PATCH works', async ({ page }) => {
    const apiRequests: Request[] = [];
    page.on('request', (r) => {
      if (r.url().includes('/api/')) apiRequests.push(r);
    });
    const summaryResponse = page.waitForResponse(
      (r) => r.url() === `${API_ORIGIN}/api/v1/guest-registrations/manage` && r.request().method() === 'GET'
    );
    await page.goto(`/en/registration/manage#token=${encodeURIComponent(seeded.editToken)}`);
    const summary = await summaryResponse;
    expect(summary.status()).toBe(200);
    expect(summary.headers()['access-control-allow-origin']).toBe(WEB_ORIGIN);
    expect(summary.headers()['access-control-expose-headers']).toContain('ETag');
    await expect(page.getByRole('heading', { name: 'Guests', exact: true })).toBeVisible();

    // A PATCH with Authorization and If-Match is not a simple request: it only reaches the API after
    // the browser's preflight succeeded.
    const patchResponse = page.waitForResponse(
      (r) => r.url() === `${API_ORIGIN}/api/v1/guest-registrations/manage` && r.request().method() === 'PATCH'
    );
    await page.getByRole('button', { name: 'Correct stay details' }).click();
    const stayForm = page.locator('form').filter({ has: page.getByRole('button', { name: 'Save' }) });
    await stayForm.getByLabel(/Estimated arrival/).fill('18:45');
    await stayForm.getByRole('button', { name: 'Save' }).click();
    expect((await patchResponse).status()).toBe(200);
    await expect(page.getByText('18:45')).toBeVisible();

    expect(apiRequests.length).toBeGreaterThanOrEqual(2);
    for (const r of apiRequests) {
      expect(r.url().startsWith(`${API_ORIGIN}/api/v1/`), r.url()).toBe(true);
      expect(r.url()).not.toContain(seeded.editToken);
      const headers = await r.allHeaders();
      expect(headers.origin, r.url()).toBe(WEB_ORIGIN);
      // Referrer-Policy: no-referrer on the manage page also covers its cross-origin API calls.
      expect(headers.referer, r.url()).toBeUndefined();
      expect(headers.authorization).toBe(`Bearer ${seeded.editToken}`);
    }
  });

  test('the registration page sends no Referer; a public page sends only its origin', async ({ page }) => {
    const facilityRequest = page.waitForRequest((r) => r.url().startsWith(`${API_ORIGIN}/api/v1/facilities/`));
    await page.goto('/en/stays/saku-no-ie/registration');
    const registrationHeaders = await (await facilityRequest).allHeaders();
    expect(registrationHeaders.origin).toBe(WEB_ORIGIN);
    expect(registrationHeaders.referer).toBeUndefined();

    // Control: Playwright reports Referer when the browser sends one (strict-origin-when-cross-origin).
    await page.goto('/en/survey');
    const publicRequest = page.waitForRequest((r) => r.url().startsWith(`${API_ORIGIN}/api/v1/facilities/`));
    const status = await page.evaluate(
      async (url) => (await fetch(url)).status,
      `${API_ORIGIN}/api/v1/facilities/saku-no-ie/public`
    );
    expect(status).toBe(200);
    expect((await (await publicRequest).allHeaders()).referer).toBe(`${WEB_ORIGIN}/`);
  });
});
