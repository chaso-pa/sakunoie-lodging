import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';

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
 * Registers a bearer-token source for requests whose URL starts with `prefix`
 * (e.g. `/guest-registrations/manage` or `/admin/`). Features register their own
 * provider so this shared module never imports a feature. Re-registering replaces.
 */
export const registerAuthTokenProvider = (prefix: string, provider: TokenProvider) => {
  tokenProviders.set(prefix, provider);
};

export const findTokenProvider = (url: string | undefined): TokenProvider | undefined => {
  if (!url) return undefined;
  let match: { prefix: string; provider: TokenProvider } | undefined;
  for (const [prefix, provider] of tokenProviders) {
    if (url.startsWith(prefix) && (!match || prefix.length > match.prefix.length)) match = { prefix, provider };
  }
  return match?.provider;
};

export const api = axios.create({
  baseURL: '/api/v1',
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
