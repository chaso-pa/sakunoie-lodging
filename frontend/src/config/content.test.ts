import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { contentFacilityCodes, defaultFacilityCode, getFacilityContent, parseAspectRatio } from '@/config/site';
import { pageSlugs, parsePage } from '@/features/public-site/utils/load-content';
import { routing } from '@/i18n/routing';

const root = path.resolve(import.meta.dirname, '../..');
const backendPolicy = JSON.parse(
  readFileSync(path.join(root, '../backend/internal/services/policy_versions.json'), 'utf8')
) as { terms_version: string; privacy_version: string };

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

  for (const locale of routing.locales) {
    for (const slug of pageSlugs) {
      it(`${locale}/${slug}.md exists and has valid front matter`, () => {
        const raw = readFileSync(path.join(root, 'content', locale, `${slug}.md`), 'utf8');
        const { meta } = parsePage(slug, raw);
        // Placeholder content is marked (要差し替え) until the operator replaces it.
        if (meta.placeholder) expect(meta.todo).toContain('要差し替え');
        if (slug === 'terms') expect(meta.version).toBe(backendPolicy.terms_version);
        if (slug === 'privacy') expect(meta.version).toBe(backendPolicy.privacy_version);
      });
    }
  }

  it('the same page has the same version in every locale', () => {
    for (const slug of pageSlugs) {
      const versions = routing.locales.map(
        (l) => parsePage(slug, readFileSync(path.join(root, 'content', l, `${slug}.md`), 'utf8')).meta.version
      );
      expect(new Set(versions).size, slug).toBe(1);
    }
  });

  it('rejects front matter with an unquoted (Date) version', () => {
    expect(() =>
      parsePage(
        'guide',
        '---\ntitle: x\ndescription: x\nversion: 2026-11-01\neffectiveDate: "2026-11-01"\nplaceholder: true\n---\n'
      )
    ).toThrow(/front matter/);
  });
});
