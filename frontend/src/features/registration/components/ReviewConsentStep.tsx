'use client';

import { Anchor, Badge, Button, Card, Checkbox, Group, Stack, Text, Title } from '@mantine/core';
import { useLocale, useTranslations } from 'next-intl';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import type { FacilityPublic } from '@/features/registration/types/api';
import type { GuestFormValues, WizardFormValues } from '@/features/registration/types/schemas';
import { fieldId, type PathForm } from '@/features/registration/utils/form-paths';
import { formatCivilDate } from '@/features/registration/utils/format';
import { Link } from '@/i18n/navigation';

type ReviewConsentStepProps = {
  form: PathForm;
  values: WizardFormValues;
  facility: FacilityPublic;
  onEditStep: (step: number) => void;
};

function Row({ label, value }: { label: string; value: string }) {
  return (
    <Group gap='xs' wrap='nowrap' align='flex-start'>
      <Text size='sm' c='dimmed' miw={120}>
        {label}
      </Text>
      <Text size='sm' style={{ overflowWrap: 'anywhere', whiteSpace: 'pre-line' }}>
        {value}
      </Text>
    </Group>
  );
}

export function PartialWarning({ registered, planned }: { registered: number; planned: number }) {
  const t = useTranslations('registration');
  if (registered >= planned) return null;
  return (
    <InfoBanner tone='warning' announce='polite'>
      {t('partialWarning', { registered, planned, remaining: planned - registered })}
    </InfoBanner>
  );
}

export function ReviewConsentStep({ form, values, facility, onEditStep }: ReviewConsentStepProps) {
  const t = useTranslations();
  const locale = useLocale();
  const notSet = t('common.notSet');
  const planned = Number(values.stay.plannedGuestCount);
  const guests: { guest: GuestFormValues; isRepresentative: boolean }[] = [
    { guest: values.representative, isRepresentative: true },
    ...values.companions.map((guest) => ({ guest, isRepresentative: false }))
  ];

  const guestRows = (g: GuestFormValues) => [
    { label: t('guest.fullName'), value: g.fullName },
    { label: t('guest.nameKana'), value: g.nameKana || notSet },
    { label: t('guest.address'), value: g.address },
    { label: t('guest.phone'), value: g.phone || notSet },
    { label: t('guest.email'), value: g.email || notSet },
    { label: t('guest.hasJapanAddress'), value: g.hasJapanAddress ? t('common.yes') : t('common.no') },
    ...(g.hasJapanAddress === false
      ? [
          { label: t('guest.nationality'), value: g.nationality },
          // Never echo the passport number back on screen.
          { label: t('guest.passportNumber'), value: t('registration.review.passportEntered') }
        ]
      : [])
  ];

  return (
    <Stack gap='lg'>
      <Text c='dimmed'>{t('registration.review.description')}</Text>
      <PartialWarning registered={guests.length} planned={planned} />

      <Card padding='lg' component='section'>
        <Group justify='space-between' mb='sm'>
          <Title order={3} fz='lg'>
            {t('registration.review.stayTitle')}
          </Title>
          <Button variant='subtle' size='sm' onClick={() => onEditStep(0)}>
            {t('registration.review.edit')}
          </Button>
        </Group>
        <Stack gap={6}>
          <Row label={t('registration.stay.facility')} value={facility.name} />
          <Row label={t('registration.stay.checkIn')} value={formatCivilDate(values.stay.checkInDate, locale)} />
          <Row label={t('registration.stay.checkOut')} value={formatCivilDate(values.stay.checkOutDate, locale)} />
          <Row label={t('registration.stay.arrival')} value={values.stay.estimatedArrivalTime || notSet} />
          <Row label={t('registration.stay.plannedCount')} value={String(planned)} />
          <Row label={t('registration.stay.bookingSource')} value={values.stay.bookingSource || notSet} />
          <Row label={t('registration.stay.bookingReference')} value={values.stay.bookingReference || notSet} />
        </Stack>
      </Card>

      <Card padding='lg' component='section'>
        <Title order={3} fz='lg' mb='sm'>
          {t('registration.review.guestsTitle')}
        </Title>
        <Stack gap='lg'>
          {guests.map(({ guest, isRepresentative }, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: guests have no stable id before submission
            <Stack key={index} gap={6}>
              <Group justify='space-between'>
                <Badge variant='outline' color={isRepresentative ? 'forest' : 'gray'} size='lg'>
                  {isRepresentative
                    ? t('registration.review.representativeBadge')
                    : `${t('registration.review.companionBadge')} ${index}`}
                </Badge>
                <Button variant='subtle' size='sm' onClick={() => onEditStep(isRepresentative ? 1 : 2)}>
                  {t('registration.review.edit')}
                </Button>
              </Group>
              {guestRows(guest).map((row) => (
                <Row key={row.label} label={row.label} value={row.value} />
              ))}
            </Stack>
          ))}
        </Stack>
      </Card>

      <Card padding='lg' component='section'>
        <Title order={3} fz='lg' mb='sm'>
          {t('registration.review.consentTitle')}
        </Title>
        <Stack gap='md'>
          <Group gap='md'>
            <Anchor component={Link} href={paths.public.terms.getHref()} target='_blank'>
              {t('registration.review.termsLink')}
            </Anchor>
            <Anchor component={Link} href={paths.public.privacy.getHref()} target='_blank'>
              {t('registration.review.privacyLink')}
            </Anchor>
          </Group>
          <Checkbox
            id={fieldId('termsAccepted')}
            size='md'
            label={t('registration.review.termsLabel', { version: facility.terms_version })}
            {...form.getInputProps('termsAccepted', { type: 'checkbox' })}
          />
          <Checkbox
            id={fieldId('privacyAccepted')}
            size='md'
            label={t('registration.review.privacyLabel', { version: facility.privacy_version })}
            {...form.getInputProps('privacyAccepted', { type: 'checkbox' })}
          />
        </Stack>
      </Card>
    </Stack>
  );
}
