import { describe, expect, it } from 'vitest';
import { formatCivilDate, formatInstant } from '@/features/registration/utils/format';

describe('formatInstant', () => {
  it('shows the edit expiry in the facility time zone for every locale', () => {
    // End of 2026-12-03 in Asia/Tokyo.
    const iso = '2026-12-03T14:59:59.999999Z';
    for (const locale of ['ja', 'en', 'zh-CN', 'ko']) {
      expect(formatInstant(iso, locale, 'Asia/Tokyo'), locale).toMatch(/3/);
    }
    expect(formatInstant(iso, 'en', 'Asia/Tokyo')).toContain('Dec 3, 2026');
    expect(formatInstant(iso, 'en', 'Asia/Tokyo')).toMatch(/11:59\s?PM/);
  });
});

describe('formatCivilDate', () => {
  it('never shifts a civil date through a time zone', () => {
    expect(formatCivilDate('2026-12-01', 'en')).toBe('Dec 1, 2026');
  });

  it('returns invalid input unchanged instead of throwing', () => {
    expect(formatCivilDate('', 'en')).toBe('');
  });
});
