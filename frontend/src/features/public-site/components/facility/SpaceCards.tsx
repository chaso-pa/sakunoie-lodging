import { Badge } from '@mantine/core';
import { IconKey, IconLock, IconUsers } from '@tabler/icons-react';
import Image from 'next/image';
import { useTranslations } from 'next-intl';
import type { FacilityPhoto } from '@/config/site';
import { FeatureIcon } from '@/features/public-site/components/FeatureCard';
import classes from '@/features/public-site/components/facility/SpaceCards.module.scss';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';
import type { Locale } from '@/i18n/routing';

type Spaces = PageFrontMatter<'facilities'>['spaces'];
type Props = {
  spaces: Spaces;
  /** Resolved from `spaces.private.photo` / `spaces.shared.photos` (facility photo ids). */
  privatePhoto?: FacilityPhoto;
  sharedPhotos: FacilityPhoto[];
  locale: Locale;
};

/**
 * "Yours" vs "shared with the host": two cards with photos, a text badge and a short list. Mobile shows
 * the first photo as a thumbnail beside the title; ≥ sm, the photos run across the top of the card.
 */
export function SpaceCards({ spaces, privatePhoto, sharedPhotos, locale }: Props) {
  const t = useTranslations('publicSite');
  const badge = t('placeholderPhoto');
  return (
    <div className={classes.wrap}>
      <p className={classes.intro}>{spaces.intro}</p>
      <div className={classes.cards}>
        <article className={classes.card} aria-labelledby='space-private-title'>
          <div className={classes.photos} data-count={privatePhoto ? 1 : 0}>
            {privatePhoto ? <SpacePhoto photo={privatePhoto} locale={locale} badge={badge} /> : null}
          </div>
          <div className={classes.body}>
            <div className={classes.head}>
              <Badge
                className={classes.badgePrivate}
                leftSection={<IconKey size={14} aria-hidden />}
                radius='sm'
                size='lg'
              >
                {t('badgePrivate')}
              </Badge>
              <h3 id='space-private-title' className={classes.title}>
                {spaces.private.title}
              </h3>
              <p className={classes.text}>{spaces.private.body}</p>
            </div>
            <SpaceList items={spaces.private.items} />
          </div>
        </article>

        <article className={classes.card} aria-labelledby='space-shared-title'>
          <div className={classes.photos} data-count={sharedPhotos.length}>
            {sharedPhotos.map((p) => (
              <SpacePhoto key={p.id} photo={p} locale={locale} badge={badge} />
            ))}
          </div>
          <div className={classes.body}>
            <div className={classes.head}>
              <Badge
                className={classes.badgeShared}
                leftSection={<IconUsers size={14} aria-hidden />}
                radius='sm'
                size='lg'
              >
                {t('badgeShared')}
              </Badge>
              <h3 id='space-shared-title' className={classes.title}>
                {spaces.shared.title}
              </h3>
              <p className={classes.text}>{spaces.shared.body}</p>
            </div>
            <SpaceList items={spaces.shared.items} />
          </div>
        </article>
      </div>
      <p className={classes.offLimits}>
        <IconLock size={18} aria-hidden stroke={1.8} />
        {spaces.offLimits}
      </p>
    </div>
  );
}

function SpaceList({ items }: { items: Spaces['private']['items'] }) {
  return (
    <ul className={classes.list}>
      {items.map((item) => (
        <li key={item.text} className={classes.listItem}>
          <span className={classes.listIcon}>
            <FeatureIcon name={item.icon} size={20} />
          </span>
          {item.text}
        </li>
      ))}
    </ul>
  );
}

function SpacePhoto({ photo, locale, badge }: { photo: FacilityPhoto; locale: Locale; badge: string }) {
  return (
    <div className={classes.photo}>
      <Image
        src={photo.src}
        alt={photo.alt[locale]}
        fill
        sizes='(min-width: 62em) 360px, (min-width: 48em) 50vw, 96px'
        unoptimized
        style={{ objectFit: 'cover', objectPosition: photo.focalPoint }}
      />
      {photo.placeholder ? (
        <Badge pos='absolute' top={8} left={8} color='dark' variant='filled' radius='sm' tt='none'>
          {badge}
        </Badge>
      ) : null}
    </div>
  );
}
