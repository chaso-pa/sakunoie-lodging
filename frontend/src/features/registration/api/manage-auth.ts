import { manageTokenStore } from '@/features/registration/stores/manage-token-store';
import { registerAuthTokenProvider } from '@/lib/api-client';

// Imported for its side effect by every manage API module.
registerAuthTokenProvider('/guest-registrations/manage', () => manageTokenStore.get());
