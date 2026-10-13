import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { type Spot, spotSchema } from '@/types/community';

// The spot page answers with reviews and posts too; the composer only needs the spot.
export const linkedSpotResponseSchema = z.object({ spot: spotSchema });

const getLinkedSpot = async (id: string): Promise<Spot> => {
  const response = await api.get(`/spots/${encodeURIComponent(id)}`);
  return linkedSpotResponseSchema.parse(response.data).spot;
};

/**
 * The spot a `?spot=` link names. It is fetched by id because the public spot list holds only the
 * curated spots, while a link can point at one a member registered.
 */
export const useLinkedSpot = (id: string | undefined) =>
  useQuery({
    queryKey: queryKeys.spots.linked(id ?? ''),
    queryFn: () => getLinkedSpot(id as string),
    enabled: Boolean(id)
  });
