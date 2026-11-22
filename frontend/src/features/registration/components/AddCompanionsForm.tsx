'use client';

import { Button, Card, Group, Stack, Text, Title } from '@mantine/core';
import { useForm } from '@mantine/form';
import { IconPlus, IconTrash } from '@tabler/icons-react';
import { useTranslations } from 'next-intl';
import { useAddCompanions } from '@/features/registration/api/add-companions';
import { ApiErrorBanner } from '@/features/registration/components/ApiErrorBanner';
import { GuestFields } from '@/features/registration/components/GuestFields';
import type { ManageSummary } from '@/features/registration/types/api';
import {
  type AddCompanionsFormValues,
  addCompanionsSchema,
  emptyGuest,
  toFormErrors
} from '@/features/registration/types/schemas';
import { parseGuestLocation } from '@/features/registration/utils/api-errors';
import { toCompanionPayload } from '@/features/registration/utils/build-payload';
import { ApiError } from '@/lib/api-client';

type AddCompanionsFormProps = {
  summary: ManageSummary;
  onAdded: () => void;
  onError: (error: unknown) => boolean;
};

export function AddCompanionsForm({ summary, onAdded, onError }: AddCompanionsFormProps) {
  const t = useTranslations();
  const mutation = useAddCompanions();
  const form = useForm<AddCompanionsFormValues>({ mode: 'controlled', initialValues: { guests: [emptyGuest()] } });
  const guests = form.getValues().guests;

  const submit = form.onSubmit((values) => {
    const errors = toFormErrors(addCompanionsSchema.safeParse(values));
    if (Object.keys(errors).length > 0) {
      form.setErrors(Object.fromEntries(Object.entries(errors).map(([path, key]) => [path, t(`validation.${key}`)])));
      return;
    }
    mutation.mutate(values.guests.map(toCompanionPayload), {
      onSuccess: () => {
        form.reset();
        onAdded();
      },
      onError: (error) => {
        if (onError(error)) return;
        const mapped: Record<string, string> = {};
        for (const detail of error instanceof ApiError ? error.details : []) {
          const loc = parseGuestLocation(detail.location);
          if (loc) mapped[`guests.${loc.index}.${loc.field}`] = t('validation.fieldInvalid');
        }
        form.setErrors(mapped);
      }
    });
  });

  return (
    <form onSubmit={submit} noValidate>
      <Stack gap='md'>
        <Text size='sm'>{t('manage.addHelp', { remaining: summary.remaining_count })}</Text>
        {guests.map((guest, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: new guests have no id yet
          <Card key={index} padding='lg' component='section' aria-labelledby={`new-guest-${index}`}>
            <Stack gap='md'>
              <Group justify='space-between' wrap='nowrap'>
                <Title order={3} fz='lg' id={`new-guest-${index}`}>
                  {t('registration.companions.cardTitle', { index: summary.registered_count + index })}
                </Title>
                {guests.length > 1 ? (
                  <Button
                    variant='subtle'
                    color='gray'
                    leftSection={<IconTrash size={18} aria-hidden />}
                    onClick={() => form.removeListItem('guests', index)}
                  >
                    {t('registration.companions.remove', { index: summary.registered_count + index })}
                  </Button>
                ) : null}
              </Group>
              <GuestFields form={form} path={`guests.${index}`} guest={guest} />
            </Stack>
          </Card>
        ))}
        {guests.length < summary.remaining_count ? (
          <div>
            <Button
              variant='outline'
              leftSection={<IconPlus size={18} aria-hidden />}
              onClick={() => form.insertListItem('guests', emptyGuest())}
            >
              {t('registration.companions.add')}
            </Button>
          </div>
        ) : null}
        {mutation.isError ? <ApiErrorBanner error={mutation.error} /> : null}
        <div>
          <Button type='submit' loading={mutation.isPending}>
            {t('manage.addSubmit')}
          </Button>
        </div>
      </Stack>
    </form>
  );
}
