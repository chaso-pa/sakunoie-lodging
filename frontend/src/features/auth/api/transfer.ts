import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { setSignedInMember } from '@/features/auth/api/use-me';
import { api } from '@/lib/api-client';
import { memberSchema } from '@/types/member';

/** Plan section 4.7: 12 characters of A-Z0-9, typed in any case, with or without hyphens and spaces. */
export const normalizeTransferCode = (value: string) => value.replace(/[\s-]/g, '').toUpperCase();

export const transferInputSchema = z.object({
  transfer_code: z
    .string()
    .transform(normalizeTransferCode)
    .pipe(z.string().regex(/^[A-Z0-9]{12}$/))
});
export type TransferInput = z.input<typeof transferInputSchema>;

export const transferResponseSchema = z.object({ member: memberSchema });

const transfer = async (data: TransferInput) => {
  const response = await api.post('/auth/transfer', transferInputSchema.parse(data));
  return transferResponseSchema.parse(response.data).member;
};

export const useTransfer = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: transfer,
    onSuccess: (member) => setSignedInMember(queryClient, member)
  });
};
