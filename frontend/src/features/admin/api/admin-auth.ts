import { registerAuthTokenProvider } from '@/lib/api-client';
import { getFirebaseAuth } from '@/lib/firebase';

// Imported for its side effect by every admin API module: `/admin/*` requests carry the current
// Firebase ID token (refreshed by the SDK when it is about to expire).
registerAuthTokenProvider('/admin/', async () => {
  const user = getFirebaseAuth().currentUser;
  return user ? user.getIdToken() : null;
});

export const adminQueryKey = ['admin'] as const;
