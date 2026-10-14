import { useSyncExternalStore } from 'react';
import { env } from '@/config/env';
import { getGoogleMapsFailure, subscribeGoogleMapsFailure } from '@/features/post/utils/google-maps-status';

/** Why the Google map could not be used on this page, or null. */
export const useGoogleMapsFailure = () =>
  useSyncExternalStore(subscribeGoogleMapsFailure, getGoogleMapsFailure, () => null);

export type MapKind = 'google' | 'leaflet';

/**
 * Which map the picker draws: Google's when a browser key is configured and it has not failed on this page,
 * the Leaflet map otherwise. Google's terms allow its places only on a Google map, so this also decides
 * whether Google-sourced results may be drawn.
 */
export const useMapKind = (): MapKind => {
  const failed = useGoogleMapsFailure();
  return env.googleMapsEnabled && !failed ? 'google' : 'leaflet';
};
