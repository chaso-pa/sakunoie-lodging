'use client';

import { NativeSelect, NumberInput, SimpleGrid, Stack, Text, TextInput } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { OptionalLabel } from '@/features/registration/components/OptionalLabel';
import type { FacilityPublic } from '@/features/registration/types/api';
import { MAX_GUESTS_PER_REQUEST } from '@/features/registration/types/schemas';
import { fieldId, type PathForm } from '@/features/registration/utils/form-paths';

export function StayStep({ form, facility }: { form: PathForm; facility: FacilityPublic }) {
  const t = useTranslations('registration.stay');
  const input = (path: string) => ({ id: fieldId(path), ...form.getInputProps(path) });

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
      <SimpleGrid cols={{ base: 1, xs: 2 }} spacing='md'>
        <TextInput type='date' label={t('checkIn')} withAsterisk {...input('stay.checkInDate')} />
        <TextInput type='date' label={t('checkOut')} withAsterisk {...input('stay.checkOutDate')} />
      </SimpleGrid>
      <TextInput type='time' label={<OptionalLabel label={t('arrival')} />} {...input('stay.estimatedArrivalTime')} />
      <NumberInput
        label={t('plannedCount')}
        description={t('plannedCountHelp')}
        withAsterisk
        min={1}
        max={Math.min(facility.max_guests, MAX_GUESTS_PER_REQUEST)}
        allowDecimal={false}
        allowNegative={false}
        {...input('stay.plannedGuestCount')}
      />
      <NativeSelect
        label={<OptionalLabel label={t('bookingSource')} />}
        data={[
          { value: '', label: t('bookingSourceNone') },
          { value: 'airbnb', label: t('bookingSourceAirbnb') },
          { value: 'other', label: t('bookingSourceOther') }
        ]}
        {...input('stay.bookingSource')}
      />
      <TextInput
        label={<OptionalLabel label={t('bookingReference')} />}
        autoComplete='off'
        {...input('stay.bookingReference')}
      />
    </Stack>
  );
}
