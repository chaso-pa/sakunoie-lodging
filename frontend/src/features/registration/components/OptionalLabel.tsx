import { Text } from '@mantine/core';
import { useTranslations } from 'next-intl';

export function OptionalLabel({ label }: { label: string }) {
  const t = useTranslations('common');
  return (
    <>
      {label}{' '}
      <Text span size='sm' c='dimmed' fw={400}>
        ({t('optional')})
      </Text>
    </>
  );
}
