import { z } from 'zod';
import { httpUrlOrNull } from '@/utils/http-url';

/** Go encodes an empty slice as `null`; treat that as an empty list. */
const listOf = <T extends z.ZodType>(item: T) =>
  z
    .array(item)
    .nullable()
    .transform((value) => value ?? []);

/** A member as shown next to their content. `avatar_url` is an empty string when unset. */
export const memberSummarySchema = z.object({
  id: z.string(),
  nickname: z.string(),
  avatar_url: z.string()
});
export type MemberSummary = z.infer<typeof memberSummarySchema>;

export const mediaSchema = z.object({
  url: z.string(),
  type: z.string()
});
export type Media = z.infer<typeof mediaSchema>;

/** seed, osm, google or manual today. Left open so a source added later does not fail every list. */
export const spotSourceSchema = z.string();

/**
 * Where a spot is, when known. A `google` spot keeps only its place id on the server: the rest is fetched
 * when it is read, and `details_available` is false (with an empty `name`) when that failed.
 */
const spotPlaceFields = {
  source: spotSourceSchema,
  address: z.string().nullable(),
  latitude: z.number().nullable(),
  longitude: z.number().nullable(),
  /** Rendered as a link, so anything but an http(s) URL is dropped. */
  map_url: httpUrlOrNull,
  details_available: z.boolean()
};

export const postSpotSchema = z.object({
  id: z.string(),
  name: z.string(),
  emoji: z.string(),
  /** Empty for a member-registered spot outside the four regions. */
  area: z.string(),
  ...spotPlaceFields
});
export type PostSpot = z.infer<typeof postSpotSchema>;

/** A post as the signed-in member sees it (`is_liked`, `is_bookmarked` and `is_mine` are per viewer). */
export const postSchema = z.object({
  id: z.string(),
  member: memberSummarySchema,
  spot: postSpotSchema.nullable(),
  caption: z.string(),
  media: listOf(mediaSchema),
  language: z.string(),
  area: z.string().nullable(),
  weather: z.string().nullable(),
  temperature: z.number().nullable(),
  visited_at: z.string().nullable(),
  tags: listOf(z.string()),
  like_count: z.number(),
  is_liked: z.boolean(),
  is_bookmarked: z.boolean(),
  is_mine: z.boolean(),
  is_sample: z.boolean(),
  created_at: z.string()
});
export type Post = z.infer<typeof postSchema>;

export const postListSchema = z.object({ posts: listOf(postSchema) });

export const spotSchema = z.object({
  id: z.string(),
  name: z.string(),
  emoji: z.string(),
  area: z.string(),
  category: z.string(),
  description: z.string(),
  is_sample: z.boolean(),
  ...spotPlaceFields,
  external_id: z.string().nullable(),
  /** The local-language name, when it differs from `name`. */
  original_name: z.string().nullable(),
  is_mine: z.boolean(),
  created_at: z.string()
});
export type Spot = z.infer<typeof spotSchema>;

export const spotListItemSchema = spotSchema.extend({
  review_count: z.number(),
  avg_stars: z.number(),
  post_count: z.number()
});
export type SpotListItem = z.infer<typeof spotListItemSchema>;

export const reviewSchema = z.object({
  id: z.string(),
  member: memberSummarySchema,
  stars: z.number(),
  text: z.string(),
  is_sample: z.boolean(),
  created_at: z.string()
});
export type Review = z.infer<typeof reviewSchema>;

/**
 * A footprint. `latitude` and `longitude` are null unless the viewer owns it or its owner chose to publish
 * the exact position; the API decides that, and the UI only ever shows what it was given.
 */
export const gpsVisitSchema = z.object({
  id: z.string(),
  member: memberSummarySchema,
  is_mine: z.boolean(),
  spot_id: z.string().nullable(),
  place_name: z.string(),
  emoji: z.string(),
  area: z.string(),
  visited_date: z.string(),
  weather: z.string().nullable(),
  temperature: z.number().nullable(),
  note: z.string().nullable(),
  is_public: z.boolean(),
  exact_public: z.boolean(),
  latitude: z.number().nullable(),
  longitude: z.number().nullable(),
  created_at: z.string()
});
export type GpsVisit = z.infer<typeof gpsVisitSchema>;

export const gpsVisitListSchema = z.object({ visits: listOf(gpsVisitSchema) });

export const okResponseSchema = z.object({ ok: z.boolean() });

export { listOf };
