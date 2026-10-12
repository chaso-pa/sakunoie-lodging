import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { type Post, postListSchema } from '@/types/community';

const getMyPosts = async (): Promise<Post[]> => {
  const response = await api.get('/posts/mine');
  return postListSchema.parse(response.data).posts;
};

/** The signed-in member's own posts, newest first. */
export const useMyPosts = () => useQuery({ queryKey: queryKeys.posts.mine, queryFn: getMyPosts });
