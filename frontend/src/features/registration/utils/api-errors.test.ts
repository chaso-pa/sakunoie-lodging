import { describe, expect, it } from 'vitest';
import { errorMessageKey, parseGuestLocation, wizardPathFromLocation } from '@/features/registration/utils/api-errors';
import { ApiError } from '@/lib/api-client';

describe('wizardPathFromLocation', () => {
  it('maps guests[0] to the representative and guests[n] to companion n-1', () => {
    expect(wizardPathFromLocation('body.guests[0].passport_number')).toBe('representative.passportNumber');
    expect(wizardPathFromLocation('body.guests[2].full_name')).toBe('companions.1.fullName');
  });

  it('maps stay and consent fields', () => {
    expect(wizardPathFromLocation('body.check_out_date')).toBe('stay.checkOutDate');
    expect(wizardPathFromLocation('body.planned_guest_count')).toBe('stay.plannedGuestCount');
    expect(wizardPathFromLocation('body.terms_accepted')).toBe('termsAccepted');
  });

  it('returns null for unknown locations', () => {
    expect(wizardPathFromLocation('header.Idempotency-Key')).toBeNull();
    expect(wizardPathFromLocation('body.guests')).toBeNull();
    expect(wizardPathFromLocation('body.facility_code')).toBeNull();
  });

  it('parses manage guest locations', () => {
    expect(parseGuestLocation('body.guests[1].name_kana')).toEqual({ index: 1, field: 'nameKana' });
  });
});

describe('errorMessageKey', () => {
  const apiError = (status: number, code: string) => new ApiError(status, code, 'msg', [], null);

  it('localizes known codes by code', () => {
    expect(errorMessageKey(apiError(409, 'idempotency_conflict'))).toBe('idempotency_conflict');
    expect(errorMessageKey(apiError(410, 'token_expired'))).toBe('token_expired');
  });

  it('falls back by status for unknown codes', () => {
    expect(errorMessageKey(apiError(0, 'whatever'))).toBe('network_error');
    expect(errorMessageKey(apiError(503, 'bad_gateway'))).toBe('internal_error');
    expect(errorMessageKey(apiError(400, 'bad_request'))).toBe('unknown');
    expect(errorMessageKey(new Error('x'))).toBe('unknown');
  });
});
