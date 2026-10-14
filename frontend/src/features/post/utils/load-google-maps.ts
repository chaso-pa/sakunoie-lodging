import { importLibrary, setOptions } from '@googlemaps/js-api-loader';
import { reportGoogleMapsFailure } from '@/features/post/utils/google-maps-status';

declare global {
  interface Window {
    /** Called by the Maps JavaScript API when it refuses the key (documented as its auth-error hook). */
    gm_authFailure?: () => void;
  }
}

// The browser key is read here and nowhere else, so it is inlined only into the chunk of the Google map,
// which is loaded when the map opens. It is a public key by design (restricted by HTTP referrer and to the
// Maps JavaScript API in the Cloud console); the server key never reaches the frontend.
const apiKey = process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY ?? '';

/**
 * Map ID of the map style. Advanced Markers need one; `DEMO_MAP_ID` is Google's ID for trying them out and
 * is used only when none is configured.
 */
export const googleMapId = process.env.NEXT_PUBLIC_GOOGLE_MAPS_MAP_ID || 'DEMO_MAP_ID';

const loadTimeoutMs = 15_000;

/** Maps JavaScript API language codes for the app's locales. */
const mapLanguage = (locale: string) => (locale === 'zh' ? 'zh-CN' : locale);

export type GoogleMapsLibraries = {
  Map: typeof google.maps.Map;
  AdvancedMarkerElement: typeof google.maps.marker.AdvancedMarkerElement;
  ControlPosition: typeof google.maps.ControlPosition;
};

let loading: Promise<GoogleMapsLibraries> | null = null;

/**
 * Loads the Maps JavaScript API, once per page: the language of the first call stays until the page is
 * reloaded, which is how the API works. A failure is reported to `google-maps-status`, which switches the
 * picker to the Leaflet map, and the returned promise rejects.
 */
export const loadGoogleMaps = (locale: string): Promise<GoogleMapsLibraries> => {
  if (!loading) {
    window.gm_authFailure = () => reportGoogleMapsFailure('auth');
    setOptions({ key: apiKey, v: 'weekly', language: mapLanguage(locale), region: 'JP' });
    const libraries = Promise.all([importLibrary('maps'), importLibrary('marker'), importLibrary('core')]).then(
      ([maps, marker, core]) => ({
        Map: maps.Map,
        AdvancedMarkerElement: marker.AdvancedMarkerElement,
        ControlPosition: core.ControlPosition
      })
    );
    const timeout = new Promise<never>((_, reject) => setTimeout(() => reject(new Error('timeout')), loadTimeoutMs));
    loading = Promise.race([libraries, timeout]);
    loading.catch((error: unknown) =>
      reportGoogleMapsFailure(error instanceof Error && error.message === 'timeout' ? 'timeout' : 'load')
    );
  }
  return loading;
};
