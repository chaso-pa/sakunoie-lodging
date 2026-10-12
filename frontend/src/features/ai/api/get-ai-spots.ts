import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { listOf } from '@/types/community';

/** Genres as the API stores them (Japanese); the UI shows a translated label for each. */
export const aiGenres = ['穴場スポット', '自然・絶景', '歴史・文化', 'カフェ・グルメ'] as const;
export const aiPurposes = ['all', 'karuizawa', 'onsen', 'hiking', 'japan', 'scenic', 'rainyday', 'family'] as const;
export const aiWeathers = ['any', 'sunny', 'rainy'] as const;
export const aiDurations = ['half', 'short', 'full'] as const;
export const aiTransports = ['car', 'transit'] as const;

export const aiFiltersSchema = z.object({
  genre: z.enum(aiGenres),
  purpose: z.enum(aiPurposes),
  weather: z.enum(aiWeathers),
  duration: z.enum(aiDurations),
  transport: z.enum(aiTransports)
});
export type AiFilters = z.infer<typeof aiFiltersSchema>;

const httpUrl = z.string().refine((value) => /^https?:\/\//i.test(value));

export const aiSpotSchema = z.object({
  id: z.string(),
  name: z.string(),
  area: z.string(),
  kind: z.string(),
  drive_min: z.number(),
  transit_min: z.number(),
  stay_min: z.number(),
  description: z.string(),
  map_query: z.string(),
  // Rendered as a link, so anything but http(s) is refused.
  search_url: httpUrl
});
export type AiSpot = z.infer<typeof aiSpotSchema>;

export const aiSpotsResponseSchema = z.object({ spots: listOf(aiSpotSchema) });

const getAiSpots = async (filters: AiFilters): Promise<AiSpot[]> => {
  const response = await api.get('/ai-spots', { params: aiFiltersSchema.parse(filters) });
  return aiSpotsResponseSchema.parse(response.data).spots;
};

/** Curated spots matching the chosen conditions. Runs once the member asks for recommendations. */
export const useAiSpots = (filters: AiFilters | null) =>
  useQuery({
    queryKey: queryKeys.aiSpots(filters ?? {}),
    queryFn: () => getAiSpots(filters as AiFilters),
    enabled: filters !== null
  });
