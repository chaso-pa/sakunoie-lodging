'use client';

import { Stack, Text } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { GuestFields } from '@/features/registration/components/GuestFields';
import type { GuestFormValues } from '@/features/registration/types/schemas';
import type { PathForm } from '@/features/registration/utils/form-paths';

export function RepresentativeStep({ form, representative }: { form: PathForm; representative: GuestFormValues }) {
  const t = useTranslations('registration.representative');
  return (
    <Stack gap='md'>
      <Text c='dimmed'>{t('description')}</Text>
      <GuestFields form={form} path='representative' guest={representative} />
    </Stack>
  );
}
