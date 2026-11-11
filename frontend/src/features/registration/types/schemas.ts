import { z } from 'zod';

// No `new Function` JIT (or its feature probe): the nonce CSP has no 'unsafe-eval' in production.
z.config({ jitless: true });

/**
 * Client-side validation (UX only; the Go API is the authority). Limits mirror the
 * DTOs in `backend/api/openapi.yaml`. Issue messages are keys of the `validation`
 * message namespace, translated by the components.
 */
export type ValidationKey =
  | 'required'
  | 'tooLong'
  | 'invalidDate'
  | 'checkOutAfterCheckIn'
  | 'invalidTime'
  | 'guestCountRange'
  | 'tooManyGuests'
  | 'email'
  | 'passportFormat'
  | 'consentRequired'
  | 'representativeExactlyOne';

/** API hard limit on guests per request (`maxItems: 20`). */
export const MAX_GUESTS_PER_REQUEST = 20;

const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const TIME_PATTERN = /^([01][0-9]|2[0-3]):[0-5][0-9]$/;
const PASSPORT_PATTERN = /^[A-Za-z0-9 ]{6,20}$/;
const FACILITY_CODE_PATTERN = /^[a-z0-9-]{1,64}$/;

const msg = (key: ValidationKey) => ({ error: key });

const requiredText = (max: number) => z.string().trim().min(1, msg('required')).max(max, msg('tooLong'));
const optionalText = (max: number) => z.string().trim().max(max, msg('tooLong'));
const optionalEmail = optionalText(255).refine((v) => v === '' || z.email().safeParse(v).success, msg('email'));
/** A real calendar date: `Date` rolls `2026-02-30` over to March, so the value must round-trip. */
const isValidDate = (v: string) => {
  if (!DATE_PATTERN.test(v)) return false;
  const d = new Date(`${v}T00:00:00Z`);
  return !Number.isNaN(d.getTime()) && d.toISOString().slice(0, 10) === v;
};

/**
 * The single place for the conditional legal rule (contract §3.3/§11 VERIFY):
 * a guest without an address in Japan must give nationality and passport number.
 */
export const requiresPassport = (guest: { hasJapanAddress: boolean | null }) => guest.hasJapanAddress === false;

export const guestFormSchema = z
  .object({
    fullName: requiredText(255),
    nameKana: optionalText(255),
    address: requiredText(2000),
    phone: optionalText(50),
    email: optionalEmail,
    hasJapanAddress: z.boolean().nullable(),
    nationality: optionalText(100),
    passportNumber: z
      .string()
      .trim()
      .refine((v) => v === '' || PASSPORT_PATTERN.test(v), msg('passportFormat'))
  })
  .superRefine((guest, ctx) => {
    if (guest.hasJapanAddress === null) {
      ctx.addIssue({ code: 'custom', path: ['hasJapanAddress'], message: 'required' satisfies ValidationKey });
    }
    if (requiresPassport(guest)) {
      if (guest.nationality === '')
        ctx.addIssue({ code: 'custom', path: ['nationality'], message: 'required' satisfies ValidationKey });
      if (guest.passportNumber === '')
        ctx.addIssue({ code: 'custom', path: ['passportNumber'], message: 'required' satisfies ValidationKey });
    }
  });

export type GuestFormValues = z.input<typeof guestFormSchema>;

export const emptyGuest = (): GuestFormValues => ({
  fullName: '',
  nameKana: '',
  address: '',
  phone: '',
  email: '',
  hasJapanAddress: null,
  nationality: '',
  passportNumber: ''
});

const plannedCount = (min: number, max: number) =>
  z.coerce
    .number(msg('guestCountRange'))
    .int(msg('guestCountRange'))
    .min(min, msg('guestCountRange'))
    .max(max, msg('guestCountRange'));

export const stayFormSchema = (maxGuests: number) =>
  z
    .object({
      checkInDate: z.string().refine(isValidDate, msg('invalidDate')),
      checkOutDate: z.string().refine(isValidDate, msg('invalidDate')),
      estimatedArrivalTime: z.string().refine((v) => v === '' || TIME_PATTERN.test(v), msg('invalidTime')),
      plannedGuestCount: plannedCount(1, Math.min(maxGuests, MAX_GUESTS_PER_REQUEST)),
      bookingSource: optionalText(50),
      bookingReference: optionalText(255)
    })
    .superRefine((stay, ctx) => {
      // ISO dates compare lexicographically.
      if (isValidDate(stay.checkInDate) && isValidDate(stay.checkOutDate) && stay.checkOutDate <= stay.checkInDate) {
        ctx.addIssue({
          code: 'custom',
          path: ['checkOutDate'],
          message: 'checkOutAfterCheckIn' satisfies ValidationKey
        });
      }
    });

export type StayFormValues = Omit<z.input<ReturnType<typeof stayFormSchema>>, 'plannedGuestCount'> & {
  /** Mantine NumberInput yields '' while empty. */
  plannedGuestCount: number | string;
};

export const wizardFormSchema = (maxGuests: number) =>
  z
    .object({
      stay: stayFormSchema(maxGuests),
      representative: guestFormSchema,
      companions: z.array(guestFormSchema),
      termsAccepted: z.boolean(),
      privacyAccepted: z.boolean()
    })
    .superRefine((values, ctx) => {
      const planned = Number(values.stay.plannedGuestCount);
      if (Number.isInteger(planned) && values.companions.length + 1 > planned) {
        ctx.addIssue({ code: 'custom', path: ['companions'], message: 'tooManyGuests' satisfies ValidationKey });
      }
      if (!values.termsAccepted)
        ctx.addIssue({ code: 'custom', path: ['termsAccepted'], message: 'consentRequired' satisfies ValidationKey });
      if (!values.privacyAccepted) {
        ctx.addIssue({ code: 'custom', path: ['privacyAccepted'], message: 'consentRequired' satisfies ValidationKey });
      }
    });

