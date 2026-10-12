import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { locales } from '@/i18n/config';
import { api } from '@/lib/api-client';
import { memberSchema } from '@/types/member';

export const avatarMimeTypes = ['image/jpeg', 'image/png', 'image/webp'];
/** The API rejects request bodies over 10MB (plan section 4.6). */
export const avatarMaxBytes = 10 * 1024 * 1024;
export const nicknameMaxLength = 30;

export const signupInputSchema = z.object({
  nickname: z.string().trim().min(1).max(nicknameMaxLength),
  language: z.enum(locales),
  avatar: z
    .instanceof(File)
    .refine((file) => avatarMimeTypes.includes(file.type))
    .refine((file) => file.size <= avatarMaxBytes)
    .nullish()
});
export type SignupInput = z.infer<typeof signupInputSchema>;

export const signupResponseSchema = z.object({
  member: memberSchema,
  transfer_code: z.string()
});
export type SignupResponse = z.infer<typeof signupResponseSchema>;

// multipart/form-data: axios sets the boundary header itself when given FormData.
const signup = async (data: SignupInput): Promise<SignupResponse> => {
  const input = signupInputSchema.parse(data);
  const form = new FormData();
  form.append('nickname', input.nickname);
  form.append('language', input.language);
  if (input.avatar) {
    form.append('avatar', input.avatar);
  }
  const response = await api.post('/auth/signup', form);
  return signupResponseSchema.parse(response.data);
};

/**
 * Does not touch the `me` cache: the caller shows the transfer code first and signs the member in afterwards
 * (see AuthGate), because the code is returned only by this one response.
 */
export const useSignup = () => useMutation({ mutationFn: signup });
