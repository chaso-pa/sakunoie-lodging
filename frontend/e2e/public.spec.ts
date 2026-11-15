import { readFileSync } from 'node:fs';
import facility from '@content/facilities/saku-no-ie.json';
import { expect, type Page, test } from '@playwright/test';
import matter from 'gray-matter';
import { API_URL } from './fixtures/api-context';
import { unexpectedErrors, waitForHydrated, watchPage } from './fixtures/page-watch';

/** Waits until React has hydrated the language switcher. */
const waitForHydratedSwitcher = (page: Page) => waitForHydrated(page, 'header [data-locale-switcher]');

/** Opens the language menu and picks a language by its native name. */
const switchLocale = async (page: Page, label: RegExp, language: string) => {
  await page.locator('header').getByRole('button', { name: label }).click();
  await page.getByRole('menuitem', { name: language }).click();
};

/** Whether `content/<locale>/<page>.md` still says `placeholder: true` (the page then shows the banner). */
const isPlaceholder = (locale: string, p: string) =>
  matter(readFileSync(`content/${locale}/${p.slice(1) || 'home'}.md`, 'utf8')).data.placeholder === true;

const LOCALES = ['ja', 'en', 'zh-CN', 'ko'] as const;
const PAGES = ['', '/facilities', '/guide', '/faq', '/rates', '/terms', '/privacy', '/survey'] as const;

