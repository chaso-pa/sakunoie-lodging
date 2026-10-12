'use client';

import { Button, type ButtonProps } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslations } from 'next-intl';
import { useLogout } from '@/features/auth/api/logout';

export function LogoutButton(props: ButtonProps) {
  const t = useTranslations('auth.logout');
  const tErrors = useTranslations('errors');
  const logout = useLogout();

  return (
    <Button
      variant='light'
      {...props}
      loading={logout.isPending}
      onClick={() =>
        logout.mutate(undefined, {
          onSuccess: () => notifications.show({ message: t('done') }),
          onError: () => notifications.show({ color: 'red', message: tErrors('generic') })
        })
      }
    >
      {t('button')}
    </Button>
  );
}
