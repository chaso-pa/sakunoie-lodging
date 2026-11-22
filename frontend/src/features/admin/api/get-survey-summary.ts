import '@/features/admin/api/admin-auth';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { adminQueryKey } from '@/features/admin/api/admin-auth';
import type { SurveySummary, SurveySummaryFilter } from '@/features/admin/types/api';
import { api } from '@/lib/api-client';

export const surveySummaryParams = (f: SurveySummaryFilter) => {
  const params: Record<string, string | number> = {};
  if (f.facilityId) params.facility_id = f.facilityId;
  if (f.from) params.from = f.from;
  if (f.to) params.to = f.to;
  return params;
};

export const getSurveySummary = (f: SurveySummaryFilter) =>
  api.get<SurveySummary>('/admin/surveys/summary', { params: surveySummaryParams(f) }).then((r) => r.data);

export const useSurveySummary = (uid: string, filter: SurveySummaryFilter) =>
  useQuery({
    queryKey: [...adminQueryKey, uid, 'survey-summary', surveySummaryParams(filter)],
    queryFn: () => getSurveySummary(filter),
    placeholderData: keepPreviousData
  });
