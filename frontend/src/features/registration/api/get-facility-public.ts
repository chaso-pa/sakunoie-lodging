import { useQuery } from '@tanstack/react-query';
import type { FacilityPublic } from '@/features/registration/types/api';
import { api } from '@/lib/api-client';

export const getFacilityPublic = (code: string) =>
  api.get<FacilityPublic>(`/facilities/${encodeURIComponent(code)}/public`).then((r) => r.data);

export const useFacilityPublic = (code: string) =>
  useQuery({ queryKey: ['facility-public', code], queryFn: () => getFacilityPublic(code), staleTime: 5 * 60_000 });
