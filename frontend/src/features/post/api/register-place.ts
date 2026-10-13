import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { locales } from '@/i18n/config';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { spotSchema } from '@/types/community';

export const placeNameMaxLength = 100;
export const placeAddressMaxLength = 200;

/**
 * A place to keep as a community spot. A search result is named by `source` and `external_id` alone: the
 * server looks the rest up itself, so nothing a provider returned is sent back. A hand-entered place
 * carries its name, and `latitude` and `longitude` only when the member put a pin on the map; they are
 * left out, never zeroed, otherwise.
 */
export const registerPlaceInputSchema = z.discriminatedUnion('source', [
  z.object({
    source: z.enum(['osm', 'google']),
    external_id: z.string().min(1).max(512),
    language: z.enum(locales)
  }),
  z
    .object({
      source: z.literal('manual'),
      language: z.enum(locales),
      name: z.string().trim().min(1).max(placeNameMaxLength),
      address: z.string().trim().max(placeAddressMaxLength).optional(),
      latitude: z.number().min(-90).max(90).optional(),
      longitude: z.number().min(-180).max(180).optional()
    })
    .refine((value) => (value.latitude === undefined) === (value.longitude === undefined))
]);
export type RegisterPlaceInput = z.infer<typeof registerPlaceInputSchema>;

export const registerPlaceResponseSchema = z.object({
  spot: spotSchema,
  /** false when the same place was a spot already. */
  created: z.boolean()
});

const registerPlace = async (data: RegisterPlaceInput) => {
  const response = await api.post('/spots', registerPlaceInputSchema.parse(data));
  return registerPlaceResponseSchema.parse(response.data).spot;
};

export const useRegisterPlace = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: registerPlace,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.spots.all })
  });
};
