import { existsSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { contentFacilityCodes, defaultFacilityCode, getFacilityContent, parseAspectRatio } from '@/config/site';
import { routing } from '@/i18n/routing';

const root = path.resolve(import.meta.dirname, '../..');

describe('site content', () => {
  it('lists the default facility and has content for every listed facility', () => {
    expect(contentFacilityCodes).toContain(defaultFacilityCode);
    for (const code of contentFacilityCodes) {
      const f = getFacilityContent(code);
      expect(f, code).toBeDefined();
      expect(f?.code).toBe(code);
      expect(f?.airbnbUrl).toMatch(/^https:\/\//);
    }
  });

  it('has photos in public/ with alt text in every locale, ratio and focal point', () => {
    for (const code of contentFacilityCodes) {
      const f = getFacilityContent(code);
      for (const photo of f?.photos ?? []) {
        expect(existsSync(path.join(root, 'public', photo.src)), photo.src).toBe(true);
        expect(photo.src.startsWith('/images/')).toBe(true);
        expect(parseAspectRatio(photo.aspectRatio)).toBeGreaterThan(0);
        expect(photo.aspectRatio).toMatch(/^\d+:\d+$/);
        expect(photo.focalPoint).toMatch(/^\d+% \d+%$/);
        for (const locale of routing.locales) expect(photo.alt[locale]?.trim(), `${photo.id} ${locale}`).toBeTruthy();
      }
      for (const a of f?.amenities ?? []) {
        for (const locale of routing.locales) expect(a.label[locale]?.trim(), `${a.code} ${locale}`).toBeTruthy();
      }
    }
  });
});
