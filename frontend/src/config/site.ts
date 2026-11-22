import sakuNoIe from '@content/facilities/saku-no-ie.json';
import site from '@content/site.json';
import type { Locale } from '@/i18n/routing';

/** Content in Git (`frontend/content`), typed. `src/config/content.test.ts` checks the files. */
export type LocalizedText = Record<Locale, string>;

export type FacilityPhoto = {
  id: string;
  src: string;
  /** Intrinsic ratio of the asset, `w:h` (DESIGN §3: store aspect ratio and focal point). */
  aspectRatio: string;
  /** CSS `object-position` of the subject, kept when a crop changes. */
  focalPoint: string;
  placeholder: boolean;
  alt: LocalizedText;
};

export type FacilityAmenity = { code: string; icon: string; label: LocalizedText };

export type FacilityContent = {
  code: string;
  name: LocalizedText;
  area: LocalizedText;
  /** External booking link (Airbnb decides prices and availability). */
  airbnbUrl: string;
  airbnbUrlIsPlaceholder: boolean;
  amenities: FacilityAmenity[];
  photos: FacilityPhoto[];
};

/** Register each `content/facilities/<code>.json` here (static imports keep it bundled). */
const facilityContents: Record<string, FacilityContent> = {
  'saku-no-ie': sakuNoIe
};

/** Facility codes the public site offers (survey choices, links). */
export const contentFacilityCodes: readonly string[] = site.facilities;

/** Facility used when a page has no facility in its URL (home CTA, `/survey` without `?facility=`). */
export const defaultFacilityCode: string = site.defaultFacilityCode;

export const getFacilityContent = (code: string): FacilityContent | undefined => facilityContents[code];

export const defaultFacility: FacilityContent = facilityContents[defaultFacilityCode];

/** `"16:9"` → 16/9. */
export const parseAspectRatio = (ratio: string) => {
  const [w, h] = ratio.split(':').map(Number);
  return w > 0 && h > 0 ? w / h : 4 / 3;
};
