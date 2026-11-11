import { ApiError } from '@/lib/api-client';

/** Error codes with a message in the `errors` namespace (see root CLAUDE.md §13). */
const KNOWN_CODES = [
  'validation_error',
  'facility_unavailable',
  'capacity_exceeded',
  'idempotency_conflict',
  'rate_limited',
  'token_invalid',
  'token_expired',
  'field_not_editable',
  'precondition_failed',
  'not_found',
  'network_error',
  'internal_error'
] as const;

export type ErrorMessageKey = (typeof KNOWN_CODES)[number] | 'unknown';

/** Localize by `code`, never by the server's English `message`. */
export const errorMessageKey = (error: unknown): ErrorMessageKey => {
  if (!(error instanceof ApiError)) return 'unknown';
  if ((KNOWN_CODES as readonly string[]).includes(error.code)) return error.code as ErrorMessageKey;
  if (error.isNetworkError) return 'network_error';
  if (error.status >= 500) return 'internal_error';
  return 'unknown';
};

const snakeToCamel = (s: string) => s.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase());

/** `body.guests[2].passport_number` → `{ index: 2, field: 'passportNumber' }`. */
export const parseGuestLocation = (location: string): { index: number; field: string } | null => {
  const m = /^body\.guests\[(\d+)\]\.([a-z_]+)$/.exec(location);
  return m ? { index: Number(m[1]), field: snakeToCamel(m[2]) } : null;
};

const STAY_FIELDS: Record<string, string> = {
  check_in_date: 'stay.checkInDate',
  check_out_date: 'stay.checkOutDate',
  estimated_arrival_time: 'stay.estimatedArrivalTime',
  planned_guest_count: 'stay.plannedGuestCount',
  booking_source: 'stay.bookingSource',
  booking_reference: 'stay.bookingReference',
  terms_accepted: 'termsAccepted',
  privacy_accepted: 'privacyAccepted'
};

/**
 * Maps an API error location of the create request to a wizard form path.
 * `guests[0]` is the representative (see `buildCreatePayload`).
 */
export const wizardPathFromLocation = (location: string): string | null => {
  const guest = parseGuestLocation(location);
  if (guest) {
    return guest.index === 0 ? `representative.${guest.field}` : `companions.${guest.index - 1}.${guest.field}`;
  }
  const top = /^body\.([a-z_]+)$/.exec(location);
  return top ? (STAY_FIELDS[top[1]] ?? null) : null;
};
