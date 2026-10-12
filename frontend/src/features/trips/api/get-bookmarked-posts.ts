import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { type Post, postListSchema } from '@/types/community';

const getBookmarkedPosts = async (): Promise<Post[]> => {
  const response = await api.get('/posts/bookmarked');
  return postListSchema.parse(response.data).posts;
};

/** Posts the member saved, newest first. */
export const useBookmarkedPosts = () => useQuery({ queryKey: queryKeys.posts.bookmarked, queryFn: getBookmarkedPosts });
