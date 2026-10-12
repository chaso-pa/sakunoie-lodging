import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';

export const weatherResponseSchema = z.object({
  /** Japanese label, the same vocabulary posts store (晴れ, くもり, …). */
  weather: z.string(),
  temperature: z.number(),
  icon: z.string(),
  code: z.number(),
  source: z.string()
});
export type Weather = z.infer<typeof weatherResponseSchema>;

// No coordinates are sent: the API then reports the weather around Saku City, so composing a post
// never needs the member's location.
const getWeather = async (): Promise<Weather> => {
  const response = await api.get('/weather');
  return weatherResponseSchema.parse(response.data);
};

export const useWeather = () =>
  useQuery({ queryKey: queryKeys.weather, queryFn: getWeather, staleTime: 10 * 60 * 1000, retry: false });