export type WizardFormValues = {
  stay: StayFormValues;
  representative: GuestFormValues;
  companions: GuestFormValues[];
  termsAccepted: boolean;
  privacyAccepted: boolean;
};

export const emptyWizardValues = (): WizardFormValues => ({
  stay: {
    checkInDate: '',
    checkOutDate: '',
    estimatedArrivalTime: '',
    plannedGuestCount: 1,
    bookingSource: '',
    bookingReference: ''
  },
  representative: emptyGuest(),
  companions: [],
  termsAccepted: false,
  privacyAccepted: false
});

/** Form error map: Mantine path (`companions.0.fullName`) → validation key. First issue per path wins. */
export type FormErrorMap = Record<string, ValidationKey>;

export const toFormErrors = (result: { success: boolean; error?: z.ZodError }): FormErrorMap => {
  const errors: FormErrorMap = {};
  if (result.success || !result.error) return errors;
  for (const issue of result.error.issues) {
    const path = issue.path.join('.');
    if (!(path in errors)) errors[path] = issue.message as ValidationKey;
  }
  return errors;
};

export const validateWizard = (values: WizardFormValues, maxGuests: number) =>
  toFormErrors(wizardFormSchema(maxGuests).safeParse(values));

export const WIZARD_STEPS = ['stay', 'representative', 'companions', 'review'] as const;

/** Which wizard step owns a form error path. */
export const stepOfPath = (path: string): number => {
  if (path.startsWith('stay.')) return 0;
  if (path.startsWith('representative.')) return 1;
  if (path === 'companions' || path.startsWith('companions.')) return 2;
  return 3;
};

export const errorsForStep = (errors: FormErrorMap, step: number): FormErrorMap =>
  Object.fromEntries(Object.entries(errors).filter(([path]) => stepOfPath(path) === step));

// --- API payloads ---------------------------------------------------------------------------

const payloadGuestSchema = z.object({
  is_representative: z.boolean(),
  full_name: z.string().min(1).max(255),
  name_kana: z.string().max(255).optional(),
  address: z.string().min(1).max(2000),
  phone: z.string().max(50).optional(),
  email: z.string().max(255).optional(),
  has_japan_address: z.boolean(),
  nationality: z.string().min(1).max(100).optional(),
  passport_number: z.string().regex(PASSPORT_PATTERN).optional()
});

export type GuestPayload = z.infer<typeof payloadGuestSchema>;
export type CompanionPayload = Omit<GuestPayload, 'is_representative'>;

/** Mirrors `CreateGuestRegistrationBody`, plus the cross-field rules the server enforces. */
export const createRegistrationPayloadSchema = z
  .object({
    facility_code: z.string().regex(FACILITY_CODE_PATTERN),
    check_in_date: z.string().regex(DATE_PATTERN),
    check_out_date: z.string().regex(DATE_PATTERN),
    estimated_arrival_time: z.string().regex(TIME_PATTERN).optional(),
    planned_guest_count: z.number().int().min(1).max(50),
    booking_source: z.string().max(50).optional(),
    booking_reference: z.string().max(255).optional(),
    terms_version: z.string().min(1).max(64),
    privacy_version: z.string().min(1).max(64),
    terms_accepted: z.literal(true),
    privacy_accepted: z.literal(true),
    guests: z.array(payloadGuestSchema).min(1).max(MAX_GUESTS_PER_REQUEST)
  })
  .superRefine((payload, ctx) => {
    if (payload.guests.filter((g) => g.is_representative).length !== 1) {
      ctx.addIssue({ code: 'custom', path: ['guests'], message: 'representativeExactlyOne' satisfies ValidationKey });
    }
    if (payload.guests.length > payload.planned_guest_count) {
      ctx.addIssue({ code: 'custom', path: ['guests'], message: 'tooManyGuests' satisfies ValidationKey });
    }
    payload.guests.forEach((g, i) => {
      if (!g.has_japan_address && (!g.nationality || !g.passport_number)) {
        ctx.addIssue({
          code: 'custom',
          path: ['guests', i, 'passport_number'],
          message: 'required' satisfies ValidationKey
        });
      }
    });
  });

export type CreateRegistrationPayload = z.infer<typeof createRegistrationPayloadSchema>;

// --- Manage forms ---------------------------------------------------------------------------

export const manageStaySchema = (registeredCount: number, maxGuests: number) =>
  z.object({
    estimatedArrivalTime: z.string().refine((v) => v === '' || TIME_PATTERN.test(v), msg('invalidTime')),
    bookingSource: optionalText(50),
    bookingReference: optionalText(255),
    plannedGuestCount: plannedCount(Math.max(1, registeredCount), maxGuests)
  });

export type ManageStayFormValues = {
  estimatedArrivalTime: string;
  bookingSource: string;
  bookingReference: string;
  plannedGuestCount: number | string;
};

/** Companion correction: every field optional; an empty field means "leave unchanged". */
export const companionCorrectionSchema = z.object({
  fullName: optionalText(255),
  nameKana: optionalText(255),
  address: optionalText(2000),
  phone: optionalText(50),
  email: optionalEmail
});

export type CompanionCorrectionValues = z.input<typeof companionCorrectionSchema>;

export const addCompanionsSchema = z.object({ guests: z.array(guestFormSchema).min(1) });
export type AddCompanionsFormValues = { guests: GuestFormValues[] };
