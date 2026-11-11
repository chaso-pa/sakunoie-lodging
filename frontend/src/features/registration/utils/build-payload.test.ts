import { describe, expect, it } from 'vitest';
import type { FacilityPublic, ManageSummary } from '@/features/registration/types/api';
import {
  createRegistrationPayloadSchema,
  emptyGuest,
  emptyWizardValues,
  type WizardFormValues
} from '@/features/registration/types/schemas';
import {
  buildCreatePayload,
  buildPatchGuestBody,
  buildPatchRegistrationBody,
  manageStayValuesFrom
} from '@/features/registration/utils/build-payload';

const facility: FacilityPublic = {
  code: 'saku-no-ie',
  name: 'Test Facility',
  max_guests: 4,
  timezone: 'Asia/Tokyo',
  terms_version: '2026-11-01',
  privacy_version: '2026-11-02'
};

const values: WizardFormValues = {
  ...emptyWizardValues(),
  stay: {
    checkInDate: '2026-12-01',
    checkOutDate: '2026-12-03',
    estimatedArrivalTime: '',
    plannedGuestCount: '3',
    bookingSource: '',
    bookingReference: ' E2E-1 '
  },
  representative: {
    ...emptyGuest(),
    fullName: ' Test Rep ',
    address: 'Addr',
    hasJapanAddress: false,
    nationality: 'Testland',
    passportNumber: 'TESTPP1234'
  },
  companions: [
    {
      ...emptyGuest(),
      fullName: 'Test Companion',
      address: 'Addr',
      phone: '000-0000',
      hasJapanAddress: true,
      // Left over from toggling the answer: must not be sent.
      nationality: 'Testland',
      passportNumber: 'TESTPP9999'
    }
  ],
  termsAccepted: true,
  privacyAccepted: true
};

describe('buildCreatePayload', () => {
  const payload = buildCreatePayload(values, facility);

  it('puts the representative first and exactly once', () => {
    expect(payload.guests.map((g) => g.is_representative)).toEqual([true, false]);
    expect(createRegistrationPayloadSchema.safeParse(payload).success).toBe(true);
  });

  it('omits empty optional fields instead of sending null or ""', () => {
    expect(payload).not.toHaveProperty('estimated_arrival_time');
    expect(payload).not.toHaveProperty('booking_source');
    expect(payload.booking_reference).toBe('E2E-1');
    expect(payload.guests[0]).not.toHaveProperty('email');
    expect(JSON.stringify(payload)).not.toContain('null');
  });

  it('sends passport data only for guests without a Japan address', () => {
    expect(payload.guests[0]).toMatchObject({ nationality: 'Testland', passport_number: 'TESTPP1234' });
    expect(payload.guests[1]).not.toHaveProperty('passport_number');
    expect(payload.guests[1]).not.toHaveProperty('nationality');
  });

  it('takes versions from the facility and coerces the planned count', () => {
    expect(payload.terms_version).toBe('2026-11-01');
    expect(payload.privacy_version).toBe('2026-11-02');
    expect(payload.planned_guest_count).toBe(3);
    expect(payload.guests[0].full_name).toBe('Test Rep');
  });
});

const summary: ManageSummary = {
  public_id: 'p',
  facility: { code: 'saku-no-ie', name: 'Test Facility', timezone: 'Asia/Tokyo', max_guests: 4 },
  check_in_date: '2026-12-01',
  check_out_date: '2026-12-03',
  estimated_arrival_time: '16:00',
  planned_guest_count: 3,
  registered_count: 2,
  remaining_count: 1,
  is_complete: false,
  booking_source: 'airbnb',
  booking_reference: null,
  edit_expires_at: '2026-12-03T14:59:59.999999Z',
  version: '"rabc"',
  guests: []
};

describe('buildPatchRegistrationBody', () => {
  it('sends nothing when nothing changed', () => {
    expect(buildPatchRegistrationBody(manageStayValuesFrom(summary), summary)).toEqual({});
  });

  it('sends only changed fields, with "" to clear', () => {
    const body = buildPatchRegistrationBody(
      { ...manageStayValuesFrom(summary), estimatedArrivalTime: '', bookingReference: 'NEW', plannedGuestCount: 4 },
      summary
    );
    expect(body).toEqual({ estimated_arrival_time: '', booking_reference: 'NEW', planned_guest_count: 4 });
  });
});

describe('buildPatchGuestBody', () => {
  it('treats empty fields as unchanged', () => {
    expect(buildPatchGuestBody({ fullName: '', nameKana: '', address: '', phone: ' 090 ', email: '' })).toEqual({
      phone: '090'
    });
  });
});
