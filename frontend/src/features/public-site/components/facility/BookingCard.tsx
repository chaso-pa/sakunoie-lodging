import { Text } from '@mantine/core';
import { IconUsers } from '@tabler/icons-react';
import { useTranslations } from 'next-intl';
import { AirbnbCta } from '@/components/AirbnbCta';
import { paths } from '@/config/paths';
import { ArrowLink } from '@/features/public-site/components/facility/ArrowLink';
import classes from '@/features/public-site/components/facility/BookingCard.module.scss';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Props = { booking: PageFrontMatter<'facilities'>['booking'] };

/** Price summary and the Airbnb CTA (`#booking`). Airbnb decides the actual price. */
export function BookingCard({ booking }: Props) {
  const t = useTranslations('publicSite');
  return (
    <section id='booking' aria-labelledby='booking-title' className={classes.card}>
      <h2 id='booking-title' className={classes.title}>
        {t('priceTitle')}
      </h2>
      <p className={classes.priceBlock}>
        <span className={classes.priceLabel}>{booking.priceLabel}</span>
        <span className={classes.price}>{booking.price}</span>
      </p>
      <p className={classes.detail}>
        <IconUsers size={18} aria-hidden stroke={1.6} />
        {booking.priceDetail}
      </p>
      <AirbnbCta size='lg' fullWidth />
      <Text size='sm' c='dimmed' lh={1.6}>
        {booking.note}
      </Text>
      <ArrowLink href={paths.public.rates.getHref()}>{t('ratesLink')}</ArrowLink>
    </section>
  );
}

/** Mobile-only reminder at the end of the page (the desktop booking card is sticky instead). */
export function BookingReminder() {
  const t = useTranslations('publicSite');
  return (
    <aside aria-labelledby='booking-reminder-title' className={classes.reminder}>
      <h2 id='booking-reminder-title' className={classes.reminderTitle}>
        {t('bookingTitle')}
      </h2>
      <Text c='dimmed'>{t('bookingBody')}</Text>
      <AirbnbCta size='lg' fullWidth />
      <ArrowLink href={paths.public.rates.getHref()}>{t('ratesLink')}</ArrowLink>
    </aside>
  );
}
