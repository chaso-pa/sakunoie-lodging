import '@/features/registration/api/invitation-auth';
import { useQuery } from '@tanstack/react-query';
import type { CurrentInvitation } from '@/features/registration/types/api';
import { api } from '@/lib/api-client';

export const currentInvitationQueryKey = ['registration', 'invitation'] as const;

/** The host's invitation (Bearer from sessionStorage): facility and the host-set stay dates. */
export const getCurrentInvitation = () => api.get<CurrentInvitation>('/invitations/current').then((r) => r.data);

export const useCurrentInvitation = (enabled: boolean) =>
  useQuery({
    queryKey: currentInvitationQueryKey,
    queryFn: getCurrentInvitation,
    enabled,
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: 0
  });
