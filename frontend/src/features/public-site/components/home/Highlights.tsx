import Image from 'next/image';
import type { FacilityPhoto } from '@/config/site';
import { FeatureIcon } from '@/features/public-site/components/FeatureCard';
import classes from '@/features/public-site/components/home/Highlights.module.scss';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';
import type { Locale } from '@/i18n/routing';

type Item = PageFrontMatter<'home'>['highlights']['items'][number];
type Props = {
  items: Item[];
  photoById: (id: string) => FacilityPhoto | undefined;
  locale: Locale;
};

/**
 * Three confirmed highlights as a bento: the first is a large photo card with the text on a scrim;
 * the others are horizontal cards with a photo (or a large icon tile when there is no photo).
 */
export function Highlights({ items, photoById, locale }: Props) {
  return (
    <ul className={classes.grid}>
      {items.map((item, i) => {
        const photo = item.photo ? photoById(item.photo) : undefined;
        return (
          <li key={item.title} className={classes.card} data-feature={i === 0 || undefined}>
            <div className={classes.media} data-icon={photo ? undefined : true}>
              {photo ? (
                <Image
                  src={photo.src}
                  alt={photo.alt[locale]}
                  fill
                  sizes={i === 0 ? '(min-width: 62em) 640px, 100vw' : '(min-width: 62em) 240px, 40vw'}
                  unoptimized
                  className={classes.photo}
                  style={{ objectPosition: photo.focalPoint }}
                />
              ) : (
                <FeatureIcon name={item.icon} size={56} />
              )}
            </div>
            <div className={classes.body}>
              <span className={classes.icon}>
                <FeatureIcon name={item.icon} size={22} />
              </span>
              <h3 className={classes.title}>{item.title}</h3>
              <p className={classes.text}>{item.body}</p>
            </div>
          </li>
        );
      })}
    </ul>
  );
}
