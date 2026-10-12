import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/lib/api-client';

export const verifyInviteInputSchema = z.object({ code: z.string().trim().min(1) });
export type VerifyInviteInput = z.infer<typeof verifyInviteInputSchema>;

export const verifyInviteResponseSchema = z.object({ valid: z.literal(true) });

// The code is checked by the server only; on success the session is marked as invite-verified.
const verifyInvite = async (data: VerifyInviteInput) => {
  const response = await api.post('/auth/verify-invite', verifyInviteInputSchema.parse(data));
  return verifyInviteResponseSchema.parse(response.data);
};

export const useVerifyInvite = () => useMutation({ mutationFn: verifyInvite });
