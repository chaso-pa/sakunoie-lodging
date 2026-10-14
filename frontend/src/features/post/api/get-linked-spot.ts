import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { useDropUnusedQuery } from '@/hooks/use-drop-unused-query';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { type Spot, spotSchema } from '@/types/community';

// The spot page answers with reviews and posts too; the composer only needs the spot.
export const linkedSpotResponseSchema = z.object({ spot: spotSchema });

const getLinkedSpot = async (id: string): Promise<Spot> => {
  const response = await api.get(`/spots/${encodeURIComponent(id)}`);
  return linkedSpotResponseSchema.parse(response.data).spot;
};

// A Google spot's name and address are read from Google on every request; they may not be kept.
const isGoogleSpot = (spot: Spot | undefined) => spot?.source === 'google';

/**
 * The spot a `?spot=` link names. It is fetched by id because the public spot list holds only the
 * curated spots, while a link can point at one a member registered. A Google spot is dropped from the
 * cache when the composer closes.
 */
export const useLinkedSpot = (id: string | undefined) => {
  const queryKey = queryKeys.spots.linked(id ?? '');
  useDropUnusedQuery(queryKey, isGoogleSpot);
  return useQuery({ queryKey, queryFn: () => getLinkedSpot(id as string), enabled: Boolean(id) });
};
