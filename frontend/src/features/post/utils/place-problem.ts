import { isAxiosError } from 'axios';

/**
 * `provider` is the search service failing or being busy (worth another try), `notConfigured` the
 * deployment having none (not worth one).
 */
export type PlaceProblem = 'network' | 'rateLimited' | 'notConfigured' | 'provider' | 'generic';

const defaultRetryAfterSeconds = 10;

/** Why a place lookup failed, as far as the member needs to tell the cases apart. */
export const getPlaceProblem = (error: unknown): PlaceProblem => {
  if (!isAxiosError(error)) {
    // Not an HTTP failure: the response did not match the schema. Say so where a developer will see it.
    console.error('Unexpected place API response', error);
    return 'generic';
  }
  const status = error.response?.status;
  if (status === undefined) {
    return 'network';
  }
  if (status === 429) {
    return 'rateLimited';
  }
  if (status === 503) {
    // The API answers 503 both when the provider's queue is full and when no provider is configured.
    // Only the first carries Retry-After (docs/spot-search.md, section 4).
    return error.response?.headers?.['retry-after'] ? 'provider' : 'notConfigured';
  }
  if (status === 502 || status === 504) {
    return 'provider';
  }
  return 'generic';
};

const maxRetryAfterSeconds = 60;

/**
 * Seconds to wait after a 429: from `Retry-After`, or from the `X-RateLimit-Reset` time the API's limiter
 * sends, and a short default when neither is there.
 */
export const getRetryAfterSeconds = (error: unknown): number => {
  if (!isAxiosError(error)) {
    return defaultRetryAfterSeconds;
  }
  const headers = error.response?.headers;
  const retryAfter = Number(headers?.['retry-after']);
  const reset = Number(headers?.['x-ratelimit-reset']);
  const seconds =
    Number.isFinite(retryAfter) && retryAfter > 0
      ? retryAfter
      : Number.isFinite(reset) && reset > 0
        ? reset - Date.now() / 1000
        : defaultRetryAfterSeconds;
  return Math.min(Math.max(Math.ceil(seconds), 1), maxRetryAfterSeconds);
};
