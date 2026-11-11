/** Admin API shapes (backend/api/openapi.yaml, tag Admin). */

export type AdminFacility = {
  id: number;
  code: string;
  name: string;
  timezone: string;
  max_guests: number;
  is_active: boolean;
};

export type AdminRegistrationRow = {
  public_id: string;
  facility: { id: number; code: string; name: string };
  check_in_date: string;
  check_out_date: string;
  booking_source: string | null;
  booking_reference: string | null;
  representative_name: string;
  planned_guest_count: number;
  registered_count: number;
  remaining_count: number;
  is_complete: boolean;
  identity_verified: boolean;
  identity_verified_at: string | null;
  submitted_at: string;
};

export type AdminLedgerMetrics = { registrations: number; guests: number; incomplete: number; verified: number };

export type AdminRegistrationList = {
  items: AdminRegistrationRow[];
  page: number;
  page_size: number;
  total: number;
  metrics: AdminLedgerMetrics;
};

export type AdminGuest = {
  public_id: string;
  is_representative: boolean;
  full_name: string;
  name_kana: string | null;
  address: string;
  phone: string | null;
  email: string | null;
  has_japan_address: boolean;
  nationality: string | null;
  passport_last4: string | null;
};

export type AdminRegistrationDetail = {
  public_id: string;
  facility: AdminFacility;
  check_in_date: string;
  check_out_date: string;
  estimated_arrival_time: string | null;
  planned_guest_count: number;
  capacity_snapshot: number;
  registered_count: number;
  remaining_count: number;
  is_complete: boolean;
  booking_source: string | null;
  booking_reference: string | null;
  terms_version: string;
  privacy_version: string;
  submitted_at: string;
  identity_verified: boolean;
  identity_verified_at: string | null;
  active_edit_links: number;
  edit_link_expires_at: string | null;
  revealed: boolean;
  version: string;
  guests: AdminGuest[];
};

export type AdminPatchGuest = {
  public_id: string;
  is_representative?: true;
  full_name?: string;
  name_kana?: string;
  address?: string;
  phone?: string;
  email?: string;
  has_japan_address?: boolean;
  nationality?: string;
  passport_number?: string;
};

export type AdminPatchRegistration = {
  check_in_date?: string;
  check_out_date?: string;
  estimated_arrival_time?: string;
  planned_guest_count?: number;
  capacity_override?: boolean;
  booking_source?: string;
  booking_reference?: string;
  guests?: AdminPatchGuest[];
};

export type ReissueEditLinkResult = { edit_token: string; edit_token_expires_at: string; expired: boolean };

export type LedgerFilter = { facilityId?: number; from?: string; to?: string; q?: string; page?: number };
