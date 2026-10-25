import { useMutation } from '@tanstack/react-query';
import { type PlaceList, placeListSchema } from '@/features/post/api/search-places';
import type { LatLng } from '@/features/post/types/place';
import { api } from '@/lib/api-client';

const getNearbyPlaces = async ({ position, language }: { position: LatLng; language: string }): Promise<PlaceList> => {
  const response = await api.post('/places/nearby', {
    latitude: position.latitude,
    longitude: position.longitude,
    language
  });
  return placeListSchema.parse(response.data);
};

/**
 * Places around a position. A mutation rather than a query on purpose: the member's position would
 * otherwise sit in a query key in the cache. It lives only in the variables of this one call, and
 * travels in the request body, not in a URL. The answer is never reused, so it is not kept once the
 * picker unmounts (`gcTime: 0`): neither the position nor, with Google, its places.
 */
export const useNearbyPlaces = () => useMutation({ mutationFn: getNearbyPlaces, gcTime: 0 });
