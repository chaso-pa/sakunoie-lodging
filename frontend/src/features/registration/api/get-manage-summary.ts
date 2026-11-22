import '@/features/registration/api/manage-auth';
import { useQuery } from '@tanstack/react-query';
import type { ManageSummary } from '@/features/registration/types/api';
import { ApiError, api } from '@/lib/api-client';

export const manageSummaryQueryKey = ['registration', 'manage-summary'] as const;

export const getManageSummary = () => api.get<ManageSummary>('/guest-registrations/manage').then((r) => r.data);

export const useManageSummary = (enabled: boolean) =>
  useQuery({
    queryKey: manageSummaryQueryKey,
    queryFn: getManageSummary,
    enabled,
    // Token answers (401/410) are final; only transient failures are retried.
    retry: (count, error) => count < 2 && error instanceof ApiError && (error.isNetworkError || error.status >= 500)
  });
