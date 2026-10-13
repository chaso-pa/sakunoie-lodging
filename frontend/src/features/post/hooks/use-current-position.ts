import { useCallback, useState } from 'react';
import type { LatLng } from '@/features/post/types/place';

export type LocateStatus = 'idle' | 'locating' | 'located' | 'denied' | 'unavailable' | 'timeout' | 'unsupported';

/**
 * Asks the browser for the position, once, each time `locate` is called and never otherwise. The position
 * is handed to the caller and not kept here; only how the request went is.
 */
export const useCurrentPosition = () => {
  const [status, setStatus] = useState<LocateStatus>('idle');

  const locate = useCallback(
    () =>
      new Promise<LatLng | null>((resolve) => {
        if (!('geolocation' in navigator)) {
          setStatus('unsupported');
          resolve(null);
          return;
        }
        setStatus('locating');
        navigator.geolocation.getCurrentPosition(
          (result) => {
            setStatus('located');
            resolve({ latitude: result.coords.latitude, longitude: result.coords.longitude });
          },
          (error) => {
            setStatus(
              error.code === error.PERMISSION_DENIED
                ? 'denied'
                : error.code === error.POSITION_UNAVAILABLE
                  ? 'unavailable'
                  : 'timeout'
            );
            resolve(null);
          },
          { enableHighAccuracy: false, timeout: 10000, maximumAge: 60000 }
        );
      }),
    []
  );

  return { status, locate };
};
