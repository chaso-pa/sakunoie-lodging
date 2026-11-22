import { type APIRequestContext, expect } from '@playwright/test';
import { adminCredentials } from './admin-login';
import { API_URL } from './api-context';
import { FACILITY_CODE, uniqueStay } from './synthetic';

// Invitations are issued through the admin API as the emulator owner that global-setup.ts created and
// granted, exactly as the host does in the console (root CLAUDE.md §13, 2026-11-15).
const EMULATOR = `http://${process.env.FIREBASE_AUTH_EMULATOR_HOST ?? '127.0.0.1:9099'}`;

let idToken: { value: string; at: number } | null = null;

/** A Firebase ID token of the E2E owner from the Auth Emulator (cached; ID tokens live 1 h). */
const adminIdToken = async (): Promise<string> => {
  if (idToken && Date.now() - idToken.at < 30 * 60_000) return idToken.value;
  const { email, password } = adminCredentials();
  const res = await fetch(
    `${EMULATOR}/identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=demo-api-key`,
    {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ email, password, returnSecureToken: true })
    }
  );
  if (!res.ok) throw new Error(`emulator sign-in failed: ${res.status}`);
  idToken = { value: ((await res.json()) as { idToken: string }).idToken, at: Date.now() };
  return idToken.value;
};

export const adminHeaders = async () => ({ Authorization: `Bearer ${await adminIdToken()}` });

let facilityId: number | null = null;

const sakuFacilityId = async (request: APIRequestContext): Promise<number> => {
  if (facilityId !== null) return facilityId;
  const res = await request.get(`${API_URL}/api/v1/admin/facilities`, { headers: await adminHeaders() });
  expect(res.status(), await res.text()).toBe(200);
  const { facilities } = (await res.json()) as { facilities: { id: number; code: string }[] };
  const saku = facilities.find((f) => f.code === FACILITY_CODE);
  if (!saku) throw new Error('the E2E owner has no saku-no-ie grant');
  facilityId = saku.id;
  return facilityId;
};

export type IssuedInvitation = { token: string; publicId: string; checkIn: string; checkOut: string };

/** Issues an invitation for a (by default unique far-future) stay. Works with any request context (absolute URLs). */
export const issueInvitation = async (
  request: APIRequestContext,
  options: { checkIn?: string; checkOut?: string; bookingReference?: string; note?: string } = {}
): Promise<IssuedInvitation> => {
  const stay =
    options.checkIn && options.checkOut ? { checkIn: options.checkIn, checkOut: options.checkOut } : uniqueStay();
  const res = await request.post(`${API_URL}/api/v1/admin/invitations`, {
    headers: await adminHeaders(),
    data: {
      facility_id: await sakuFacilityId(request),
      check_in_date: stay.checkIn,
      check_out_date: stay.checkOut,
      ...(options.bookingReference ? { booking_reference: options.bookingReference } : {}),
      ...(options.note ? { note: options.note } : {})
    }
  });
  expect(res.status(), await res.text()).toBe(201);
  const body = (await res.json()) as { invitation_token: string; invitation: { public_id: string } };
  return { token: body.invitation_token, publicId: body.invitation.public_id, ...stay };
};

export const revokeInvitation = async (request: APIRequestContext, publicId: string) => {
  const res = await request.post(`${API_URL}/api/v1/admin/invitations/${publicId}/revoke`, {
    headers: await adminHeaders()
  });
  expect(res.status(), await res.text()).toBe(200);
};

/** Locale-prefixed wizard path with the invitation in the fragment (never in a query string). */
export const invitationPath = (locale: string, token: string) =>
  `/${locale}/stays/${FACILITY_CODE}/registration#invite=${encodeURIComponent(token)}`;
