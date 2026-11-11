import '@/features/registration/api/manage-auth';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { manageSummaryQueryKey } from '@/features/registration/api/get-manage-summary';
import type { ManageSummary } from '@/features/registration/types/api';
import type { CompanionPayload } from '@/features/registration/types/schemas';
import { api } from '@/lib/api-client';

export const addCompanions = (guests: CompanionPayload[]) =>
  api.post<ManageSummary>('/guest-registrations/manage/guests', { guests }).then((r) => r.data);

export const useAddCompanions = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: addCompanions,
    onSuccess: (summary) => queryClient.setQueryData(manageSummaryQueryKey, summary)
  });
};
