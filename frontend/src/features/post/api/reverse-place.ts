import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { type LatLng, placeAttributionSchema, placeSchema } from '@/features/post/types/place';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';

export const reversePlaceSchema = z.object({
  /** null when the provider knows nothing at that position. */
  result: placeSchema.nullable(),
  provider: z.string(),
  attribution: placeAttributionSchema
});
export type ReversePlace = z.infer<typeof reversePlaceSchema>;

const reversePlace = async (position: LatLng, language: string, signal: AbortSignal): Promise<ReversePlace> => {
  const response = await api.post(
    '/places/reverse',
    { latitude: position.latitude, longitude: position.longitude, language },
    { signal }
  );
  return reversePlaceSchema.parse(response.data);
};

/** Name and address of a position tapped on the map. Not kept after the map closes. */
export const useReversePlace = (position: LatLng | null, language: string) =>
  useQuery({
    queryKey: queryKeys.places.reverse(position?.latitude ?? 0, position?.longitude ?? 0, language),
    queryFn: ({ signal }) => reversePlace(position as LatLng, language, signal),
    enabled: position !== null,
    retry: false,
    staleTime: 0,
    gcTime: 0
  });
