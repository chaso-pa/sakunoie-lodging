import { useInfiniteQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { listOf, postSchema } from '@/types/community';

export const feedPageSize = 20;

export const feedFiltersSchema = z.object({
  /** One of the four regions, or '' for all. */
  area: z.string(),
  /** Weather as stored (Japanese label), or '' for all. */
  weather: z.string(),
  /** Month visited as YYYY-MM, or '' for all. */
  month: z.string().regex(/^$|^\d{4}-(0[1-9]|1[0-2])$/),
  q: z.string().max(100)
});
export type FeedFilters = z.infer<typeof feedFiltersSchema>;

export const feedResponseSchema = z.object({
  posts: listOf(postSchema),
  total: z.number()
});
export type FeedPage = z.infer<typeof feedResponseSchema>;

const getPosts = async (filters: FeedFilters, offset: number): Promise<FeedPage> => {
  const { area, weather, month, q } = feedFiltersSchema.parse(filters);
  const response = await api.get('/posts', {
    params: {
      ...(area && { area }),
      ...(weather && { weather }),
      ...(month && { month }),
      ...(q && { q }),
      sort: 'latest',
      limit: feedPageSize,
      offset
    }
  });
  return feedResponseSchema.parse(response.data);
};

/** The feed, filtered on the server and loaded a page at a time. */
export const useFeedPosts = (filters: FeedFilters) =>
  useInfiniteQuery({
    queryKey: queryKeys.posts.feed(filters),
    queryFn: ({ pageParam }) => getPosts(filters, pageParam),
    initialPageParam: 0,
    getNextPageParam: (lastPage, pages) => {
      const loaded = pages.reduce((count, page) => count + page.posts.length, 0);
      return lastPage.posts.length > 0 && loaded < lastPage.total ? loaded : undefined;
    }
  });
