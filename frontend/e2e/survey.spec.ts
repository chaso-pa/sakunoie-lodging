import { expect, test } from '@playwright/test';
import { adminLogin } from './fixtures/admin-login';
import { newApiContext } from './fixtures/api-context';
import { unexpectedErrors, watchPage } from './fixtures/page-watch';
import { seedSurvey } from './fixtures/seed';

const FREE_TEXT = 'Synthetic E2E survey comment';

test.beforeAll(async ({ playwright }) => {
  const api = await newApiContext(playwright);
  await seedSurvey(api, 4);
  await api.dispose();
});

test.describe('survey', () => {
  test('anonymous submit (en), then the admin summary shows aggregates only', async ({ page, context }) => {
    const watch = await watchPage(context, page);
    await page.goto('/en/survey');

    // q01 is required: nothing is sent while it is empty.
    const q01 = page.getByRole('radiogroup', { name: /^1\. / });
    await page.getByRole('button', { name: 'Submit' }).click();
    await expect(q01).toHaveAccessibleDescription(/This field is required/);
    expect(watch.requestUrls.some((u) => u.includes('/api/v1/surveys'))).toBe(false);

    await q01.getByRole('radio', { name: '5 out of 5' }).check();
    await page.getByRole('radio', { name: 'Sightseeing' }).check();
    await page.getByRole('checkbox', { name: 'Quietness' }).check();
    await page.getByLabel(/Any comments or suggestions/).fill(FREE_TEXT);
    const [response] = await Promise.all([
      page.waitForResponse((r) => r.url().endsWith('/api/v1/surveys')),
      page.getByRole('button', { name: 'Submit' }).click()
    ]);
    expect(response.status()).toBe(202);
    const body = response.request().postDataJSON();
    expect(Object.keys(body).sort()).toEqual(['answers', 'facility_code', 'survey_version', 'website']);
    expect(body).toMatchObject({ facility_code: 'saku-no-ie', survey_version: 'v1', website: '' });
    expect(body.answers.q01).toBe(5);
    await expect(page.getByText('Thank you for your feedback')).toBeVisible();
    expect(watch.cspViolations).toEqual([]);
    expect(unexpectedErrors(watch.consoleErrors)).toEqual([]);

    // Admin summary (SURVEY_MIN_RESPONSES=1 in the E2E stack).
    await adminLogin(page);
    await page.goto('/admin/surveys');
    await expect(page.getByRole('heading', { name: 'アンケート集計', level: 1 })).toBeVisible();
    const main = page.getByRole('main');
    const q01Summary = page.getByRole('region', { name: /^1\. / });
    await expect(q01Summary).toContainText(/平均 \d\.\d\d \/ 5/);
    const total = Number(/回答数\s*(\d+)/.exec(await main.innerText())?.[1] ?? '0');
    expect(total).toBeGreaterThanOrEqual(2);
    const q10 = page.getByRole('region', { name: /^10\. / });
    await expect(q10).toContainText(/記入あり \d+件/);
    await expect(main).not.toContainText(FREE_TEXT);
  });
});
