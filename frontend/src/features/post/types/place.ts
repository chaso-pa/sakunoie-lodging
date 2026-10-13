import { z } from 'zod';
import { httpUrlOrNull } from '@/utils/http-url';

export type LatLng = { latitude: number; longitude: number };

/** Provider whose terms forbid drawing its results on another provider's map, or storing them. */
export const googleSource = 'google';

export const placeCategoryKeys = [
  'food',
  'cafe',
  'shop',
  'sight',
  'history',
  'nature',
  'onsen',
  'lodging',
  'transport',
  'other'
] as const;
export type PlaceCategoryKey = (typeof placeCategoryKeys)[number];

/**
 * A place found by search, nearby search or reverse lookup (backend `models.PlaceResult`). `name`,
 * `original_name` and `address` are shown exactly as returned. Nothing of a result is ever written to
 * browser storage.
 */
export const placeSchema = z.object({
  /** `seed` and `manual` are spots that already exist; `osm` and `google` come from the provider. */
  source: z.string(),
  external_id: z.string().nullable(),
  /** Set when the place is already a spot: it can go straight into a post. */
  spot_id: z.string().nullable(),
  name: z.string(),
  /** The local-language name, only when it differs from `name`. */
  original_name: z.string().nullable(),
  address: z.string(),
  category: z.string(),
  /** A key this build does not know yet is shown as "other" rather than failing the whole list. */
  category_key: z.enum(placeCategoryKeys).catch('other'),
  emoji: z.string(),
  latitude: z.number().nullable(),
  longitude: z.number().nullable(),
  map_url: httpUrlOrNull,
  distance_m: z.number().nullable(),
  photo_url: httpUrlOrNull
});
export type Place = z.infer<typeof placeSchema>;

/** `url` is null when the API sent something that is not an http(s) URL; the text is then shown unlinked. */
export const placeAttributionSchema = z.object({ text: z.string(), url: httpUrlOrNull });
export type PlaceAttribution = z.infer<typeof placeAttributionSchema>;

export const placeKey = (place: Place) => `${place.source}:${place.external_id ?? place.spot_id ?? place.name}`;

/** What the composer holds once a place is chosen. Only `spot` exists on the server already. */
export type PlaceSelection =
  | {
      kind: 'spot';
      id: string;
      name: string;
      originalName?: string | null;
      address: string;
      emoji: string;
      area: string;
      /** Where the spot's name and address came from: decides the credit shown with them. */
      source: string;
    }
  | { kind: 'candidate'; place: Place & { external_id: string; source: 'osm' | 'google' } }
  | { kind: 'manual'; name: string; address: string; position: LatLng | null };

/** A search result as a selection: an existing spot when it is one, otherwise a candidate to register. */
export const toSelection = (place: Place): PlaceSelection | null => {
  if (place.spot_id) {
    return {
      kind: 'spot',
      id: place.spot_id,
      name: place.name,
      originalName: place.original_name,
      address: place.address,
      emoji: place.emoji,
      source: place.source,
      // Not part of a search result; the post then takes the spot's own area on the server.
      area: ''
    };
  }
  return place.external_id && (place.source === 'osm' || place.source === 'google')
    ? { kind: 'candidate', place: { ...place, external_id: place.external_id, source: place.source } }
    : null;
};
