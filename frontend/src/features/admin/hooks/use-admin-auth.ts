'use client';

import { useQueryClient } from '@tanstack/react-query';
import { onIdTokenChanged, signInWithEmailAndPassword, signOut, type User } from 'firebase/auth';
import { useCallback, useEffect, useState } from 'react';
import { getFirebaseAuth } from '@/lib/firebase';

export type AdminAuthState =
  | { status: 'loading'; user: null }
  | { status: 'signedOut'; user: null }
  | { status: 'signedIn'; user: User };

/** Firebase sign-in state of the admin console (browser only). */
export const useAdminAuth = () => {
  const queryClient = useQueryClient();
  const [state, setState] = useState<AdminAuthState>({ status: 'loading', user: null });

  useEffect(() => {
    const auth = getFirebaseAuth();
    return onIdTokenChanged(auth, (user) =>
      setState(user ? { status: 'signedIn', user } : { status: 'signedOut', user: null })
    );
  }, []);

  const signIn = useCallback(
    (email: string, password: string) => signInWithEmailAndPassword(getFirebaseAuth(), email, password),
    []
  );

  const signOutAndClear = useCallback(async () => {
    await signOut(getFirebaseAuth());
    // Ledger data must not outlive the session in memory.
    queryClient.clear();
  }, [queryClient]);

  return { ...state, signIn, signOut: signOutAndClear };
};
