import { Text } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { InfoBanner } from '@/components/InfoBanner';

/**
 * Visible marker while a page's front matter says `placeholder: true` (要差し替え). `body` can say
 * precisely what is still pending instead of the generic text.
 */
export function PlaceholderBanner({ show = true, body }: { show?: boolean; body?: string }) {
  const t = useTranslations('publicSite');
  if (!show) return null;
  return (
    <InfoBanner tone='warning' title={t('placeholderTitle')}>
      <Text size='sm'>{body ?? t('placeholderBody')}</Text>
    </InfoBanner>
  );
}
