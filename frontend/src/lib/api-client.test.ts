import { AxiosError, AxiosHeaders } from 'axios';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, findTokenProvider, registerAuthTokenProvider, toApiError } from '@/lib/api-client';

const responseError = (status: number, data: unknown) =>
  new AxiosError('Request failed', 'ERR_BAD_RESPONSE', undefined, undefined, {
    status,
    statusText: '',
    headers: {},
    config: { headers: new AxiosHeaders() },
    data
  });

describe('toApiError', () => {
  it('normalizes the error envelope', () => {
    const error = toApiError(
      responseError(422, {
        code: 'validation_error',
        message: 'validation failed',
        details: [{ location: 'body.guests[1].passport_number', message: 'expected pattern' }],
        request_id: 'req-12345678'
      })
    );
    expect(error).toBeInstanceOf(ApiError);
    expect(error.status).toBe(422);
    expect(error.code).toBe('validation_error');
    expect(error.details).toEqual([{ location: 'body.guests[1].passport_number', message: 'expected pattern' }]);
    expect(error.requestId).toBe('req-12345678');
  });

  it('maps a missing response to network_error', () => {
    const error = toApiError(new AxiosError('timeout', 'ECONNABORTED'));
    expect(error.status).toBe(0);
    expect(error.code).toBe('network_error');
    expect(error.isNetworkError).toBe(true);
  });

  it('falls back to unknown_error for a non-envelope body', () => {
    expect(toApiError(responseError(502, '<html>bad gateway</html>')).code).toBe('unknown_error');
  });
});

describe('registerAuthTokenProvider', () => {
  it('picks the longest matching prefix', async () => {
    registerAuthTokenProvider('/guest-registrations/manage', () => 'guest-token');
    registerAuthTokenProvider('/admin/', () => 'admin-token');
    expect(await findTokenProvider('/guest-registrations/manage/guests')?.()).toBe('guest-token');
    expect(await findTokenProvider('/admin/guest-registrations')?.()).toBe('admin-token');
    expect(findTokenProvider('/guest-registrations')).toBeUndefined();
    expect(findTokenProvider('/facilities/saku-no-ie/public')).toBeUndefined();
  });
});

describe('cross-origin API base URL', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  const load = async (baseUrl: string | undefined) => {
    vi.resetModules();
    if (baseUrl !== undefined) vi.stubEnv('NEXT_PUBLIC_API_BASE_URL', baseUrl);
    return import('@/lib/api-client');
  };

  it('calls <NEXT_PUBLIC_API_BASE_URL>/api/v1 when it is set', async () => {
    const { api } = await load('https://api.sakunoie.example');
    expect(api.getUri({ url: '/facilities/saku-no-ie/public' })).toBe(
      'https://api.sakunoie.example/api/v1/facilities/saku-no-ie/public'
    );
    expect(api.defaults.withCredentials).toBeFalsy();
  });

  it('falls back to the relative /api/v1 when it is unset', async () => {
    const { api } = await load(undefined);
    expect(api.defaults.baseURL).toBe('/api/v1');
  });

  it('matches token prefixes on the path part, and only for the API origin', async () => {
    const { findTokenProvider, registerAuthTokenProvider } = await load('https://api.sakunoie.example');
    registerAuthTokenProvider('/guest-registrations/manage', () => 'guest-token');
    registerAuthTokenProvider('/admin/', () => 'admin-token');
    expect(await findTokenProvider('/guest-registrations/manage?x=1')?.()).toBe('guest-token');
    expect(await findTokenProvider('https://api.sakunoie.example/api/v1/guest-registrations/manage/guests')?.()).toBe(
      'guest-token'
    );
    expect(await findTokenProvider('https://api.sakunoie.example/api/v1/admin/facilities')?.()).toBe('admin-token');
    // Another origin, a path outside /api/v1, or a protocol-relative URL elsewhere never gets a token.
    expect(findTokenProvider('https://evil.example/api/v1/admin/facilities')).toBeUndefined();
    expect(findTokenProvider('https://api.sakunoie.example/admin/facilities')).toBeUndefined();
    expect(findTokenProvider('//evil.example/api/v1/admin/facilities')).toBeUndefined();
  });

  it('attaches the bearer token in the Authorization header, never in the URL', async () => {
    const { api, registerAuthTokenProvider } = await load('https://api.sakunoie.example');
    registerAuthTokenProvider('/guest-registrations/manage', () => 'secret-token');
    let seen: { url: string; authorization: unknown } | undefined;
    await api.get('/guest-registrations/manage', {
      adapter: async (config) => {
        seen = { url: api.getUri(config), authorization: config.headers.get('Authorization') };
        return { data: {}, status: 200, statusText: 'OK', headers: {}, config };
      }
    });
    expect(seen?.authorization).toBe('Bearer secret-token');
    expect(seen?.url).toBe('https://api.sakunoie.example/api/v1/guest-registrations/manage');
    expect(seen?.url).not.toContain('secret-token');
  });
});
