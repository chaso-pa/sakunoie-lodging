/**
 * Where the browser sends API calls (root CLAUDE.md §13 "Decision B revised").
 *
 * `NEXT_PUBLIC_API_BASE_URL` (inlined at build time) is the API origin, e.g. `https://api.sakunoie.example`;
 * the browser then calls `${origin}/api/v1` cross-origin under the API's CORS allowlist. When it is unset,
 * calls stay relative (`/api/v1`) and `next.config.ts` rewrites them to `API_PROXY_TARGET` (dev only).
 */
export const API_PATH = '/api/v1';

/** Returns the origin of an API base URL, or undefined when unset. Throws on anything but an http(s) origin. */
export const parseApiBaseUrl = (value: string | undefined): string | undefined => {
  const raw = value?.trim();
  if (!raw) return undefined;
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    throw new Error(`NEXT_PUBLIC_API_BASE_URL is not a URL: ${raw}`);
  }
  if (
    (url.protocol !== 'https:' && url.protocol !== 'http:') ||
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    url.pathname !== '/'
  ) {
    throw new Error(`NEXT_PUBLIC_API_BASE_URL must be an origin such as https://api.sakunoie.example, got ${raw}`);
  }
  return url.origin;
};

/** The API origin, or undefined in same-origin (rewrite) mode. */
export const apiOrigin = parseApiBaseUrl(process.env.NEXT_PUBLIC_API_BASE_URL);

/** axios `baseURL`: absolute in cross-origin mode, relative otherwise. */
export const apiBaseUrl = apiOrigin ? `${apiOrigin}${API_PATH}` : API_PATH;
