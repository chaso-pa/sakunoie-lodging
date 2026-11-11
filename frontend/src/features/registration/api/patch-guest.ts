import '@/features/registration/api/manage-auth';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { manageSummaryQueryKey } from '@/features/registration/api/get-manage-summary';
import type { ManageSummary } from '@/features/registration/types/api';
import type { PatchGuestBody } from '@/features/registration/utils/build-payload';
import { api } from '@/lib/api-client';

export const patchGuest = ({
  guestPublicId,
  body,
  version
}: {
  guestPublicId: string;
  body: PatchGuestBody;
  version: string;
}) =>
  api
    .patch<ManageSummary>(`/guest-registrations/manage/guests/${encodeURIComponent(guestPublicId)}`, body, {
      headers: { 'If-Match': version }
    })
    .then((r) => r.data);

export const usePatchGuest = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: patchGuest,
    onSuccess: (summary) => queryClient.setQueryData(manageSummaryQueryKey, summary)
  });
};
