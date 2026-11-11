import { Group, Stack, Text, Title } from '@mantine/core';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';
import { PlaceholderBanner } from '@/features/public-site/components/PlaceholderBanner';

type Props = {
  title: string;
  intro?: ReactNode;
  placeholder: boolean;
  /** Shown for legal pages (terms/privacy keep consent-version references). */
  version?: { version: string; effectiveDate: string };
};

export function ContentPageHeader({ title, intro, placeholder, version }: Props) {
  const t = useTranslations('publicSite');
  return (
    <Stack gap='md'>
      <PlaceholderBanner show={placeholder} />
      <Title order={1} fz={{ base: 26, md: 34 }}>
        {title}
      </Title>
      {version ? (
        <Group gap='lg'>
          <Text size='sm' c='dimmed'>
            {t('version', { version: version.version })}
          </Text>
          <Text size='sm' c='dimmed'>
            {t('effectiveDate', { date: version.effectiveDate })}
          </Text>
        </Group>
      ) : null}
      {intro ? (
        <Text size='lg' c='dimmed'>
          {intro}
        </Text>
      ) : null}
    </Stack>
  );
}
