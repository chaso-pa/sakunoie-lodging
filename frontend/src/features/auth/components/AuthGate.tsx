'use client';

import { Button, Loader } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';
import type { SignupResponse } from '@/features/auth/api/signup';
import { setSignedInMember, useMe } from '@/features/auth/api/use-me';
import classes from '@/features/auth/components/Auth.module.scss';
import { InviteCodeForm } from '@/features/auth/components/InviteCodeForm';
import { SignupForm } from '@/features/auth/components/SignupForm';
import { TransferCodeDisplay } from '@/features/auth/components/TransferCodeDisplay';
import { TransferCodeForm } from '@/features/auth/components/TransferCodeForm';
import { WelcomePage } from '@/features/auth/components/WelcomePage';

type Step = 'welcome' | 'invite' | 'signup' | 'restore';

/**
 * Gate for the community area. Members see `children`; everyone else gets the invite flow in place,
 * at whichever community URL they opened. This only decides what to draw: the API enforces access
 * (401 on every member-only endpoint), and the invite code is checked by the server.
 */
export function AuthGate({ children }: { children: React.ReactNode }) {
  const t = useTranslations();
  const queryClient = useQueryClient();
  const me = useMe();
  const [step, setStep] = useState<Step>('welcome');
  // Result of a signup whose transfer code the member has not yet confirmed saving.
  const [issued, setIssued] = useState<SignupResponse | null>(null);

  const isSignedIn = Boolean(me.data);
  useEffect(() => {
    // After a logout, go straight back to the invite form rather than replaying the landing.
    if (isSignedIn) {
      setStep('invite');
    }
  }, [isSignedIn]);

  if (issued) {
    return (
      <TransferCodeDisplay
        code={issued.transfer_code}
        onContinue={() => {
          setSignedInMember(queryClient, issued.member);
          setIssued(null);
        }}
      />
    );
  }

  if (me.isPending) {
    return (
      <div className={classes.center}>
        <Loader aria-label={t('common.loading')} />
      </div>
    );
  }

  if (me.isError) {
    return (
      <div className={classes.center}>
        <div>
          <p className={classes.error} role='alert'>
            {t('errors.network')}
          </p>
          <Button variant='light' fullWidth mt={10} loading={me.isRefetching} onClick={() => me.refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      </div>
    );
  }

  if (me.data) {
    return children;
  }

  switch (step) {
    case 'welcome':
      return <WelcomePage onStart={() => setStep('invite')} />;
    case 'signup':
      return <SignupForm onSignedUp={setIssued} onBackToInvite={() => setStep('invite')} />;
    case 'restore':
      return <TransferCodeForm onBackToInvite={() => setStep('invite')} />;
    default:
      return <InviteCodeForm onVerified={() => setStep('signup')} onRestore={() => setStep('restore')} />;
  }
}
