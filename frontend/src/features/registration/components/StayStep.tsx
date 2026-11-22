'use client';

import { InputWrapper, SegmentedControl, SimpleGrid, Stack, Text, TextInput } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { OptionalLabel } from '@/features/registration/components/OptionalLabel';
import type { FacilityPublic } from '@/features/registration/types/api';
import { MAX_GUESTS_PER_REQUEST } from '@/features/registration/types/schemas';
import { fieldId, type PathForm } from '@/features/registration/utils/form-paths';

export function StayStep({ form, facility }: { form: PathForm; facility: FacilityPublic }) {
  const t = useTranslations('registration.stay');
  const tInvitation = useTranslations('registration.invitation');
  const input = (path: string) => ({ id: fieldId(path), ...form.getInputProps(path) });
  const planned = form.getInputProps('stay.plannedGuestCount');
  const countOptions = Array.from({ length: Math.min(facility.max_guests, MAX_GUESTS_PER_REQUEST) }, (_, i) =>
    String(i + 1)
  );

  return (
    <Stack gap='md'>
      <div>
        <Text size='sm' fw={600}>
          {t('facility')}
        </Text>
        <Text fz='lg' fw={700}>
          {facility.name}
        </Text>
        <Text size='sm' c='dimmed'>
          {t('maxGuests', { count: facility.max_guests })}
        </Text>
      </div>
      {/* Set by the host in the invitation: shown, never editable (the server rejects other dates). */}
      <Stack gap={6}>
        <SimpleGrid cols={{ base: 1, xs: 2 }} spacing='md'>
          <TextInput
            type='date'
            label={t('checkIn')}
            readOnly
            aria-describedby='stay-dates-fixed'
            {...input('stay.checkInDate')}
          />
          <TextInput
            type='date'
            label={t('checkOut')}
            readOnly
            aria-describedby='stay-dates-fixed'
            {...input('stay.checkOutDate')}
          />
        </SimpleGrid>
        <Text id='stay-dates-fixed' size='sm' c='dimmed'>
          {tInvitation('datesFixed')}
        </Text>
      </Stack>
      <TextInput type='time' label={<OptionalLabel label={t('arrival')} />} {...input('stay.estimatedArrivalTime')} />
      <InputWrapper
        id={fieldId('stay.plannedGuestCount')}
        labelElement='div'
        label={t('plannedCount')}
        description={t('plannedCountHelp')}
        withAsterisk
        error={planned.error}
      >
        {/* One button per allowed count (1..facility max), so an out-of-range value cannot be entered. */}
        <SegmentedControl
          mt={6}
          size='md'
          fullWidth
          aria-labelledby={`${fieldId('stay.plannedGuestCount')}-label`}
          aria-describedby={`${fieldId('stay.plannedGuestCount')}-description`}
          data={countOptions}
          value={String(planned.value)}
          onChange={(value) => form.setFieldValue('stay.plannedGuestCount', Number(value))}
        />
      </InputWrapper>
    </Stack>
  );
}
