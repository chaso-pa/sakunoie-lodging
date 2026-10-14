import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { type LatLng, placeAttributionSchema, placeSchema } from '@/features/post/types/place';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';

export const reversePlaceSchema = z.object({
  /** null when the provider knows nothing at that position. */
  result: placeSchema.nullable(),
  provider: z.string(),
  attribution: placeAttributionSchema,
  /**
   * The provider is Google and the position was not picked on a Google map, so it was not asked: Google's
   * content may only be shown on its own map. `result` is then null.
   */
  needs_google_map: z.boolean().catch(false)
});
export type ReversePlace = z.infer<typeof reversePlaceSchema>;

const reversePlace = async (
  position: LatLng,
  language: string,
  googleMap: boolean,
  signal: AbortSignal
): Promise<ReversePlace> => {
  const response = await api.post(
    '/places/reverse',
    { latitude: position.latitude, longitude: position.longitude, language, google_map: googleMap },
    { signal }
  );
  return reversePlaceSchema.parse(response.data);
};

/**
 * Name and address of a position tapped on the map. Not kept after the map closes. `googleMap` says whether
 * the position was picked on a Google map.
 */
export const useReversePlace = (position: LatLng | null, language: string, googleMap: boolean) =>
  useQuery({
    queryKey: queryKeys.places.reverse(position?.latitude ?? 0, position?.longitude ?? 0, language, googleMap),
    queryFn: ({ signal }) => reversePlace(position as LatLng, language, googleMap, signal),
    enabled: position !== null,
    retry: false,
    staleTime: 0,
    gcTime: 0
  });
