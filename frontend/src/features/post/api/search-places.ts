import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { googleSource, placeAttributionSchema, placeSchema } from '@/features/post/types/place';
import { useDropUnusedQuery } from '@/hooks/use-drop-unused-query';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { listOf } from '@/types/community';

export const placeQueryMaxLength = 100;

export const placeListSchema = z.object({
  results: listOf(placeSchema),
  /** Service that answered: decides the attribution shown and whether results may go on the map. */
  provider: z.string(),
  attribution: placeAttributionSchema
});
export type PlaceList = z.infer<typeof placeListSchema>;

const searchPlaces = async (query: string, language: string, signal: AbortSignal): Promise<PlaceList> => {
  const response = await api.post('/places/search', { query, language }, { signal });
  return placeListSchema.parse(response.data);
};

/** Google's results may be shown but not kept (docs/spot-search.md, section 7). */
export const isGooglePlaceList = (data: PlaceList | undefined) => data?.provider === googleSource;

const keepMs = 5 * 60 * 1000;

/**
 * Keyword search, for a query the member explicitly submitted. It must not be called per keystroke: the
 * default provider (Nominatim) forbids search-as-you-type. TanStack Query aborts the request of a query
 * nobody is looking at any more, so a slow answer to an old keyword never replaces a newer one. A failure
 * is not retried: the provider is rate limited, and the member can ask again.
 *
 * OpenStreetMap results stay in memory for five minutes, so repeating a keyword is answered from there.
 * Google's are dropped as soon as the picker moves on to another keyword or closes; while they are shown
 * they count as fresh, so nothing (a re-render, focus, reconnecting) asks Google for them again.
 */
export const usePlaceSearch = (query: string, language: string) => {
  const queryKey = queryKeys.places.search(query, language);
  useDropUnusedQuery(queryKey, isGooglePlaceList);
  return useQuery({
    queryKey,
    queryFn: ({ signal }) => searchPlaces(query, language, signal),
    enabled: query !== '',
    retry: false,
    staleTime: keepMs,
    gcTime: keepMs
  });
};
