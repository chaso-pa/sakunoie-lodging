'use client';

import { Button, Checkbox, CopyButton } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import classes from '@/features/auth/components/Auth.module.scss';
import { AuthCard } from '@/features/auth/components/AuthCard';
import { formatTransferCode } from '@/features/auth/utils/format-transfer-code';

type TransferCodeDisplayProps = {
  code: string;
  onContinue: () => void;
};

/**
 * Shown once, right after signup. The API never returns the code again, so the member has to confirm
 * that it is saved before moving on. The code is kept in memory only.
 */
export function TransferCodeDisplay({ code, onContinue }: TransferCodeDisplayProps) {
  const t = useTranslations('auth.transferCode');
  const [saved, setSaved] = useState(false);
  const formatted = formatTransferCode(code);

  return (
    <AuthCard icon='🔑' eyebrow={t('eyebrow')} title={t('title')} description={t('description')}>
      <div className={classes.fieldLabel} style={{ marginTop: 16 }}>
        {t('codeLabel')}
      </div>
      <output className={classes.transferCode} style={{ display: 'block' }}>
        {formatted}
      </output>
      <CopyButton value={formatted}>
        {({ copied, copy }) => (
          <Button variant='light' size='md' fullWidth onClick={copy}>
            {copied ? t('copied') : t('copy')}
          </Button>
        )}
      </CopyButton>
      <p className={classes.warning} role='note'>
        {t('warning')}
      </p>
      <Checkbox
        label={t('confirm')}
        checked={saved}
        onChange={(event) => setSaved(event.currentTarget.checked)}
        size='sm'
      />
      <Button size='md' fullWidth mt={14} disabled={!saved} onClick={onContinue}>
        {t('continue')}
      </Button>
    </AuthCard>
  );
}
