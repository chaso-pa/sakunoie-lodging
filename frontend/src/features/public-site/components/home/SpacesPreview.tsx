import { IconKey, IconUsers } from '@tabler/icons-react';
import Image from 'next/image';
import type { FacilityPhoto } from '@/config/site';
import classes from '@/features/public-site/components/home/SpacesPreview.module.scss';
import type { Locale } from '@/i18n/routing';

type Tile = { photo: FacilityPhoto; label: string; kind: 'private' | 'shared' };
type Props = {
  tiles: Tile[];
  badges: { private: string; shared: string };
  locale: Locale;
};

/**
 * Photo mosaic of the guest room and the shared spaces (none of them hero slides), each captioned with
 * a text badge "専用"/"共用" plus the space name, so private vs shared never relies on colour alone.
 */
export function SpacesPreview({ tiles, badges, locale }: Props) {
  return (
    <ul className={classes.grid} data-count={tiles.length}>
      {tiles.map(({ photo, label, kind }, i) => (
        <li key={photo.id} className={classes.tile}>
          <Image
            src={photo.src}
            alt={photo.alt[locale]}
            fill
            sizes={i === 0 ? '(min-width: 62em) 560px, 100vw' : '(min-width: 62em) 260px, 33vw'}
            unoptimized
            className={classes.photo}
            style={{ objectPosition: photo.focalPoint }}
          />
          <p className={classes.caption}>
            <span className={classes.badge} data-kind={kind}>
              {kind === 'private' ? <IconKey size={13} aria-hidden /> : <IconUsers size={13} aria-hidden />}
              {kind === 'private' ? badges.private : badges.shared}
            </span>
            <span className={classes.label}>{label}</span>
          </p>
        </li>
      ))}
    </ul>
  );
}
