import type { BrowserContext, Page } from '@playwright/test';

const CSP_RE = /Content Security Policy|Refused to (execute|load|apply|connect|frame)|CSP violation/;

/**
 * Collects console errors, page errors, CSP violations (console text plus `securitypolicyviolation`
 * events reported from an init script) and every request URL of a page.
 */
export const watchPage = async (context: BrowserContext, page: Page) => {
  const consoleErrors: string[] = [];
  const cspViolations: string[] = [];
  const requestUrls: string[] = [];
  await context.addInitScript(() => {
    document.addEventListener('securitypolicyviolation', (e) => {
      console.error(`CSP violation ${e.violatedDirective} ${e.blockedURI}`);
    });
  });
  page.on('console', (m) => {
    const text = m.text();
    if (CSP_RE.test(text)) cspViolations.push(text);
    else if (m.type() === 'error') consoleErrors.push(text);
  });
  page.on('pageerror', (e) => consoleErrors.push(`pageerror: ${e.message}`));
  page.on('request', (r) => requestUrls.push(r.url()));
  return { consoleErrors, cspViolations, requestUrls };
};

/** Console errors other than the browser's own log line for an expected 4xx response. */
export const unexpectedErrors = (errors: string[]) =>
  errors.filter((e) => !/Failed to load resource: the server responded with a status of 4\d\d/.test(e));

/** Waits until React has hydrated the element matching `selector` (its DOM node carries React props). */
export const waitForHydrated = (page: Page, selector: string) =>
  page.waitForFunction((sel) => {
    const el = document.querySelector(sel);
    return !!el && Object.keys(el).some((k) => k.startsWith('__reactProps'));
  }, selector);
