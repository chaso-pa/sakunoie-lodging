import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { locales } from '@/i18n/config';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { postSchema } from '@/types/community';
import { uploadMaxBytes } from '@/utils/downscale-image';

export const captionMaxLength = 2000;
export const tagsMaxLength = 300;

export const createPostInputSchema = z
  .object({
    caption: z.string().trim().max(captionMaxLength),
    /** Already downscaled to a JPEG (see utils/downscale-image). The API accepts one photo per post. */
    media: z
      .instanceof(File)
      .refine((file) => file.type === 'image/jpeg')
      .refine((file) => file.size <= uploadMaxBytes)
      .nullable(),
    spot_id: z.string(),
    language: z.enum(locales),
    weather: z.string().max(20),
    temperature: z.number().min(-60).max(60).nullable(),
    visited_at: z.string().regex(/^$|^\d{4}-\d{2}-\d{2}$/),
    /** Used by the API only when no spot is chosen. */
    area: z.string(),
    tags: z.string().max(tagsMaxLength)
  })
  .refine((value) => value.caption !== '' || value.media !== null);
export type CreatePostInput = z.infer<typeof createPostInputSchema>;

export const createPostResponseSchema = z.object({ post: postSchema });

// multipart/form-data: axios sets the boundary header itself when given FormData.
const createPost = async (data: CreatePostInput) => {
  const input = createPostInputSchema.parse(data);
  const form = new FormData();
  form.append('caption', input.caption);
  form.append('language', input.language);
  if (input.spot_id) {
    form.append('spot_id', input.spot_id);
  } else if (input.area) {
    form.append('area', input.area);
  }
  if (input.weather) {
    form.append('weather', input.weather);
  }
  if (input.temperature !== null) {
    form.append('temperature', String(input.temperature));
  }
  if (input.visited_at) {
    form.append('visited_at', input.visited_at);
  }
  if (input.tags.trim()) {
    form.append('tags', input.tags.trim());
  }
  if (input.media) {
    form.append('media', input.media);
  }
  const response = await api.post('/posts', form);
  return createPostResponseSchema.parse(response.data).post;
};

export const useCreatePost = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createPost,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.posts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.rankings.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spots.all });
    }
  });
};
