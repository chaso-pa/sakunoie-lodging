import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { gpsVisitSchema, okResponseSchema } from '@/types/community';

export const updateVisibilityInputSchema = z.object({
  is_public: z.boolean(),
  /** Omitted to leave the exact-position setting as it is. */
  exact_public: z.boolean().optional()
});
export type UpdateVisibilityInput = z.infer<typeof updateVisibilityInputSchema>;

export const visitResponseSchema = z.object({ visit: gpsVisitSchema });

const updateVisibility = async ({ id, ...data }: UpdateVisibilityInput & { id: string }) => {
  const response = await api.patch(
    `/gps-visits/${encodeURIComponent(id)}/visibility`,
    updateVisibilityInputSchema.parse(data)
  );
  return visitResponseSchema.parse(response.data).visit;
};

const invalidateVisits = (queryClient: ReturnType<typeof useQueryClient>) => {
  queryClient.invalidateQueries({ queryKey: queryKeys.gpsVisits.all });
  queryClient.invalidateQueries({ queryKey: queryKeys.rankings.all });
};

export const useUpdateGpsVisitVisibility = () => {
  const queryClient = useQueryClient();
  return useMutation({ mutationFn: updateVisibility, onSuccess: () => invalidateVisits(queryClient) });
};

const deleteGpsVisit = async (id: string) => {
  const response = await api.delete(`/gps-visits/${encodeURIComponent(id)}`);
  return okResponseSchema.parse(response.data);
};

export const useDeleteGpsVisit = () => {
  const queryClient = useQueryClient();
  return useMutation({ mutationFn: deleteGpsVisit, onSuccess: () => invalidateVisits(queryClient) });
};
