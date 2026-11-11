import { describe, expect, it } from 'vitest';
import { routing } from '@/i18n/routing';
import en from '../../messages/en.json';
import ja from '../../messages/ja.json';
import ko from '../../messages/ko.json';
import zhCN from '../../messages/zh-CN.json';

const flatten = (value: unknown, prefix = ''): Record<string, string> => {
  if (typeof value === 'string') return { [prefix]: value };
  return Object.entries(value as Record<string, unknown>).reduce<Record<string, string>>(
    (acc, [key, child]) => Object.assign(acc, flatten(child, prefix ? `${prefix}.${key}` : key)),
    {}
  );
};

const catalogs = { ja, en, 'zh-CN': zhCN, ko } as const;
const base = flatten(ja);

describe('message catalogs', () => {
  it('has one catalog per routing locale', () => {
    expect(Object.keys(catalogs).sort()).toEqual([...routing.locales].sort());
  });

  for (const [locale, catalog] of Object.entries(catalogs)) {
    it(`${locale} has exactly the keys of ja, all non-empty`, () => {
      const flat = flatten(catalog);
      expect(Object.keys(flat).sort()).toEqual(Object.keys(base).sort());
      for (const [key, value] of Object.entries(flat)) expect(value.trim(), key).not.toBe('');
    });

    it(`${locale} uses the same ICU arguments as ja`, () => {
      const flat = flatten(catalog);
      const args = (s: string) => [...s.matchAll(/\{(\w+)/g)].map((m) => m[1]);
      for (const key of Object.keys(base)) {
        expect(new Set(args(flat[key])), key).toEqual(new Set(args(base[key])));
      }
    });
  }

  it('has a message for every API error code the UI localizes', () => {
    for (const code of [
      'validation_error',
      'facility_unavailable',
      'capacity_exceeded',
      'idempotency_conflict',
      'rate_limited',
      'token_invalid',
      'token_expired',
      'field_not_editable',
      'precondition_failed',
      'not_found',
      'network_error',
      'internal_error',
      'unknown'
    ]) {
      expect(base[`errors.${code}`], code).toBeTruthy();
    }
  });
});
