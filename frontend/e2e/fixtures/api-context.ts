import type { APIRequestContext, PlaywrightWorkerArgs } from '@playwright/test';

/** Web origin (Playwright `baseURL`). */
export const WEB_URL = process.env.E2E_BASE_URL ?? 'http://127.0.0.1:3100';
/** API origin the web build calls cross-origin (`NEXT_PUBLIC_API_BASE_URL` of the E2E build). */
export const API_URL = process.env.E2E_API_URL ?? 'http://127.0.0.1:8180';

/** An API request context on the API origin for `beforeAll` (the `request` fixture is test-scoped). */
export const newApiContext = (playwright: PlaywrightWorkerArgs['playwright']): Promise<APIRequestContext> =>
  playwright.request.newContext({ baseURL: API_URL });
