import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { gpsVisitSchema } from '@/types/community';

export const visitNoteMaxLength = 2000;
export const visitPlaceMaxLength = 100;

/**
 * A new footprint. Either `spot_id` or `place_name` names the place. `latitude` and `longitude` are sent
 * only when the member tapped "use current location" and the browser granted it; `exact_public` decides
 * whether other members may ever see them, and `is_public` whether they see the footprint at all.
 */
export const createGpsVisitInputSchema = z
  .object({
    spot_id: z.string().optional(),
    place_name: z.string().trim().max(visitPlaceMaxLength).optional(),
    visited_date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
    weather: z.string().max(20).optional(),
    temperature: z.number().min(-60).max(60).optional(),
    note: z.string().max(visitNoteMaxLength).optional(),
    is_public: z.boolean(),
    exact_public: z.boolean(),
    latitude: z.number().min(-90).max(90).optional(),
    longitude: z.number().min(-180).max(180).optional()
  })
  .refine((value) => Boolean(value.spot_id) || Boolean(value.place_name))
  .refine((value) => (value.latitude === undefined) === (value.longitude === undefined));
export type CreateGpsVisitInput = z.infer<typeof createGpsVisitInputSchema>;

export const gpsVisitResponseSchema = z.object({ visit: gpsVisitSchema });

const createGpsVisit = async (data: CreateGpsVisitInput) => {
  const response = await api.post('/gps-visits', createGpsVisitInputSchema.parse(data));
  return gpsVisitResponseSchema.parse(response.data).visit;
};

export const useCreateGpsVisit = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createGpsVisit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.gpsVisits.all });
      // Public footprints count towards the spot ranking.
      queryClient.invalidateQueries({ queryKey: queryKeys.rankings.all });
    }
  });
};
