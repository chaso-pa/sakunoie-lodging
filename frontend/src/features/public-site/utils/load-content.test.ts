import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { defaultFacility } from '@/config/site';
import { featureIconNames } from '@/features/public-site/types/icons';
import { pageSlugs, parsePage } from '@/features/public-site/utils/load-content';
import { routing } from '@/i18n/routing';

const root = path.resolve(import.meta.dirname, '../../../..');
const backendPolicy = JSON.parse(
  readFileSync(path.join(root, '../backend/internal/services/policy_versions.json'), 'utf8')
) as { terms_version: string; privacy_version: string };

describe('public page content', () => {
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

  it('facilities: photo ids exist and the booking price matches the basic rate in rates.md', () => {
    const photoIds = new Set(defaultFacility.photos.map((p) => p.id));
    const digits = (s: string) => s.match(/\d[\d,]*/g)?.map((n) => n.replaceAll(',', '')) ?? [];
    for (const locale of routing.locales) {
      const read = (slug: 'facilities' | 'rates') =>
        readFileSync(path.join(root, 'content', locale, `${slug}.md`), 'utf8');
      const { meta } = parsePage('facilities', read('facilities'));
      for (const id of [meta.spaces.private.photo, ...meta.spaces.shared.photos]) {
        expect(photoIds.has(id), `${locale} photo ${id}`).toBe(true);
      }
      const rates = parsePage('rates', read('rates')).meta.rates;
      expect(digits(meta.booking.price), locale).toEqual(digits(rates[0].price));
    }
  });

  it('facility amenities use known icons', () => {
    for (const a of defaultFacility.amenities) expect(featureIconNames, a.code).toContain(a.icon);
  });

  it('facilities: each fact appears once (amenities repeat nothing from key facts, spaces or rules)', () => {
    const amenityIcons = new Set(defaultFacility.amenities.map((a) => a.icon));
    for (const locale of routing.locales) {
      const { meta } = parsePage(
        'facilities',
        readFileSync(path.join(root, 'content', locale, 'facilities.md'), 'utf8')
      );
      const shownElsewhere = [
        ...meta.keyFacts.map((f) => f.icon),
        ...meta.spaces.private.items.map((i) => i.icon),
        ...meta.spaces.shared.items.map((i) => i.icon),
        ...meta.rules.items.map((r) => r.icon)
      ];
      for (const icon of shownElsewhere) expect(amenityIcons.has(icon), `${locale}: ${icon}`).toBe(false);
      // Towels are an amenity only; air conditioning and heating belong to the guest room only.
      expect(amenityIcons.has('towel')).toBe(true);
      expect(shownElsewhere, locale).not.toContain('towel');
      expect(amenityIcons.has('airConditioning')).toBe(false);
      // Room size, bed and air conditioning/heating are listed once: in the private card, not in the key facts.
      for (const icon of ['ruler', 'bed', 'airConditioning']) {
        expect(
          meta.spaces.private.items.map((i) => i.icon),
          `${locale}: ${icon}`
        ).toContain(icon);
        expect(
          meta.keyFacts.map((f) => f.icon),
          `${locale}: ${icon}`
        ).not.toContain(icon);
      }
      // Check-in/out lives in the key facts only.
      expect(
        meta.rules.items.map((r) => r.icon),
        locale
      ).not.toContain('clock');
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
