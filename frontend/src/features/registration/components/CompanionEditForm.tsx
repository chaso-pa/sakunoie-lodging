'use client';

import { Button, Group, Stack, Text, Textarea, TextInput } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { usePatchGuest } from '@/features/registration/api/patch-guest';
import { ApiErrorBanner } from '@/features/registration/components/ApiErrorBanner';
import type { ManageGuest, ManageSummary } from '@/features/registration/types/api';
import {
  type CompanionCorrectionValues,
  companionCorrectionSchema,
  toFormErrors
} from '@/features/registration/types/schemas';
import { buildPatchGuestBody } from '@/features/registration/utils/build-payload';

type CompanionEditFormProps = {
  guest: ManageGuest;
  summary: ManageSummary;
  onDone: () => void;
  onCancel: () => void;
  onError: (error: unknown) => boolean;
};

/** Allowlisted companion corrections only (name, kana, address, phone, email). */
export function CompanionEditForm({ guest, summary, onDone, onCancel, onError }: CompanionEditFormProps) {
  const t = useTranslations();
  const mutation = usePatchGuest();
  const [nothingChanged, setNothingChanged] = useState(false);
  const form = useForm<CompanionCorrectionValues>({
    mode: 'controlled',
    initialValues: { fullName: '', nameKana: '', address: '', phone: '', email: '' }
  });

  const submit = form.onSubmit((values) => {
    const errors = toFormErrors(companionCorrectionSchema.safeParse(values));
    if (Object.keys(errors).length > 0) {
      form.setErrors(Object.fromEntries(Object.entries(errors).map(([path, key]) => [path, t(`validation.${key}`)])));
      return;
    }
    const body = buildPatchGuestBody(values);
    if (Object.keys(body).length === 0) {
      setNothingChanged(true);
      return;
    }
    setNothingChanged(false);
    mutation.mutate(
      { guestPublicId: guest.public_id, body, version: summary.version },
      { onSuccess: onDone, onError: (error) => void onError(error) }
    );
  });

  return (
    <form onSubmit={submit} noValidate>
      <Stack gap='md'>
        <Text size='sm' c='dimmed'>
          {t('manage.editGuestHelp')}
        </Text>
        <TextInput
          label={t('guest.fullName')}
          placeholder={guest.full_name_masked}
          {...form.getInputProps('fullName')}
        />
        <TextInput
          label={t('guest.nameKana')}
          placeholder={guest.name_kana_masked ?? ''}
          {...form.getInputProps('nameKana')}
        />
        <Textarea label={t('guest.address')} autosize minRows={2} {...form.getInputProps('address')} />
        <TextInput
          label={t('guest.phone')}
          type='tel'
          placeholder={guest.phone_masked ?? ''}
          {...form.getInputProps('phone')}
        />
        <TextInput
          label={t('guest.email')}
          type='email'
          placeholder={guest.email_masked ?? ''}
          {...form.getInputProps('email')}
        />
        {nothingChanged ? <InfoBanner tone='info'>{t('manage.nothingToSave')}</InfoBanner> : null}
        {mutation.isError ? <ApiErrorBanner error={mutation.error} /> : null}
        <Group gap='sm'>
          <Button type='submit' loading={mutation.isPending}>
            {t('manage.save')}
          </Button>
          <Button variant='default' onClick={onCancel} disabled={mutation.isPending}>
            {t('manage.cancel')}
          </Button>
        </Group>
      </Stack>
    </form>
  );
}
