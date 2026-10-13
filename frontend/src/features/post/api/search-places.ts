import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { placeAttributionSchema, placeSchema } from '@/features/post/types/place';
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

/**
 * Keyword search, for a query the member explicitly submitted. It must not be called per keystroke: the
 * default provider (Nominatim) forbids search-as-you-type. TanStack Query aborts the request of a query
 * nobody is looking at any more, so a slow answer to an old keyword never replaces a newer one, and
 * repeating a query within five minutes is answered from memory. A failure is not retried: the provider
 * is rate limited, and the member can ask again.
 */
export const usePlaceSearch = (query: string, language: string) =>
  useQuery({
    queryKey: queryKeys.places.search(query, language),
    queryFn: ({ signal }) => searchPlaces(query, language, signal),
    enabled: query !== '',
    retry: false,
    staleTime: 5 * 60 * 1000,
    gcTime: 5 * 60 * 1000
  });
