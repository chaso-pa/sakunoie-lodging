'use client';

import { Button, Card, Container, CopyButton, Group, Stack, Text, TextInput, Title } from '@mantine/core';
import { IconCheck, IconCopy } from '@tabler/icons-react';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import { RegistrationStatus } from '@/features/registration/components/RegistrationStatus';
import { useWizardStore } from '@/features/registration/stores/wizard-store';
import { formatInstant } from '@/features/registration/utils/format';
import { Link } from '@/i18n/navigation';

export function CompletionView() {
  const t = useTranslations('complete');
  const locale = useLocale();
  const result = useWizardStore((s) => s.result);
  const acknowledgeCompletion = useWizardStore((s) => s.acknowledgeCompletion);
  const [origin, setOrigin] = useState<string | null>(null);

  useEffect(() => {
    setOrigin(window.location.origin);
    // The completion is on screen now, so the idempotency key has done its job.
    if (result) acknowledgeCompletion();
  }, [result, acknowledgeCompletion]);

  if (!result || !result.registration.edit_token) {
    return (
      <Container size={760} py={{ base: 24, md: 48 }}>
        <Stack gap='lg'>
          <Title order={1} fz={{ base: 26, md: 32 }}>
            {t('noResultTitle')}
          </Title>
          <InfoBanner tone='info'>
            <Text size='sm'>{t('noResultBody')}</Text>
          </InfoBanner>
          <div>
            <Button component={Link} href={paths.public.home.getHref()} variant='outline'>
              {t('backHome')}
            </Button>
          </div>
        </Stack>
      </Container>
    );
  }

  const { registration, facility } = result;
  const token = registration.edit_token as string;
  const manageUrl = origin ? paths.registration.manage.getAbsoluteHref(origin, locale, token) : '';
  const expiresAt = registration.edit_token_expires_at
    ? formatInstant(registration.edit_token_expires_at, locale, facility.timezone)
    : '';

  return (
    <Container size={760} py={{ base: 24, md: 48 }}>
      <Stack gap='lg'>
        <Stack gap='xs'>
          <Title order={1} fz={{ base: 26, md: 32 }}>
            {t('title')}
          </Title>
          <Text>{t('lead')}</Text>
        </Stack>

        <Card padding='lg'>
          <Stack gap='sm'>
            <Text fw={700}>{facility.name}</Text>
            <RegistrationStatus
              registered={registration.registered_count}
              planned={registration.planned_guest_count}
              remaining={registration.remaining_count}
              isComplete={registration.is_complete}
            />
          </Stack>
        </Card>

        <Card padding='lg' component='section' aria-labelledby='edit-link-title'>
          <Stack gap='md'>
            <Title order={2} fz='xl' id='edit-link-title'>
              {t('editLinkTitle')}
            </Title>
            <Text size='sm'>{t('editLinkBody', { expiresAt })}</Text>
            <TextInput
              label={t('editLinkLabel')}
              value={manageUrl}
              readOnly
              onFocus={(e) => e.currentTarget.select()}
            />
            <InfoBanner tone='warning'>
              <Text size='sm'>{t('keepSafe')}</Text>
            </InfoBanner>
            <Group gap='sm'>
              <CopyButton value={manageUrl}>
                {({ copied, copy }) => (
                  <Button
                    variant='outline'
                    onClick={copy}
                    disabled={!manageUrl}
                    leftSection={copied ? <IconCheck size={18} aria-hidden /> : <IconCopy size={18} aria-hidden />}
                  >
                    <span aria-live='polite'>{copied ? t('copied') : t('copy')}</span>
                  </Button>
                )}
              </CopyButton>
              {/* Full navigation: the manage page reads the token from the fragment. */}
              <Button component='a' href={manageUrl || undefined}>
                {registration.is_complete ? t('openManage') : t('addCompanions')}
              </Button>
            </Group>
          </Stack>
        </Card>
      </Stack>
    </Container>
  );
}
