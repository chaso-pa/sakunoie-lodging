import { type QueryClient, useQuery } from '@tanstack/react-query';
import { z } from 'zod';
import { api, getApiErrorStatus } from '@/lib/api-client';
import { type Member, memberSchema } from '@/types/member';

export const meResponseSchema = z.object({ member: memberSchema });

export const meQueryKey = ['auth', 'me'] as const;

/** Resolves to null when there is no session (401), so "signed out" is a state rather than an error. */
const getMe = async (): Promise<Member | null> => {
  try {
    const response = await api.get('/auth/me');
    return meResponseSchema.parse(response.data).member;
  } catch (error) {
    if (getApiErrorStatus(error) === 401) {
      return null;
    }
    throw error;
  }
};

export const useMe = () => useQuery({ queryKey: meQueryKey, queryFn: getMe });

/**
 * Records who is signed in on this browser (null = nobody) and drops every other cached response,
 * since those belong to the previous session. The `me` query is updated in place rather than cleared:
 * `queryClient.clear()` would detach the mounted AuthGate from it and the screen would not change.
 */
export const setSignedInMember = (queryClient: QueryClient, member: Member | null) => {
  queryClient.setQueryData(meQueryKey, member);
  queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== meQueryKey[0] });
};
