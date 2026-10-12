import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { reviewSchema } from '@/types/community';

export const reviewTextMaxLength = 1000;

export const createReviewInputSchema = z.object({
  stars: z.number().int().min(1).max(5),
  text: z.string().trim().min(1).max(reviewTextMaxLength)
});
export type CreateReviewInput = z.infer<typeof createReviewInputSchema>;

export const createReviewResponseSchema = z.object({ review: reviewSchema });

const createReview = async ({ spotId, ...data }: CreateReviewInput & { spotId: string }) => {
  const response = await api.post(`/spots/${encodeURIComponent(spotId)}/reviews`, createReviewInputSchema.parse(data));
  return createReviewResponseSchema.parse(response.data).review;
};

export const useCreateReview = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createReview,
    // The spot page lists the review; the spot list carries its count and average.
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.spots.all })
  });
};
