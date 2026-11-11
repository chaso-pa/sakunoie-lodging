import { describe, expect, it } from 'vitest';
import {
  createRegistrationPayloadSchema,
  emptyGuest,
  emptyWizardValues,
  errorsForStep,
  type GuestFormValues,
  guestFormSchema,
  manageStaySchema,
  requiresPassport,
  stepOfPath,
  toFormErrors,
  validateWizard,
  type WizardFormValues
} from '@/features/registration/types/schemas';

const japanGuest = (overrides: Partial<GuestFormValues> = {}): GuestFormValues => ({
  ...emptyGuest(),
  fullName: 'Test Rep',
  address: '1-1 Test-cho, Saku',
  hasJapanAddress: true,
  ...overrides
});

const validWizard = (overrides: Partial<WizardFormValues> = {}): WizardFormValues => ({
  ...emptyWizardValues(),
  stay: {
    checkInDate: '2026-12-01',
    checkOutDate: '2026-12-03',
    estimatedArrivalTime: '16:00',
    plannedGuestCount: 3,
    bookingSource: 'airbnb',
    bookingReference: ''
  },
  representative: japanGuest(),
  companions: [japanGuest({ fullName: 'Test Companion' })],
  termsAccepted: true,
  privacyAccepted: true,
  ...overrides
});

const guestErrors = (guest: GuestFormValues) => toFormErrors(guestFormSchema.safeParse(guest));

describe('guest schema: conditional passport rule', () => {
  it('requires nationality and passport when there is no Japan address', () => {
    const guest = japanGuest({ hasJapanAddress: false });
    expect(requiresPassport(guest)).toBe(true);
    expect(guestErrors(guest)).toEqual({ nationality: 'required', passportNumber: 'required' });
  });

  it('accepts nationality and a well-formed passport', () => {
    expect(
      guestErrors(japanGuest({ hasJapanAddress: false, nationality: 'Testland', passportNumber: 'TESTPP1234' }))
    ).toEqual({});
  });

  it('rejects a malformed passport number', () => {
    const errors = guestErrors(
      japanGuest({ hasJapanAddress: false, nationality: 'Testland', passportNumber: 'AB-12' })
    );
    expect(errors).toEqual({ passportNumber: 'passportFormat' });
  });

  it('does not require passport fields with a Japan address', () => {
    expect(requiresPassport(japanGuest())).toBe(false);
    expect(guestErrors(japanGuest())).toEqual({});
  });

  it('requires an explicit answer to the Japan-address question', () => {
    expect(guestErrors(japanGuest({ hasJapanAddress: null }))).toEqual({ hasJapanAddress: 'required' });
  });

  it('reports required, length and email problems together', () => {
    const errors = guestErrors(japanGuest({ fullName: '  ', address: 'x'.repeat(2001), email: 'not-an-email' }));
    expect(errors).toEqual({ fullName: 'required', address: 'tooLong', email: 'email' });
  });
});

