import type { APIRequestContext, PlaywrightWorkerArgs, TestInfo } from '@playwright/test';

/** An API request context for `beforeAll` (the `request` fixture is test-scoped). */
export const newApiContext = (
  playwright: PlaywrightWorkerArgs['playwright'],
  testInfo: TestInfo
): Promise<APIRequestContext> => playwright.request.newContext({ baseURL: testInfo.project.use.baseURL });
