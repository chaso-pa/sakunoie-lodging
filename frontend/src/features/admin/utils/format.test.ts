import { describe, expect, it } from 'vitest';
import { adminErrorMessage, signInErrorMessage } from '@/features/admin/utils/error-message';
import { filenameFromDisposition, formatDate, formatInstant } from '@/features/admin/utils/format';
import { ApiError } from '@/lib/api-client';

describe('admin format utils', () => {
  it('formats civil dates without time-zone shifts', () => {
    expect(formatDate('2026-12-01')).toBe('2026/12/01');
  });

  it('formats instants in the facility time zone', () => {
    expect(formatInstant('2026-12-03T14:59:59Z', 'Asia/Tokyo')).toBe('2026/12/03 23:59');
    expect(formatInstant('2026-12-04T09:59:59Z', 'Pacific/Honolulu')).toBe('2026/12/03 23:59');
  });

  it('takes only safe filenames from Content-Disposition', () => {
    const h = 'attachment; filename="registrations-saku-no-ie-2026-12-01-2026-12-31.csv"';
    expect(filenameFromDisposition(h, 'x.csv')).toBe('registrations-saku-no-ie-2026-12-01-2026-12-31.csv');
    expect(filenameFromDisposition('attachment; filename="../evil.csv"', 'x.csv')).toBe('x.csv');
    expect(filenameFromDisposition(null, 'x.csv')).toBe('x.csv');
  });
});

describe('admin error messages', () => {
  it('maps envelope codes to Japanese copy', () => {
    expect(adminErrorMessage(new ApiError(403, 'not_authorized', 'x', [], null))).toMatch(/権限がありません/);
    expect(adminErrorMessage(new ApiError(500, 'internal_error', 'x', [], null))).toMatch(/サーバー/);
    expect(adminErrorMessage(new ApiError(418, 'teapot', 'x', [], null))).toMatch(/処理できませんでした/);
  });

  it('never distinguishes unknown accounts from wrong passwords', () => {
    expect(signInErrorMessage({ code: 'auth/user-not-found' })).toBe(
      signInErrorMessage({ code: 'auth/wrong-password' })
    );
  });
});
