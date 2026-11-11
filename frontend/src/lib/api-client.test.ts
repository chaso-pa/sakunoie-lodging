import { AxiosError, AxiosHeaders } from 'axios';
import { describe, expect, it } from 'vitest';
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
