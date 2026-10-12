import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { meQueryKey } from '@/hooks/use-me';
import { locales } from '@/i18n/config';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { memberSchema } from '@/types/member';
import { uploadMaxBytes } from '@/utils/downscale-image';

export const nicknameMaxLength = 30;

/** Every field is optional: the API leaves out what is not sent. */
export const updateProfileInputSchema = z.object({
  nickname: z
    .string()
    .trim()
    .min(1)
    .refine((value) => [...value].length <= nicknameMaxLength)
    .optional(),
  language: z.enum(locales).optional(),
  /** Already downscaled to a JPEG (see utils/downscale-image). */
  avatar: z
    .instanceof(File)
    .refine((file) => file.type === 'image/jpeg')
    .refine((file) => file.size <= uploadMaxBytes)
    .optional()
});
export type UpdateProfileInput = z.infer<typeof updateProfileInputSchema>;

export const updateProfileResponseSchema = z.object({ member: memberSchema });

// multipart/form-data: axios sets the boundary header itself when given FormData.
const updateProfile = async (data: UpdateProfileInput) => {
  const input = updateProfileInputSchema.parse(data);
  const form = new FormData();
  if (input.nickname) {
    form.append('nickname', input.nickname);
  }
  if (input.language) {
    form.append('language', input.language);
  }
  if (input.avatar) {
    form.append('avatar', input.avatar);
  }
  const response = await api.patch('/members/me', form);
  return updateProfileResponseSchema.parse(response.data).member;
};

export const useUpdateProfile = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: updateProfile,
    onSuccess: (member) => {
      queryClient.setQueryData(meQueryKey, member);
      // Posts and footprints show the author's nickname and picture.
      queryClient.invalidateQueries({ queryKey: queryKeys.posts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.gpsVisits.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spots.all });
    }
  });
};
