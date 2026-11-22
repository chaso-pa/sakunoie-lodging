import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { contentFacilityCodes, defaultFacilityCode, getFacilityContent, parseAspectRatio } from '@/config/site';
import { routing } from '@/i18n/routing';

const root = path.resolve(import.meta.dirname, '../..');

/** Pixel size of a baseline or progressive JPEG, read from its SOF segment. */
const jpegSize = (file: string) => {
  const b = readFileSync(file);
  expect(b.readUInt16BE(0), `${file} is a JPEG`).toBe(0xffd8);
  for (let i = 2; i + 9 < b.length; i += 2 + b.readUInt16BE(i + 2)) {
    const marker = b[i + 1];
    if (marker >= 0xc0 && marker <= 0xcf && ![0xc4, 0xc8, 0xcc].includes(marker)) {
      return { width: b.readUInt16BE(i + 7), height: b.readUInt16BE(i + 5) };
    }
  }
  throw new Error(`${file}: no SOF segment`);
};

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

  it('real photos are JPEGs outside the placeholder folder, stored with the pixel ratio of the file', () => {
    for (const code of contentFacilityCodes) {
      for (const photo of getFacilityContent(code)?.photos ?? []) {
        if (photo.placeholder) continue;
        expect(photo.src, photo.id).toMatch(/^\/images\/(?!placeholder\/)[\w/-]+\.jpg$/);
        const { width, height } = jpegSize(path.join(root, 'public', photo.src));
        expect(parseAspectRatio(photo.aspectRatio), photo.id).toBeCloseTo(width / height, 3);
      }
    }
  });
});
