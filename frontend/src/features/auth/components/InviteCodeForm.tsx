'use client';

import { Button, TextInput, UnstyledButton } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useTranslations } from 'next-intl';
import { useVerifyInvite } from '@/features/auth/api/verify-invite';
import classes from '@/features/auth/components/Auth.module.scss';
import { AuthCard } from '@/features/auth/components/AuthCard';
import { getApiErrorStatus } from '@/lib/api-client';

type InviteCodeFormProps = {
  onVerified: () => void;
  onRestore: () => void;
};

export function InviteCodeForm({ onVerified, onRestore }: InviteCodeFormProps) {
  const t = useTranslations('auth.invite');
  const tErrors = useTranslations('errors');
  const verifyInvite = useVerifyInvite();

  const form = useForm({
    mode: 'uncontrolled',
    initialValues: { code: '' },
    validate: { code: (value) => (value.trim() ? null : t('errors.required')) }
  });

  const errorMessage = () => {
    if (!verifyInvite.isError) {
      return '';
    }
    const status = getApiErrorStatus(verifyInvite.error);
    if (status === 401 || status === 422) {
      return t('errors.invalid');
    }
    if (status === 429) {
      return tErrors('rateLimited');
    }
    return status === undefined ? tErrors('network') : tErrors('generic');
  };

  return (
    <AuthCard icon='🏡' eyebrow={t('eyebrow')} title={t('title')} description={t('description')} step={1}>
      <form
        onSubmit={form.onSubmit(({ code }) =>
          verifyInvite.mutate({ code: code.trim().toUpperCase() }, { onSuccess: onVerified })
        )}
      >
        <TextInput
          key={form.key('code')}
          {...form.getInputProps('code')}
          label={t('codeLabel')}
          placeholder={t('codePlaceholder')}
          autoComplete='one-time-code'
          autoCapitalize='characters'
          spellCheck={false}
          size='md'
          radius='sm'
          classNames={{ label: classes.fieldLabel, input: classes.codeInput }}
        />
        <p className={classes.error} role='alert'>
          {errorMessage()}
        </p>
        <Button type='submit' size='md' fullWidth mt={10} loading={verifyInvite.isPending}>
          {t('submit')}
        </Button>
      </form>
      <div className={classes.note}>
        <strong>{t('noteTitle')}</strong>
        {t('noteBody')}
      </div>
      <UnstyledButton className={classes.textLink} onClick={onRestore}>
        {t('restoreLink')}
      </UnstyledButton>
    </AuthCard>
  );
}
