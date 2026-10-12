import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { setSignedInMember } from '@/hooks/use-me';
import { api } from '@/lib/api-client';

export const logoutResponseSchema = z.object({ ok: z.boolean() });

const logout = async () => {
  const response = await api.post('/auth/logout', {});
  return logoutResponseSchema.parse(response.data);
};

export const useLogout = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: logout,
    onSuccess: () => setSignedInMember(queryClient, null)
  });
};
