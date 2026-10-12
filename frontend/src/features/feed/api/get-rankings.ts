import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { listOf } from '@/types/community';

export const rankingTypeSchema = z.enum(['spots', 'posts']);
export type RankingType = z.infer<typeof rankingTypeSchema>;

export const rankingItemSchema = z.object({
  rank: z.number(),
  /** Spot id for the spot ranking, post id for the post ranking. */
  id: z.string(),
  name: z.string(),
  score: z.number(),
  detail: z.string(),
  emoji: z.string(),
  area: z.string(),
  post_count: z.number(),
  like_count: z.number(),
  spot_id: z.string().nullable(),
  is_sample: z.boolean()
});
export type RankingItem = z.infer<typeof rankingItemSchema>;

export const rankingsResponseSchema = z.object({ items: listOf(rankingItemSchema) });

const getRankings = async (type: RankingType): Promise<RankingItem[]> => {
  const response = await api.get('/rankings', { params: { type: rankingTypeSchema.parse(type), limit: 3 } });
  return rankingsResponseSchema.parse(response.data).items;
};

export const useRankings = (type: RankingType) =>
  useQuery({ queryKey: queryKeys.rankings.byType(type), queryFn: () => getRankings(type) });
