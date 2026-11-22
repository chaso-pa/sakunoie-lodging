import '@/features/admin/api/admin-auth';
import { useQuery } from '@tanstack/react-query';
import { adminQueryKey } from '@/features/admin/api/admin-auth';
import type { AdminFacility } from '@/features/admin/types/api';
import { api } from '@/lib/api-client';

export const getAdminFacilities = () =>
  api.get<{ facilities: AdminFacility[] }>('/admin/facilities').then((r) => r.data.facilities);

/** Also the authorization probe of the console: 403 means "signed in, but not an admin". */
export const useAdminFacilities = (uid: string | null) =>
  useQuery({
    queryKey: [...adminQueryKey, uid, 'facilities'],
    queryFn: getAdminFacilities,
    enabled: uid !== null,
    staleTime: 5 * 60_000
  });