describe('wizard schema: counts and dates', () => {
  it('accepts a valid partial registration (2 of 3)', () => {
    expect(validateWizard(validWizard(), 4)).toEqual({});
  });

  it('bounds the planned count by the facility max', () => {
    const tooMany = validWizard({ stay: { ...validWizard().stay, plannedGuestCount: 5 } });
    expect(validateWizard(tooMany, 4)).toMatchObject({ 'stay.plannedGuestCount': 'guestCountRange' });
    const zero = validWizard({ stay: { ...validWizard().stay, plannedGuestCount: 0 }, companions: [] });
    expect(validateWizard(zero, 4)).toMatchObject({ 'stay.plannedGuestCount': 'guestCountRange' });
    const empty = validWizard({ stay: { ...validWizard().stay, plannedGuestCount: '' }, companions: [] });
    expect(validateWizard(empty, 4)).toMatchObject({ 'stay.plannedGuestCount': 'guestCountRange' });
  });

  it('rejects more guests than planned', () => {
    const values = validWizard({
      stay: { ...validWizard().stay, plannedGuestCount: 1 }
    });
    expect(validateWizard(values, 4)).toMatchObject({ companions: 'tooManyGuests' });
  });

  it('requires check-out after check-in', () => {
    const same = validWizard({ stay: { ...validWizard().stay, checkOutDate: '2026-12-01' } });
    expect(validateWizard(same, 4)).toMatchObject({ 'stay.checkOutDate': 'checkOutAfterCheckIn' });
    const missing = validWizard({ stay: { ...validWizard().stay, checkInDate: '' } });
    expect(validateWizard(missing, 4)).toMatchObject({ 'stay.checkInDate': 'invalidDate' });
  });

  it('requires both consents', () => {
    expect(validateWizard(validWizard({ termsAccepted: false, privacyAccepted: false }), 4)).toEqual({
      termsAccepted: 'consentRequired',
      privacyAccepted: 'consentRequired'
    });
  });

  it('assigns every error path to its step', () => {
    expect(stepOfPath('stay.checkInDate')).toBe(0);
    expect(stepOfPath('representative.passportNumber')).toBe(1);
    expect(stepOfPath('companions')).toBe(2);
    expect(stepOfPath('companions.0.fullName')).toBe(2);
    expect(stepOfPath('termsAccepted')).toBe(3);
    const errors = validateWizard(
      validWizard({ representative: japanGuest({ fullName: '' }), termsAccepted: false }),
      4
    );
    expect(errorsForStep(errors, 1)).toEqual({ 'representative.fullName': 'required' });
    expect(errorsForStep(errors, 0)).toEqual({});
  });
});

describe('create payload schema: representative exactly one', () => {
  const payload = {
    facility_code: 'saku-no-ie',
    check_in_date: '2026-12-01',
    check_out_date: '2026-12-03',
    planned_guest_count: 2,
    terms_version: '2026-11-01',
    privacy_version: '2026-11-01',
    terms_accepted: true as const,
    privacy_accepted: true as const,
    guests: [
      { is_representative: true, full_name: 'Test Rep', address: 'Addr', has_japan_address: true },
      { is_representative: false, full_name: 'Test Companion', address: 'Addr', has_japan_address: true }
    ]
  };
  const issues = (p: unknown) => {
    const r = createRegistrationPayloadSchema.safeParse(p);
    return r.success ? [] : r.error.issues.map((i) => i.message);
  };

  it('accepts exactly one representative', () => {
    expect(issues(payload)).toEqual([]);
  });

  it('rejects zero or two representatives', () => {
    expect(issues({ ...payload, guests: payload.guests.map((g) => ({ ...g, is_representative: false })) })).toContain(
      'representativeExactlyOne'
    );
    expect(issues({ ...payload, guests: payload.guests.map((g) => ({ ...g, is_representative: true })) })).toContain(
      'representativeExactlyOne'
    );
  });

  it('rejects more guests than planned, and zero guests', () => {
    expect(issues({ ...payload, planned_guest_count: 1 })).toContain('tooManyGuests');
    expect(issues({ ...payload, guests: [] }).length).toBeGreaterThan(0);
  });

  it('requires passport data for guests without a Japan address', () => {
    const guests = [{ ...payload.guests[0], has_japan_address: false }];
    expect(issues({ ...payload, planned_guest_count: 1, guests })).toContain('required');
  });
});

describe('manage stay schema', () => {
  it('keeps the planned count between the registered count and the facility max', () => {
    const schema = manageStaySchema(3, 4);
    const base = { estimatedArrivalTime: '', bookingSource: '', bookingReference: '' };
    expect(toFormErrors(schema.safeParse({ ...base, plannedGuestCount: 2 }))).toEqual({
      plannedGuestCount: 'guestCountRange'
    });
    expect(toFormErrors(schema.safeParse({ ...base, plannedGuestCount: 5 }))).toEqual({
      plannedGuestCount: 'guestCountRange'
    });
    expect(toFormErrors(schema.safeParse({ ...base, plannedGuestCount: 4 }))).toEqual({});
  });
});
