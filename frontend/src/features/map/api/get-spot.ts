import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { useDropUnusedQuery } from '@/hooks/use-drop-unused-query';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { listOf, mediaSchema, memberSummarySchema, reviewSchema, spotSchema } from '@/types/community';

/** A post as listed on a spot's page: shorter than a feed post, with no per-viewer like or save state. */
export const spotPostSchema = z.object({
  id: z.string(),
  caption: z.string(),
  media: listOf(mediaSchema),
  member: memberSummarySchema,
  like_count: z.number(),
  is_sample: z.boolean(),
  created_at: z.string()
});
export type SpotPost = z.infer<typeof spotPostSchema>;

export const spotDetailSchema = z.object({
  spot: spotSchema,
  reviews: listOf(reviewSchema),
  posts: listOf(spotPostSchema)
});
export type SpotDetail = z.infer<typeof spotDetailSchema>;

const getSpot = async (id: string): Promise<SpotDetail> => {
  const response = await api.get(`/spots/${encodeURIComponent(id)}`);
  return spotDetailSchema.parse(response.data);
};

// A Google spot's name and address are read from Google on every request; they may not be kept.
const isGoogleSpot = (detail: SpotDetail | undefined) => detail?.spot.source === 'google';

/** A spot's page. A Google spot is dropped from the cache when the page is left. */
export const useSpot = (id: string) => {
  const queryKey = queryKeys.spots.detail(id);
  useDropUnusedQuery(queryKey, isGoogleSpot);
  return useQuery({ queryKey, queryFn: () => getSpot(id) });
};
