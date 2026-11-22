import { createFragmentTokenStore } from '@/features/registration/stores/fragment-token-store';

/**
 * Registration invitation from the host (root CLAUDE.md §13, 2026-11-15): `#invite=` →
 * sessionStorage['sakunoie.registration.invite'] (this tab only). Sent only to
 * `GET /invitations/current` and `POST /guest-registrations`; cleared after a successful create.
 */
const invitation = createFragmentTokenStore('sakunoie.registration.invite', 'invite');

export const invitationTokenStore = invitation.store;
export const parseInviteFragment = invitation.parse;
export const consumeInviteFragment = invitation.consume;
