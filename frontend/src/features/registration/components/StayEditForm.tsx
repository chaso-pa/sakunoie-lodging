'use client';

import { Button, Group, NumberInput, Stack, Text, TextInput } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { usePatchRegistration } from '@/features/registration/api/patch-registration';
import { ApiErrorBanner } from '@/features/registration/components/ApiErrorBanner';
import { OptionalLabel } from '@/features/registration/components/OptionalLabel';
import type { ManageSummary } from '@/features/registration/types/api';
import { type ManageStayFormValues, manageStaySchema, toFormErrors } from '@/features/registration/types/schemas';
import { buildPatchRegistrationBody, manageStayValuesFrom } from '@/features/registration/utils/build-payload';

type StayEditFormProps = {
  summary: ManageSummary;
  onDone: () => void;
  onCancel: () => void;
  /** Returns true if the error was handled by the page (token or precondition failure). */
  onError: (error: unknown) => boolean;
};

export function StayEditForm({ summary, onDone, onCancel, onError }: StayEditFormProps) {
  const t = useTranslations();
  const mutation = usePatchRegistration();
  const [nothingChanged, setNothingChanged] = useState(false);
  const form = useForm<ManageStayFormValues>({ mode: 'controlled', initialValues: manageStayValuesFrom(summary) });

  const submit = form.onSubmit((values) => {
    const errors = toFormErrors(
      manageStaySchema(summary.registered_count, summary.facility.max_guests).safeParse(values)
    );
    if (Object.keys(errors).length > 0) {
      form.setErrors(
        Object.fromEntries(
          Object.entries(errors).map(([path, key]) => [
            path,
            t(`validation.${key}`, { max: summary.facility.max_guests })
          ])
        )
      );
      return;
    }
    const body = buildPatchRegistrationBody(values, summary);
    if (Object.keys(body).length === 0) {
      setNothingChanged(true);
      return;
    }
    setNothingChanged(false);
    mutation.mutate({ body, version: summary.version }, { onSuccess: onDone, onError: (error) => void onError(error) });
  });

  return (
    <form onSubmit={submit} noValidate>
      <Stack gap='md'>
        <Text size='sm' c='dimmed'>
          {t('manage.editStayHelp')}
        </Text>
        <TextInput
          type='time'
          label={<OptionalLabel label={t('registration.stay.arrival')} />}
          {...form.getInputProps('estimatedArrivalTime')}
        />
        <NumberInput
          label={t('registration.stay.plannedCount')}
          min={Math.max(1, summary.registered_count)}
          max={summary.facility.max_guests}
          allowDecimal={false}
          allowNegative={false}
          {...form.getInputProps('plannedGuestCount')}
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
