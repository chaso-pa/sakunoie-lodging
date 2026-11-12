import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { API_PATH, apiBaseUrl, apiOrigin } from '@/config/api';

export type ApiErrorDetail = { location: string; message: string };

/**
 * Normalized API failure. `code` is the stable error code from the envelope
 * `{code,message,details,request_id}` and is what the UI localizes; `message` is
 * developer-facing English and is never shown to guests. Network failures and
 * timeouts have `status` 0 and code `network_error`.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details: ApiErrorDetail[];
  readonly requestId: string | null;

  constructor(status: number, code: string, message: string, details: ApiErrorDetail[], requestId: string | null) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
    this.requestId = requestId;
  }

  get isNetworkError() {
    return this.status === 0;
  }
}

const isEnvelope = (
  data: unknown
): data is { code: string; message?: string; details?: unknown; request_id?: string } =>
  typeof data === 'object' && data !== null && typeof (data as { code?: unknown }).code === 'string';

export const toApiError = (error: unknown): ApiError => {
  if (error instanceof ApiError) return error;
  if (error instanceof AxiosError) {
    const response = error.response;
    if (!response) return new ApiError(0, 'network_error', error.message, [], null);
    const data = response.data;
    if (isEnvelope(data)) {
      const details = Array.isArray(data.details)
        ? data.details.filter(
            (d): d is ApiErrorDetail =>
              typeof d === 'object' && d !== null && typeof d.location === 'string' && typeof d.message === 'string'
          )
        : [];
      return new ApiError(response.status, data.code, data.message ?? '', details, data.request_id ?? null);
    }
    return new ApiError(response.status, 'unknown_error', error.message, [], null);
  }
  return new ApiError(0, 'unknown_error', error instanceof Error ? error.message : 'unknown', [], null);
};

type TokenProvider = () => Promise<string | null> | string | null;
const tokenProviders = new Map<string, TokenProvider>();

/**
 * Registers a bearer-token source for requests whose API path (relative to `/api/v1`) starts with
 * `prefix` (e.g. `/guest-registrations/manage` or `/admin/`). Features register their own provider so
 * this shared module never imports a feature. Re-registering replaces.
 */
export const registerAuthTokenProvider = (prefix: string, provider: TokenProvider) => {
  tokenProviders.set(prefix, provider);
};

/**
 * The path part used for prefix matching. A relative request URL (the normal case: axios joins it with
 * `baseURL`) loses its query and fragment. An absolute URL counts only when it points at the API origin
 * under `/api/v1`, so a token is never attached to a request for another origin.
 */
const apiRequestPath = (url: string): string | undefined => {
  if (!/^([a-z][a-z\d+.-]*:)?\/\//i.test(url)) return url.split(/[?#]/, 1)[0];
  if (!apiOrigin) return undefined;
  const parsed = new URL(url, apiOrigin);
  if (parsed.origin !== apiOrigin || !parsed.pathname.startsWith(`${API_PATH}/`)) return undefined;
  return parsed.pathname.slice(API_PATH.length);
};

export const findTokenProvider = (url: string | undefined): TokenProvider | undefined => {
  const path = url ? apiRequestPath(url) : undefined;
  if (!path) return undefined;
  let match: { prefix: string; provider: TokenProvider } | undefined;
  for (const [prefix, provider] of tokenProviders) {
    if (path.startsWith(prefix) && (!match || prefix.length > match.prefix.length)) match = { prefix, provider };
  }
  return match?.provider;
};

/**
 * Cross-origin when `NEXT_PUBLIC_API_BASE_URL` is set (`@/config/api`). Tokens travel only in the
 * `Authorization` header; the API allows no credentials (cookies), so `withCredentials` stays false.
 */
export const api = axios.create({
  baseURL: apiBaseUrl,
  timeout: 20_000,
  headers: { Accept: 'application/json' }
});

api.interceptors.request.use(async (config: InternalAxiosRequestConfig) => {
  const provider = findTokenProvider(config.url);
  if (provider) {
    const token = await provider();
    if (token) config.headers.set('Authorization', `Bearer ${token}`);
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error: unknown) => Promise.reject(toApiError(error))
);
