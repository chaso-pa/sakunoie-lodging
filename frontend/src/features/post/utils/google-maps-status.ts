/**
 * Why the Google map cannot be used: `auth` is the key being refused (invalid, not allowed on this site,
 * API not enabled, billing), reported by the Maps JavaScript API through `gm_authFailure`; `load` the script
 * not loading (offline, blocked by an extension or a network filter); `timeout` it not answering in time.
 */
export type GoogleMapsFailure = 'auth' | 'load' | 'timeout';

let failure: GoogleMapsFailure | null = null;
const listeners = new Set<() => void>();

/**
 * Records that the Google map cannot be used. It stays so until the page is reloaded: the Maps JavaScript
 * API loads once per page and cannot be loaded again with other settings.
 */
export const reportGoogleMapsFailure = (reason: GoogleMapsFailure) => {
  if (failure) {
    return;
  }
  failure = reason;
  for (const listener of listeners) {
    listener();
  }
};

export const getGoogleMapsFailure = () => failure;

export const subscribeGoogleMapsFailure = (listener: () => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};
