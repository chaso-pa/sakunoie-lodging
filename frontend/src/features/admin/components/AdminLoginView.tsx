'use client';

import { Button, Center, Paper, PasswordInput, Stack, Text, TextInput, Title } from '@mantine/core';
import { useRouter } from 'next/navigation';
import { type FormEvent, useEffect, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import { useAdminAuth } from '@/features/admin/hooks/use-admin-auth';
import { signInErrorMessage } from '@/features/admin/utils/error-message';

/** Email/password sign-in (Firebase Auth; the Auth Emulator in dev/E2E). */
export function AdminLoginView() {
  const router = useRouter();
  const auth = useAdminAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (auth.status === 'signedIn') router.replace(paths.admin.registrations.getHref());
  }, [auth.status, router]);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!email.trim() || !password) {
      setError('メールアドレスとパスワードを入力してください。');
      return;
    }
    setSubmitting(true);
    try {
      await auth.signIn(email.trim(), password);
    } catch (err) {
      setError(signInErrorMessage(err));
      setSubmitting(false);
    }
  };

  return (
    <Center mih='100dvh' p='md' bg='var(--sakunoie-canvas)'>
      <Paper component='main' id='main' withBorder radius='lg' p='xl' w='100%' maw={420}>
        <form onSubmit={onSubmit} noValidate>
          <Stack gap='md'>
            <div>
              <Text c='var(--sakunoie-forest)' fw={700}>
                佐久の家
              </Text>
              <Title order={1} fz={26}>
                管理画面ログイン
              </Title>
            </div>
            {error ? (
              <InfoBanner tone='error' announce='assertive'>
                {error}
              </InfoBanner>
            ) : null}
            <TextInput
              label='メールアドレス'
              type='email'
              autoComplete='username'
              value={email}
              onChange={(e) => setEmail(e.currentTarget.value)}
              required
            />
            <PasswordInput
              label='パスワード'
              autoComplete='current-password'
              value={password}
              onChange={(e) => setPassword(e.currentTarget.value)}
              required
            />
            <Button type='submit' color='forest' loading={submitting || auth.status === 'signedIn'}>
              ログイン
            </Button>
          </Stack>
        </form>
      </Paper>
    </Center>
  );
}
