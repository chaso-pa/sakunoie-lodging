import '@/features/admin/api/admin-auth';
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { adminQueryKey } from '@/features/admin/api/admin-auth';
import type {
  AdminInvitation,
  AdminInvitationList,
  CreateInvitationInput,
  CreateInvitationResult,
  InvitationFilter
} from '@/features/admin/types/api';
import { api } from '@/lib/api-client';

export const invitationParams = (f: InvitationFilter) => {
  const params: Record<string, string | number> = { page: f.page ?? 1 };
  if (f.facilityId) params.facility_id = f.facilityId;
  if (f.status) params.status = f.status;
  return params;
};

const invitationsKey = (uid: string) => [...adminQueryKey, uid, 'invitations'] as const;

export const getAdminInvitations = (f: InvitationFilter) =>
  api.get<AdminInvitationList>('/admin/invitations', { params: invitationParams(f) }).then((r) => r.data);

export const createInvitation = (body: CreateInvitationInput) =>
  api.post<CreateInvitationResult>('/admin/invitations', body).then((r) => r.data);

export const revokeInvitation = (publicId: string) =>
  api.post<AdminInvitation>(`/admin/invitations/${encodeURIComponent(publicId)}/revoke`).then((r) => r.data);

export const useAdminInvitations = (uid: string, filter: InvitationFilter) =>
  useQuery({
    queryKey: [...invitationsKey(uid), invitationParams(filter)],
    queryFn: () => getAdminInvitations(filter),
    placeholderData: keepPreviousData
  });

/** The result carries the one-time token: it is kept only by the caller's state, never in the query cache. */
export const useCreateInvitation = (uid: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createInvitation,
    onSettled: () => queryClient.invalidateQueries({ queryKey: invitationsKey(uid) })
  });
};

export const useRevokeInvitation = (uid: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: revokeInvitation,
    onSettled: () => queryClient.invalidateQueries({ queryKey: invitationsKey(uid) })
  });
};
