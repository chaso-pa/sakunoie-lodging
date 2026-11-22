import type { ButtonProps } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { ExternalButtonLink } from '@/components/ExternalButtonLink';
import { defaultFacility, type FacilityContent } from '@/config/site';

type Props = ButtonProps & { facility?: FacilityContent; label?: string };

/**
 * Primary booking CTA. Airbnb is the booking system; the URL comes from the facility content JSON
 * (`airbnbUrlIsPlaceholder: true` until the real listing URL replaces it, 要差し替え).
 */
export function AirbnbCta({ facility = defaultFacility, label, ...props }: Props) {
  const t = useTranslations('cta');
  return (
    <ExternalButtonLink
      href={facility.airbnbUrl}
      newTabLabel={t('opensInNewTab')}
      data-placeholder-url={facility.airbnbUrlIsPlaceholder ? 'true' : undefined}
      {...props}
    >
      {label ?? t('airbnb')}
    </ExternalButtonLink>
  );
}
