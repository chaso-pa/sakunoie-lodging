import '@/features/admin/api/admin-auth';
import { useQuery } from '@tanstack/react-query';
import { adminQueryKey } from '@/features/admin/api/admin-auth';
import type { AdminRegistrationDetail } from '@/features/admin/types/api';
import { api } from '@/lib/api-client';

export const registrationQueryKey = (uid: string, id: string) => [...adminQueryKey, uid, 'registration', id] as const;

/** `reveal` returns address/phone/email unmasked; the server writes an audit row for it. */
export const getAdminRegistration = (id: string, reveal: boolean) =>
  api
    .get<AdminRegistrationDetail>(`/admin/guest-registrations/${encodeURIComponent(id)}`, {
      params: reveal ? { reveal: true } : undefined
    })
    .then((r) => r.data);

export const useAdminRegistration = (uid: string, id: string, reveal: boolean) =>
  useQuery({
    queryKey: [...registrationQueryKey(uid, id), reveal ? 'revealed' : 'masked'],
    queryFn: () => getAdminRegistration(id, reveal),
    // Unmasked PII is not kept in the cache once the view goes back to masked.
    gcTime: reveal ? 0 : undefined
  });
