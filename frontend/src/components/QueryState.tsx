'use client';

import { Button, Loader } from '@mantine/core';
import { useTranslations } from 'next-intl';
import classes from '@/components/Community.module.scss';
import { getApiErrorStatus } from '@/lib/api-client';

export function LoadingState() {
  const t = useTranslations('common');
  return (
    <output className={classes.loading}>
      <Loader aria-label={t('loading')} />
    </output>
  );
}

type ErrorStateProps = {
  error?: unknown;
  /** Overrides the message chosen from the error. */
  message?: string;
  onRetry?: () => void;
  retrying?: boolean;
};

/** Failure of one data view. The rest of the screen keeps working; this block offers the retry. */
export function ErrorState({ error, message, onRetry, retrying }: ErrorStateProps) {
  const t = useTranslations();
  const status = getApiErrorStatus(error);
  const text =
    message ??
    (status === undefined ? t('errors.network') : status === 429 ? t('errors.rateLimited') : t('errors.generic'));

  return (
    <div className={classes.errorBox} role='alert'>
      {text}
      {onRetry && (
        <div>
          <Button variant='light' size='xs' mt={10} loading={retrying} onClick={onRetry}>
            {t('common.retry')}
          </Button>
        </div>
      )}
    </div>
  );
}

export function EmptyState({ children }: { children: React.ReactNode }) {
  return <div className={classes.empty}>{children}</div>;
}
