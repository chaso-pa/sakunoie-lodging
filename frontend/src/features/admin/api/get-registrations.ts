import '@/features/admin/api/admin-auth';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { adminQueryKey } from '@/features/admin/api/admin-auth';
import type { AdminRegistrationList, LedgerFilter } from '@/features/admin/types/api';
import { api } from '@/lib/api-client';

export const ledgerParams = (f: LedgerFilter) => {
  const params: Record<string, string | number> = { page: f.page ?? 1 };
  if (f.facilityId) params.facility_id = f.facilityId;
  if (f.from) params.from = f.from;
  if (f.to) params.to = f.to;
  if (f.q?.trim()) params.q = f.q.trim();
  return params;
};

export const getAdminRegistrations = (f: LedgerFilter) =>
  api.get<AdminRegistrationList>('/admin/guest-registrations', { params: ledgerParams(f) }).then((r) => r.data);

export const useAdminRegistrations = (uid: string, filter: LedgerFilter) =>
  useQuery({
    queryKey: [...adminQueryKey, uid, 'registrations', ledgerParams(filter)],
    queryFn: () => getAdminRegistrations(filter),
    placeholderData: keepPreviousData
  });
