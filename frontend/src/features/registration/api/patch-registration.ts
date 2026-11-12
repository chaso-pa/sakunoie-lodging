import '@/features/registration/api/manage-auth';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { manageSummaryQueryKey } from '@/features/registration/api/get-manage-summary';
import type { ManageSummary } from '@/features/registration/types/api';
import type { PatchRegistrationBody } from '@/features/registration/utils/build-payload';
import { api } from '@/lib/api-client';

export const patchRegistration = ({ body, version }: { body: PatchRegistrationBody; version: string }) =>
  api
    .patch<ManageSummary>('/guest-registrations/manage', body, { headers: { 'If-Match': version } })
    .then((r) => r.data);

export const usePatchRegistration = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: patchRegistration,
    onSuccess: (summary) => queryClient.setQueryData(manageSummaryQueryKey, summary)
  });
};
