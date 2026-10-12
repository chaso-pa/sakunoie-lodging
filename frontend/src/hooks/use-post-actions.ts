import { type QueryClient, type QueryKey, useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { okResponseSchema, type Post } from '@/types/community';

export const toggleLikeResponseSchema = z.object({ liked: z.boolean(), like_count: z.number() });
export const toggleBookmarkResponseSchema = z.object({ bookmarked: z.boolean() });

type PostPage = { posts: Post[] };
type Snapshot = [QueryKey, unknown][];

const isPostPage = (value: unknown): value is PostPage =>
  typeof value === 'object' && value !== null && Array.isArray((value as PostPage).posts);

/**
 * Applies `change` to one post wherever it is cached. Post queries hold either a plain array (own and
 * saved posts) or infinite pages of `{ posts }` (the feed).
 */
const mapCachedPosts = (data: unknown, postId: string, change: (post: Post) => Post): unknown => {
  const mapList = (posts: Post[]) => posts.map((post) => (post.id === postId ? change(post) : post));
  if (Array.isArray(data)) {
    return mapList(data as Post[]);
  }
  if (typeof data === 'object' && data !== null && Array.isArray((data as { pages?: unknown }).pages)) {
    const infinite = data as { pages: unknown[] };
    return {
      ...infinite,
      pages: infinite.pages.map((page) => (isPostPage(page) ? { ...page, posts: mapList(page.posts) } : page))
    };
  }
  return data;
};

const patchPost = (queryClient: QueryClient, postId: string, change: (post: Post) => Post) =>
  queryClient.setQueriesData({ queryKey: queryKeys.posts.all }, (data: unknown) =>
    mapCachedPosts(data, postId, change)
  );

/** Shows the change at once and returns what to put back if the request fails. */
const applyOptimistic = async (queryClient: QueryClient, postId: string, change: (post: Post) => Post) => {
  await queryClient.cancelQueries({ queryKey: queryKeys.posts.all });
  const snapshot: Snapshot = queryClient.getQueriesData({ queryKey: queryKeys.posts.all });
  patchPost(queryClient, postId, change);
  return { snapshot };
};

const restore = (queryClient: QueryClient, snapshot: Snapshot | undefined) => {
  for (const [key, data] of snapshot ?? []) {
    queryClient.setQueryData(key, data);
  }
};

const toggleLike = async (postId: string) => {
  const response = await api.post(`/posts/${encodeURIComponent(postId)}/like`, {});
  return toggleLikeResponseSchema.parse(response.data);
};

/** Like or unlike a post. The heart flips immediately and is corrected by the server's answer. */
export const useToggleLike = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: toggleLike,
    onMutate: (postId) =>
      applyOptimistic(queryClient, postId, (post) => ({
        ...post,
        is_liked: !post.is_liked,
        like_count: Math.max(0, post.like_count + (post.is_liked ? -1 : 1))
      })),
    onError: (_error, _postId, context) => restore(queryClient, context?.snapshot),
    onSuccess: (result, postId) =>
      patchPost(queryClient, postId, (post) => ({ ...post, is_liked: result.liked, like_count: result.like_count })),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.rankings.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spots.all });
    }
  });
};

const toggleBookmark = async (postId: string) => {
  const response = await api.post(`/posts/${encodeURIComponent(postId)}/bookmark`, {});
  return toggleBookmarkResponseSchema.parse(response.data);
};

/** Save or unsave a post, optimistically. */
export const useToggleBookmark = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: toggleBookmark,
    onMutate: (postId) =>
      applyOptimistic(queryClient, postId, (post) => ({ ...post, is_bookmarked: !post.is_bookmarked })),
    onError: (_error, _postId, context) => restore(queryClient, context?.snapshot),
    onSuccess: (result, postId) =>
      patchPost(queryClient, postId, (post) => ({ ...post, is_bookmarked: result.bookmarked })),
    onSettled: () => {
      // The saved list gains or loses a row, and saves count towards the spot ranking.
      queryClient.invalidateQueries({ queryKey: queryKeys.posts.bookmarked });
      queryClient.invalidateQueries({ queryKey: queryKeys.rankings.all });
    }
  });
};

const deletePost = async (postId: string) => {
  const response = await api.delete(`/posts/${encodeURIComponent(postId)}`);
  return okResponseSchema.parse(response.data);
};

/** Delete one of the member's own posts. The API refuses anyone else's with 403. */
export const useDeletePost = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deletePost,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.posts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.rankings.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spots.all });
    }
  });
};
