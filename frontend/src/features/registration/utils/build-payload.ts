import type { FacilityPublic, ManageSummary } from '@/features/registration/types/api';
import type {
  CompanionCorrectionValues,
  CompanionPayload,
  CreateRegistrationPayload,
  GuestFormValues,
  GuestPayload,
  ManageStayFormValues,
  WizardFormValues
} from '@/features/registration/types/schemas';
import { requiresPassport } from '@/features/registration/types/schemas';

/** Optional fields are omitted when empty: the API rejects `null` and treats absence as "not given". */
const optional = <K extends string>(key: K, value: string): Partial<Record<K, string>> => {
  const trimmed = value.trim();
  return trimmed === '' ? {} : ({ [key]: trimmed } as Record<K, string>);
};

export const toCompanionPayload = (guest: GuestFormValues): CompanionPayload => {
  const hasJapanAddress = guest.hasJapanAddress === true;
  return {
    full_name: guest.fullName.trim(),
    ...optional('name_kana', guest.nameKana),
    address: guest.address.trim(),
    ...optional('phone', guest.phone),
    ...optional('email', guest.email),
    has_japan_address: hasJapanAddress,
    // Nationality/passport are sent only when the legal rule requires them (data minimization).
    ...(requiresPassport(guest)
      ? { ...optional('nationality', guest.nationality), ...optional('passport_number', guest.passportNumber) }
      : {})
  };
};

/**
 * Wizard values → `POST /guest-registrations` body. The representative is always
 * `guests[0]`, companions follow in order; `utils/api-errors.ts` relies on this order
 * to map `body.guests[i]` back to form fields.
 */
export const buildCreatePayload = (values: WizardFormValues, facility: FacilityPublic): CreateRegistrationPayload => {
  const guests: GuestPayload[] = [
    { is_representative: true, ...toCompanionPayload(values.representative) },
    ...values.companions.map((c) => ({ is_representative: false, ...toCompanionPayload(c) }))
  ];
  return {
    facility_code: facility.code,
    check_in_date: values.stay.checkInDate,
    check_out_date: values.stay.checkOutDate,
    ...optional('estimated_arrival_time', values.stay.estimatedArrivalTime),
    planned_guest_count: Number(values.stay.plannedGuestCount),
    ...optional('booking_source', values.stay.bookingSource),
    ...optional('booking_reference', values.stay.bookingReference),
    terms_version: facility.terms_version,
    privacy_version: facility.privacy_version,
    terms_accepted: values.termsAccepted as true,
    privacy_accepted: values.privacyAccepted as true,
    guests
  };
};

export type PatchRegistrationBody = {
  estimated_arrival_time?: string;
  booking_source?: string;
  booking_reference?: string;
  planned_guest_count?: number;
};

export const manageStayValuesFrom = (summary: ManageSummary): ManageStayFormValues => ({
  estimatedArrivalTime: summary.estimated_arrival_time ?? '',
  bookingSource: summary.booking_source ?? '',
  bookingReference: summary.booking_reference ?? '',
  plannedGuestCount: summary.planned_guest_count
});

/** Only changed fields; an emptied optional field is sent as "" which the API treats as "clear". */
export const buildPatchRegistrationBody = (
  values: ManageStayFormValues,
  summary: ManageSummary
): PatchRegistrationBody => {
  const body: PatchRegistrationBody = {};
  const arrival = values.estimatedArrivalTime.trim();
  if (arrival !== (summary.estimated_arrival_time ?? '')) body.estimated_arrival_time = arrival;
  const source = values.bookingSource.trim();
  if (source !== (summary.booking_source ?? '')) body.booking_source = source;
  const reference = values.bookingReference.trim();
  if (reference !== (summary.booking_reference ?? '')) body.booking_reference = reference;
  const planned = Number(values.plannedGuestCount);
  if (planned !== summary.planned_guest_count) body.planned_guest_count = planned;
  return body;
};

export type PatchGuestBody = Partial<Record<'full_name' | 'name_kana' | 'address' | 'phone' | 'email', string>>;

/** Masked values cannot prefill the form, so an empty field means "leave unchanged". */
export const buildPatchGuestBody = (values: CompanionCorrectionValues): PatchGuestBody => ({
  ...optional('full_name', values.fullName),
  ...optional('name_kana', values.nameKana),
  ...optional('address', values.address),
  ...optional('phone', values.phone),
  ...optional('email', values.email)
});
