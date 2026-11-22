import { invitationTokenStore } from '@/features/registration/stores/invitation-token-store';
import { registerAuthTokenProvider } from '@/lib/api-client';

// Imported for its side effect by the invitation check and the create call. Exact paths only: the
// invitation is never sent to `/guest-registrations/manage*` (the edit token's prefix) or anywhere else.
const provider = () => invitationTokenStore.get();
registerAuthTokenProvider('/invitations/current', provider, { exact: true });
registerAuthTokenProvider('/guest-registrations', provider, { exact: true });
