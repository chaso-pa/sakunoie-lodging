import Image from 'next/image';
import { FeatureIcon } from '@/features/public-site/components/FeatureCard';
import classes from '@/features/public-site/components/home/NearbyCards.module.scss';
import { TravelTime } from '@/features/public-site/components/home/TravelTime';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Props = { items: PageFrontMatter<'home'>['around']['items'] };

/**
 * Nearby spots (owner-confirmed): icon, name, a one-line description and the travel time. The first
 * item (Karuizawa) is the featured card, with its optional photo behind a dark scrim; `localName` is
 * the Japanese name, for maps.
 */
export function NearbyCards({ items }: Props) {
  return (
    <ul className={classes.grid}>
      {items.map((item, i) => (
        <li
          key={item.name}
          className={classes.card}
          data-feature={i === 0 || undefined}
          data-photo={(i === 0 && Boolean(item.image)) || undefined}
        >
          {i === 0 && item.image ? (
            <>
              <Image
                src={item.image.src}
                alt={item.image.alt}
                fill
                unoptimized
                sizes='(min-width: 992px) 40vw, 100vw'
                className={classes.photo}
              />
              <span className={classes.photoScrim} aria-hidden />
            </>
          ) : null}
          <span className={classes.icon}>
            <FeatureIcon name={item.icon} size={i === 0 ? 28 : 22} />
          </span>
          <div className={classes.body}>
            <h3 className={classes.title}>
              {item.name}
              {item.localName ? (
                <span className={classes.localName} lang='ja'>
                  {item.localName}
                </span>
              ) : null}
            </h3>
            <p className={classes.text}>{item.description}</p>
          </div>
          <TravelTime travel={item.travel} className={classes.travel} />
        </li>
      ))}
    </ul>
  );
}
