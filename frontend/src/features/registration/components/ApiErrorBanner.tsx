import { Text } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { InfoBanner } from '@/components/InfoBanner';
import { errorMessageKey } from '@/features/registration/utils/api-errors';
import { ApiError } from '@/lib/api-client';

/** Localized message for an API failure, by error code, with the request id for support. */
export function ApiErrorBanner({ error, message }: { error: unknown; message?: string }) {
  const t = useTranslations('errors');
  const requestId = error instanceof ApiError ? error.requestId : null;
  return (
    <InfoBanner tone='error' announce='assertive'>
      <Text size='sm'>{message ?? t(errorMessageKey(error))}</Text>
      {requestId ? (
        <Text size='xs' c='dimmed' mt={4}>
          {t('requestId', { id: requestId })}
        </Text>
      ) : null}
    </InfoBanner>
  );
}
