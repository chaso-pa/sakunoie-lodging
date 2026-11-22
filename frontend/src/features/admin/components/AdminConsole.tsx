'use client';

import { Button, Center, Loader, Stack, Text, Title } from '@mantine/core';
import { useRouter } from 'next/navigation';
import { type ReactNode, useEffect, useMemo } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import { useAdminFacilities } from '@/features/admin/api/get-facilities';
import { AdminShell } from '@/features/admin/components/AdminShell';
import { useAdminAuth } from '@/features/admin/hooks/use-admin-auth';
import { AdminSessionContext } from '@/features/admin/hooks/use-admin-session';
import { adminErrorMessage } from '@/features/admin/utils/error-message';
import { ApiError } from '@/lib/api-client';

function Loading() {
  return (
    <Center mih='100dvh'>
      <Loader color='forest' aria-label='読み込み中' />
    </Center>
  );
}

/**
 * Gate for every console page: signed out → login; signed in but not an admin (403
 * not_authorized) → "not authorized"; otherwise the shell with the authorized facilities. The
 * server enforces all of this again on every request.
 */
export function AdminConsole({ children }: { children: ReactNode }) {
  const router = useRouter();
  const auth = useAdminAuth();
  const uid = auth.status === 'signedIn' ? auth.user.uid : null;
  const facilities = useAdminFacilities(uid);

  useEffect(() => {
    if (auth.status === 'signedOut') router.replace(paths.admin.login.getHref());
  }, [auth.status, router]);

  const error = facilities.error;
  useEffect(() => {
    if (error instanceof ApiError && error.status === 401) void auth.signOut();
  }, [error, auth.signOut]);

  const session = useMemo(
    () =>
      auth.status === 'signedIn' && facilities.data
        ? { uid: auth.user.uid, email: auth.user.email, facilities: facilities.data }
        : null,
    [auth, facilities.data]
  );

  if (auth.status !== 'signedIn') return <Loading />;
  if (error) {
    const notAuthorized = error instanceof ApiError && error.status === 403;
    return (
      <Center mih='100dvh' p='md'>
        <Stack maw={480} gap='md'>
          <Title order={1} fz={24}>
            {notAuthorized ? '権限がありません' : '管理画面を表示できません'}
          </Title>
          <InfoBanner tone={notAuthorized ? 'warning' : 'error'} announce='assertive'>
            <Text>{adminErrorMessage(error)}</Text>
            {notAuthorized ? <Text mt='xs'>施設の管理者に、アカウントへの権限付与を依頼してください。</Text> : null}
          </InfoBanner>
          <Button variant='default' onClick={() => void auth.signOut()}>
            ログアウト
          </Button>
        </Stack>
      </Center>
    );
  }
  if (!session) return <Loading />;
  return (
    <AdminSessionContext.Provider value={session}>
      <AdminShell email={session.email} onSignOut={() => void auth.signOut()}>
        {children}
      </AdminShell>
    </AdminSessionContext.Provider>
  );
}
