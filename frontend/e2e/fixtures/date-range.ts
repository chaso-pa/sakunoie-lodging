import { expect, type Locator, type Page } from '@playwright/test';

/** A 2-night stay 2 weeks to ~8 months ahead, so the calendar needs only a few "next month" clicks. */
export const nearStay = () => {
  const offset = 14 + Math.floor(Math.random() * 230);
  const day = (n: number) => new Date(Date.now() + n * 86_400_000).toISOString().slice(0, 10);
  return { checkIn: day(offset), checkOut: day(offset + 2) };
};

const dayLabel = (iso: string) => {
  const [y, m, d] = iso.split('-').map(Number);
  return `${y}年${m}月${d}日`;
};

/** Picks check-in and check-out in the admin range calendar (DatePickerInput type="range"). */
export const pickStayRange = async (page: Page, field: Locator, checkIn: string, checkOut: string) => {
  await field.click();
  for (const iso of [checkIn, checkOut]) {
    const day = page.getByRole('button', { name: dayLabel(iso), exact: true });
    for (let i = 0; i < 24 && !(await day.isVisible()); i++) {
      await page.getByRole('button', { name: '次の月' }).first().click();
    }
    await day.click();
  }
  await expect(field).toContainText('〜');
};