test.describe('public site', () => {
  test('8 pages x 4 locales: 200, html lang, one localized h1, banner only on placeholder pages', async ({ page }) => {
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
          const banner = page.getByText('要差し替え / Placeholder content');
          if (isPlaceholder(locale, p)) await expect(banner.first()).toBeVisible();
          else await expect(banner).toHaveCount(0);
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
    // The browser calls the API origin directly; Firebase origins appear on admin pages only.
    expect(csp).toContain(`connect-src 'self' ${API_URL};`);
    expect(csp).toMatch(/script-src 'self' 'nonce-[^']+' 'strict-dynamic'(;|$)/);
    expect(csp).not.toMatch(/googleapis|127\.0\.0\.1:9099|unsafe-eval/);
    // The Airbnb listing iframe (no SDK script): frame-src only, script-src untouched.
    expect(csp).toContain('frame-src https://www.airbnb.com https://www.airbnb.jp;');
    expect(csp).not.toMatch(/script-src[^;]*airbnb/);
    expect(response?.headers()['x-content-type-options']).toBe('nosniff');

    // Hydration under the nonce CSP: React attaches to the server HTML and a client-side control works.
    await waitForHydratedSwitcher(page);
    await switchLocale(page, /^言語/, 'English');
    await page.waitForURL('**/en', { waitUntil: 'commit' });
    await expect(page.locator('html')).toHaveAttribute('lang', 'en');
    // The /en page has hydrated (its scripts ran); violations are reported asynchronously.
    await waitForHydratedSwitcher(page);
    await expect.poll(() => watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors)).toEqual([]);
  });

  test('frame-src for Airbnb only on the home pages, never on registration, other pages or admin', async ({ page }) => {
    for (const locale of LOCALES) {
      const csp = (await page.goto(`/${locale}`))?.headers()['content-security-policy'] ?? '';
      expect(csp, `/${locale}`).toContain('frame-src https://www.airbnb.com https://www.airbnb.jp');
    }
    for (const p of [
      '/ja/stays/saku-no-ie/registration',
      '/en/registration/manage',
      '/ja/facilities',
      '/admin/login'
    ]) {
      const csp = (await page.goto(p))?.headers()['content-security-policy'] ?? '';
      expect(csp, p).toContain("default-src 'self'");
      expect(csp, p).not.toContain('frame-src');
      expect(csp, p).not.toContain('airbnb');
    }
  });

  test('home: Airbnb listing iframe without the SDK, with a fallback link', async ({ page }) => {
    await page.goto('/ja');
    const section = page.locator('section[aria-labelledby="airbnb-title"]');
    await section.scrollIntoViewIfNeeded();
    const frame = section.locator('iframe');
    await expect(frame).toHaveAttribute('title', '佐久の家の Airbnb 掲載ページ（写真・レビュー）');
    await expect(frame).toHaveAttribute('loading', 'lazy');
    await expect(frame).toHaveAttribute(
      'src',
      /^https:\/\/www\.airbnb\.jp\/embeddable\/home\?externalPageUrl=[^&]+&hide-price=true&id=1668838036170160414#/
    );
    // No Airbnb script on our page.
    await expect(page.locator('script[src*="airbnb"]')).toHaveCount(0);
    const fallback = section.getByRole('link', { name: /Airbnbで見る/ });
    await expect(fallback).toHaveAttribute('href', facility.airbnbUrl);
    await expect(fallback).toHaveAttribute('target', '_blank');
    // The single primary booking CTA of the page body is in the booking card; guest links stay text links.
    await expect(page.locator('#booking').getByRole('link', { name: /宿泊を予約/ })).toBeVisible();
    const guests = page.locator('aside[aria-labelledby="guests-title"]');
    await expect(guests.getByRole('link', { name: '宿泊者登録' })).toHaveAttribute(
      'href',
      '/ja/stays/saku-no-ie/registration'
    );
    await expect(guests.getByRole('link', { name: '滞在後アンケート' })).toHaveAttribute('href', '/ja/survey');
  });

  test('locale switcher keeps the path', async ({ page }) => {
    await page.goto('/en/faq');
    await waitForHydratedSwitcher(page);
    await switchLocale(page, /^Language/, '한국어');
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

  test('real photos load with alt text and no placeholder badge; gallery counts every photo', async ({ page }) => {
    const gallery = facility.photos.filter((p) => p.id !== 'hero');
    const heroSlides = ['room', 'shared', 'dining'].map((id) => facility.photos.find((p) => p.id === id));
    const loaded = (img: HTMLImageElement) => img.complete && img.naturalWidth > 0;

    await page.goto('/ja');
    // Hero slideshow: the first slide is exposed; dots switch slides and pause toggles autoplay.
    const hero = page.locator('section[aria-labelledby="hero-title"]');
    await waitForHydrated(page, 'section[aria-labelledby="hero-title"] button');
    await expect(hero.getByRole('img', { name: heroSlides[0]?.alt.ja })).toBeVisible();
    await hero.getByRole('button', { name: 'スライドショーを一時停止' }).click();
    await expect(hero.getByRole('button', { name: 'スライドショーを再生' })).toBeVisible();
    await hero.getByRole('button', { name: '3枚目の写真を表示' }).click();
    await expect(hero.getByRole('img', { name: heroSlides[2]?.alt.ja })).toBeVisible();
    await expect(hero.getByRole('img', { name: heroSlides[0]?.alt.ja })).toHaveCount(0);
    await expect(page.getByText('仮画像')).toHaveCount(0);
    for (const img of await page.locator('main img').all()) {
      await img.scrollIntoViewIfNeeded();
      await expect.poll(() => img.evaluate(loaded), (await img.getAttribute('src')) ?? 'img').toBe(true);
    }

    await page.goto('/ja/facilities');
    await expect(page.getByText('仮画像')).toHaveCount(0);
    // Desktop mosaic: the room photo and three supporting photos.
    for (const p of gallery.slice(0, 4)) await expect(page.getByRole('img', { name: p.alt.ja }).first()).toBeVisible();

    await page.setViewportSize({ width: 390, height: 844 });
    await expect(page.getByText(`1 / ${gallery.length}`)).toBeVisible();
    await page.getByRole('button', { name: '次の写真' }).click();
    await expect(page.getByText(`2 / ${gallery.length}`)).toBeVisible();
  });

  test('terms show the appended tables and the operator block; rates link to them', async ({ page }) => {
    await page.goto('/ja/terms');
    const penalties = page.getByRole('table').filter({ hasText: '契約解除の通知を受けた日' });
    await expect(penalties.getByRole('cell', { name: 'なし（全額返金）' })).toBeVisible();
    await expect(page.getByText('長野県佐久保健所指令07佐保第11-68号')).toBeVisible();

    await page.goto('/ja/rates');
    await expect(page.getByRole('cell', { name: '15,000円 〜 28,000円' })).toBeVisible();
    await page.getByRole('main').getByRole('link', { name: '宿泊約款' }).click();
    await expect(page).toHaveURL(/\/ja\/terms$/);
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
