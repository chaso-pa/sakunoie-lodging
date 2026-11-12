import '@/features/admin/api/admin-auth';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { adminQueryKey } from '@/features/admin/api/admin-auth';
import { registrationQueryKey } from '@/features/admin/api/get-registration';
import type {
  AdminPatchRegistration,
  AdminRegistrationDetail,
  ReissueEditLinkResult
} from '@/features/admin/types/api';
import { api } from '@/lib/api-client';

const base = (id: string) => `/admin/guest-registrations/${encodeURIComponent(id)}`;

export const patchAdminRegistration = (id: string, version: string, body: AdminPatchRegistration) =>
  api.patch<AdminRegistrationDetail>(base(id), body, { headers: { 'If-Match': `"${version}"` } }).then((r) => r.data);

export const verifyIdentity = (id: string) =>
  api.post<AdminRegistrationDetail>(`${base(id)}/verify-identity`).then((r) => r.data);

export const reissueEditLink = (id: string) =>
  api.post<ReissueEditLinkResult>(`${base(id)}/reissue-edit-link`).then((r) => r.data);

/** Refresh the detail and the ledger after any change. */
const useInvalidate = (uid: string, id: string) => {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: registrationQueryKey(uid, id) }),
      queryClient.invalidateQueries({ queryKey: [...adminQueryKey, uid, 'registrations'] })
    ]);
};

export const usePatchAdminRegistration = (uid: string, id: string) => {
  const invalidate = useInvalidate(uid, id);
  return useMutation({
    mutationFn: ({ version, body }: { version: string; body: AdminPatchRegistration }) =>
      patchAdminRegistration(id, version, body),
    onSettled: invalidate
  });
};

export const useVerifyIdentity = (uid: string, id: string) => {
  const invalidate = useInvalidate(uid, id);
  return useMutation({ mutationFn: () => verifyIdentity(id), onSettled: invalidate });
};

export const useReissueEditLink = (uid: string, id: string) => {
  const invalidate = useInvalidate(uid, id);
  return useMutation({ mutationFn: () => reissueEditLink(id), onSettled: invalidate });
};
