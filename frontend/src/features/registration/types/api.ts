/** Response shapes from `backend/api/openapi.yaml` (consumed as-is; snake_case like the wire format). */

export type FacilityPublic = {
  code: string;
  name: string;
  max_guests: number;
  timezone: string;
  terms_version: string;
  privacy_version: string;
};

export type CreateRegistrationResult = {
  public_id: string;
  facility_code: string;
  check_in_date: string;
  check_out_date: string;
  planned_guest_count: number;
  registered_count: number;
  remaining_count: number;
  is_complete: boolean;
  /** Null on a replay that cannot mint another link; then `code` is `edit_link_unavailable`. */
  edit_token: string | null;
  edit_token_expires_at: string | null;
  code?: 'edit_link_unavailable';
};

export type ManageGuest = {
  public_id: string;
  is_representative: boolean;
  full_name_masked: string;
  name_kana_masked: string | null;
  phone_masked: string | null;
  email_masked: string | null;
  has_japan_address: boolean;
  passport_last4: string | null;
  editable: boolean;
};

export type ManageSummary = {
  public_id: string;
  facility: { code: string; name: string; timezone: string; max_guests: number };
  check_in_date: string;
  check_out_date: string;
  estimated_arrival_time: string | null;
  planned_guest_count: number;
  registered_count: number;
  remaining_count: number;
  is_complete: boolean;
  booking_source: string | null;
  booking_reference: string | null;
  edit_expires_at: string;
  /** Equals the ETag; sent back as If-Match. */
  version: string;
  guests: ManageGuest[];
};
