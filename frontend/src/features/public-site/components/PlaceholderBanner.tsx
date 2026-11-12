import { Text } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { InfoBanner } from '@/components/InfoBanner';

/** Visible marker while a page's front matter says `placeholder: true` (要差し替え). */
export function PlaceholderBanner({ show = true }: { show?: boolean }) {
  const t = useTranslations('publicSite');
  if (!show) return null;
  return (
    <InfoBanner tone='warning' title={t('placeholderTitle')}>
      <Text size='sm'>{t('placeholderBody')}</Text>
    </InfoBanner>
  );
}
