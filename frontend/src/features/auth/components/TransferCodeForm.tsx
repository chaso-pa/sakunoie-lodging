'use client';

import { Button, TextInput, UnstyledButton } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useTranslations } from 'next-intl';
import { transferInputSchema, useTransfer } from '@/features/auth/api/transfer';
import classes from '@/features/auth/components/Auth.module.scss';
import { AuthCard } from '@/features/auth/components/AuthCard';
import { setLocale } from '@/i18n/set-locale';
import { getApiErrorStatus } from '@/lib/api-client';

// Shape hint only: X is a placeholder character, not part of any code.
const codeShape = 'XXXX-XXXX-XXXX';

export function TransferCodeForm({ onBackToInvite }: { onBackToInvite: () => void }) {
  const t = useTranslations('auth.restore');
  const tAuth = useTranslations('auth');
  const tErrors = useTranslations('errors');
  const transfer = useTransfer();

  const form = useForm({
    mode: 'uncontrolled',
    initialValues: { transfer_code: '' },
    validate: {
      transfer_code: (value) =>
        transferInputSchema.safeParse({ transfer_code: value }).success ? null : t('errors.format')
    }
  });

  const errorMessage = () => {
    if (!transfer.isError) {
      return '';
    }
    const status = getApiErrorStatus(transfer.error);
    if (status === 401 || status === 422) {
      return t('errors.invalid');
    }
    if (status === 429) {
      return tErrors('rateLimited');
    }
    return status === undefined ? tErrors('network') : tErrors('generic');
  };

  return (
    <AuthCard icon='🔑' eyebrow={t('eyebrow')} title={t('title')} description={t('description')}>
      <form
        style={{ marginTop: 16 }}
        onSubmit={form.onSubmit((values) =>
          transfer.mutate(values, {
            // Return to the language the member chose when registering.
            onSuccess: (member) => setLocale(member.preferred_language)
          })
        )}
      >
        <TextInput
          key={form.key('transfer_code')}
          {...form.getInputProps('transfer_code')}
          label={t('codeLabel')}
          placeholder={codeShape}
          autoComplete='off'
          autoCapitalize='characters'
          spellCheck={false}
          size='md'
          radius='sm'
          classNames={{ label: classes.fieldLabel, input: classes.codeInput }}
        />
        <p className={classes.error} role='alert'>
          {errorMessage()}
        </p>
        <Button type='submit' size='md' fullWidth mt={10} loading={transfer.isPending}>
          {t('submit')}
        </Button>
      </form>
      <div className={classes.note}>{t('note')}</div>
      <UnstyledButton className={classes.textLink} onClick={onBackToInvite}>
        {tAuth('backToInvite')}
      </UnstyledButton>
    </AuthCard>
  );
}
