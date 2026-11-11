import facility from '@content/facilities/saku-no-ie.json';
import { expect, type Page, test } from '@playwright/test';
import { unexpectedErrors, watchPage } from './fixtures/page-watch';

/** Waits until React has hydrated the language switcher (its DOM node carries React props). */
const waitForHydratedSwitcher = (page: Page) =>
  page.waitForFunction(() => {
    const select = document.querySelector('header select');
    return !!select && Object.keys(select).some((k) => k.startsWith('__reactProps'));
  });

const LOCALES = ['ja', 'en', 'zh-CN', 'ko'] as const;
const PAGES = ['', '/facilities', '/guide', '/faq', '/rates', '/terms', '/privacy', '/survey'] as const;

test.describe('public site', () => {
  test('8 pages x 4 locales: 200, html lang, one localized h1, placeholder banner', async ({ page }) => {
    const h1ByPage = new Map<string, Set<string>>();
    for (const locale of LOCALES) {
      for (const p of PAGES) {
        const response = await page.goto(`/${locale}${p}`);
        expect(response?.status(), `/${locale}${p}`).toBe(200);
        await expect(page.locator('html')).toHaveAttribute('lang', locale);
        const h1 = page.locator('h1');
        await expect(h1).toHaveCount(1);
        const text = (await h1.innerText()).trim();
        expect(text, `/${locale}${p} h1`).not.toBe('');
        h1ByPage.set(p, (h1ByPage.get(p) ?? new Set()).add(text));
        if (p !== '/survey') {
          await expect(page.getByText('要差し替え / Placeholder content').first()).toBeVisible();
        }
      }
    }
    // Localized: each page has a different h1 in each of the four locales.
    for (const [p, texts] of h1ByPage) expect(texts.size, `h1 of ${p || '/'} is localized`).toBe(LOCALES.length);
  });

  test('CSP header on /ja and zero CSP violations; hydration works', async ({ page, context }) => {
    const watch = await watchPage(context, page);
    const response = await page.goto('/ja');
    const csp = response?.headers()['content-security-policy'] ?? '';
    expect(csp).toContain("frame-ancestors 'none'");
    expect(csp).toContain("object-src 'none'");
    expect(csp).toContain("base-uri 'none'");
    expect(csp).toMatch(/connect-src 'self'(;|$)/);
    expect(csp).toMatch(/script-src 'self' 'nonce-[^']+' 'strict-dynamic'(;|$)/);
    expect(csp).not.toMatch(/googleapis|127\.0\.0\.1:9099|unsafe-eval/);
    expect(response?.headers()['x-content-type-options']).toBe('nosniff');

    // Hydration under the nonce CSP: React attaches to the server HTML and a client-side control works.
    await waitForHydratedSwitcher(page);
    await page.getByLabel('言語').selectOption('en');
    await page.waitForURL('**/en', { waitUntil: 'commit' });
    await expect(page.locator('html')).toHaveAttribute('lang', 'en');
    await page.waitForLoadState('networkidle');
    expect(watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors)).toEqual([]);
  });

  test('locale switcher keeps the path', async ({ page }) => {
    await page.goto('/en/faq');
    await waitForHydratedSwitcher(page);
    await page.getByLabel('Language').selectOption('ko');
    await page.waitForURL('**/ko/faq', { waitUntil: 'commit' });
    await expect(page.locator('html')).toHaveAttribute('lang', 'ko');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('Airbnb CTA is an external link to the content URL', async ({ page }) => {
    await page.goto('/en');
    const cta = page.locator('header').getByRole('link', { name: /Book on Airbnb/ });
    await expect(cta).toHaveAttribute('target', '_blank');
    await expect(cta).toHaveAttribute('rel', /noopener/);
    await expect(cta).toHaveAttribute('href', facility.airbnbUrl);
  });

  test('unknown localized path renders the localized 404', async ({ page }) => {
    const ja = await page.goto('/ja/does-not-exist');
    expect(ja?.status()).toBe(404);
    await expect(page.getByRole('heading', { level: 1, name: 'ページが見つかりません' })).toBeVisible();
    const en = await page.goto('/en/does-not-exist');
    expect(en?.status()).toBe(404);
    await expect(page.getByRole('heading', { level: 1, name: 'Page not found' })).toBeVisible();
  });
});
