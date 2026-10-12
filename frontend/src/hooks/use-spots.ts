import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { listOf, type SpotListItem, spotListItemSchema } from '@/types/community';

export const spotsResponseSchema = z.object({ spots: listOf(spotListItemSchema) });

const getSpots = async (): Promise<SpotListItem[]> => {
  const response = await api.get('/spots');
  return spotsResponseSchema.parse(response.data).spots;
};

/** Every spot. The list is short, so area and keyword filtering happen in the browser. */
export const useSpots = () => useQuery({ queryKey: queryKeys.spots.list, queryFn: getSpots });
