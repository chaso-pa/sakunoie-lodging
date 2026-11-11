import { randomUUID } from 'node:crypto';
import { type APIRequestContext, expect } from '@playwright/test';
import { FACILITY_CODE, POLICY_VERSION, SYNTHETIC_PASSPORT, uniqueStay } from './synthetic';

type Guest = {
  is_representative: boolean;
  full_name: string;
  address: string;
  has_japan_address: boolean;
  nationality?: string;
  passport_number?: string;
  phone?: string;
  email?: string;
};

export type SeedOverrides = {
  bookingReference: string;
  checkIn?: string;
  checkOut?: string;
  plannedGuestCount?: number;
  guests?: Guest[];
};

export type SeededRegistration = {
  publicId: string;
  editToken: string;
  bookingReference: string;
  checkIn: string;
  checkOut: string;
};

/** Representative without a Japanese address (passport required) plus one companion. */
export const defaultGuests = (): Guest[] => [
  {
    is_representative: true,
    full_name: 'Test Rep',
    address: '1 Test Street, Testland',
    has_japan_address: false,
    nationality: 'Testland',
    passport_number: SYNTHETIC_PASSPORT,
    phone: '000-0000-1234',
    email: 'test-rep@example.test'
  },
  {
    is_representative: false,
    full_name: 'Test Companion',
    address: '1 Test Street, Testland',
    has_japan_address: true
  }
];

/** Creates a registration through the public API (via the Next rewrite) and returns its one-time edit token. */
export const seedRegistration = async (
  request: APIRequestContext,
  overrides: SeedOverrides
): Promise<SeededRegistration> => {
  const stay =
    overrides.checkIn && overrides.checkOut
      ? { checkIn: overrides.checkIn, checkOut: overrides.checkOut }
      : uniqueStay();
  const response = await request.post('/api/v1/guest-registrations', {
    headers: { 'Idempotency-Key': `e2e-${randomUUID()}` },
    data: {
      facility_code: FACILITY_CODE,
      check_in_date: stay.checkIn,
      check_out_date: stay.checkOut,
      planned_guest_count: overrides.plannedGuestCount ?? 3,
      booking_source: 'airbnb',
      booking_reference: overrides.bookingReference,
      terms_version: POLICY_VERSION,
      privacy_version: POLICY_VERSION,
      terms_accepted: true,
      privacy_accepted: true,
      guests: overrides.guests ?? defaultGuests()
    }
  });
  expect(response.status(), await response.text()).toBe(201);
  const body = (await response.json()) as { public_id: string; edit_token: string };
  expect(body.edit_token).toBeTruthy();
  return {
    publicId: body.public_id,
    editToken: body.edit_token,
    bookingReference: overrides.bookingReference,
    checkIn: stay.checkIn,
    checkOut: stay.checkOut
  };
};

/** Submits one anonymous survey response through the public API. */
export const seedSurvey = async (request: APIRequestContext, q01 = 4) => {
  const response = await request.post('/api/v1/surveys', {
    data: { facility_code: FACILITY_CODE, survey_version: 'v1', answers: { q01 }, website: '' }
  });
  expect(response.status(), await response.text()).toBe(202);
};
