'use client';

import { Button, Card, Group, Stack, Text, Title } from '@mantine/core';
import { IconCopy, IconPlus, IconTrash } from '@tabler/icons-react';
import { useTranslations } from 'next-intl';
import { InfoBanner } from '@/components/InfoBanner';
import { GuestFields } from '@/features/registration/components/GuestFields';
import type { GuestFormValues } from '@/features/registration/types/schemas';
import { fieldId, type PathForm } from '@/features/registration/utils/form-paths';

type CompanionsStepProps = {
  form: PathForm;
  companions: GuestFormValues[];
  representative: GuestFormValues;
  /** Planned guest count minus the representative. */
  maxCompanions: number;
  onAdd: () => void;
  onRemove: (index: number) => void;
  companionsError?: string;
};

export function CompanionsStep({
  form,
  companions,
  representative,
  maxCompanions,
  onAdd,
  onRemove,
  companionsError
}: CompanionsStepProps) {
  const t = useTranslations('registration.companions');

  const copyFromRepresentative = (index: number) => {
    // Address and phone only; nationality and passport are never copied (contract §6).
    form.setFieldValue(`companions.${index}.address`, representative.address);
    form.setFieldValue(`companions.${index}.phone`, representative.phone);
  };

  return (
    <Stack gap='md'>
      <Text c='dimmed'>{t('description')}</Text>
      {companionsError ? (
        <div id={fieldId('companions')}>
          <InfoBanner tone='error' announce='assertive'>
            {companionsError}
          </InfoBanner>
        </div>
      ) : null}
      {companions.length === 0 ? <Text>{t('none')}</Text> : null}
      {companions.map((companion, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: companions have no stable id before submission
        <Card key={index} padding='lg' component='section' aria-labelledby={`companion-title-${index}`}>
          <Stack gap='md'>
            <Group justify='space-between' wrap='nowrap'>
              <Title order={3} fz='lg' id={`companion-title-${index}`}>
                {t('cardTitle', { index: index + 1 })}
              </Title>
              <Button
                variant='subtle'
                color='gray'
                leftSection={<IconTrash size={18} aria-hidden />}
                onClick={() => onRemove(index)}
              >
                {t('remove', { index: index + 1 })}
              </Button>
            </Group>
            <div>
              <Button
                variant='light'
                color='forest'
                size='sm'
                leftSection={<IconCopy size={16} aria-hidden />}
                onClick={() => copyFromRepresentative(index)}
              >
                {t('sameAsRepresentative')}
              </Button>
              <Text size='xs' c='dimmed' mt={4}>
                {t('copyNotice')}
              </Text>
            </div>
            <GuestFields form={form} path={`companions.${index}`} guest={companion} />
          </Stack>
        </Card>
      ))}
      {companions.length < maxCompanions ? (
        <div>
          <Button variant='outline' leftSection={<IconPlus size={18} aria-hidden />} onClick={onAdd}>
            {t('add')}
          </Button>
        </div>
      ) : (
        <Text size='sm' c='dimmed'>
          {t('limitReached')}
        </Text>
      )}
    </Stack>
  );
}
