import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import { type GpsVisit, gpsVisitListSchema } from '@/types/community';

const getMyGpsVisits = async (): Promise<GpsVisit[]> => {
  const response = await api.get('/gps-visits');
  return gpsVisitListSchema.parse(response.data).visits;
};

/** The member's own footprints, public or not. These always carry their coordinates when recorded. */
export const useMyGpsVisits = () => useQuery({ queryKey: queryKeys.gpsVisits.mine, queryFn: getMyGpsVisits });

const getPublicGpsVisits = async (): Promise<GpsVisit[]> => {
  const response = await api.get('/gps-visits/public');
  return gpsVisitListSchema.parse(response.data).visits;
};

/**
 * Footprints members made public. The API returns another member's coordinates only when that member
 * chose to publish the exact position; otherwise `latitude` and `longitude` arrive as null.
 */
export const usePublicGpsVisits = () => useQuery({ queryKey: queryKeys.gpsVisits.public, queryFn: getPublicGpsVisits });
