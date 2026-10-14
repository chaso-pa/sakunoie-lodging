'use client';

import { UnstyledButton } from '@mantine/core';
import { useTranslations } from 'next-intl';
import shared from '@/components/Community.module.scss';
import { GoogleMapsLogo } from '@/components/GoogleMapsLogo';
import classes from '@/features/post/components/Place.module.scss';
import { type PlaceAttribution as Attribution, googleSource } from '@/features/post/types/place';

export type PlaceRow = {
  key: string;
  name: string;
  /** The local-language name, when the provider gave one that differs from `name`. */
  originalName?: string | null;
  /** Address, or whatever locates the place when there is none. */
  detail: string;
  /** Already in the display language. */
  category: string;
  distanceM?: number | null;
  emoji: string;
  photoUrl?: string | null;
  onSelect: () => void;
  /** Opens the picker's own map. Absent when the place has no position or may not be drawn there. */
  onShowOnMap?: () => void;
  /** Where to see the place when it cannot go on the picker's map. */
  mapUrl?: string | null;
};

/** Candidate list. Names and addresses are printed as given; they are data, not interface text. */
export function PlaceRows({ rows, label }: { rows: PlaceRow[]; label: string }) {
  const t = useTranslations('community.compose.place');

  return (
    <ul className={classes.list} aria-label={label}>
      {rows.map((row) => (
        <li key={row.key} className={classes.row}>
          <UnstyledButton
            className={classes.rowMain}
            onClick={row.onSelect}
            aria-label={t('selectAria', { name: row.name })}
          >
            <span className={`${shared.spotEmojiSmall} ${classes.thumb}`} aria-hidden>
              {row.photoUrl ? (
                // biome-ignore lint/performance/noImgElement: a provider's thumbnail on an arbitrary host, not for next/image
                <img src={row.photoUrl} alt='' loading='lazy' referrerPolicy='no-referrer' />
              ) : (
                row.emoji
              )}
            </span>
            <span className={classes.rowText} translate='no'>
              <span className={classes.name}>{row.name}</span>
              {row.originalName && <span className={classes.originalName}>{row.originalName}</span>}
              {row.detail && <span className={classes.address}>{row.detail}</span>}
              {(row.category || typeof row.distanceM === 'number') && (
                <span className={classes.rowTags}>
                  {row.category && <span className={shared.pill}>{row.category}</span>}
                  {typeof row.distanceM === 'number' && (
                    <span className={classes.distance}>{t('distance', { meters: row.distanceM })}</span>
                  )}
                </span>
              )}
            </span>
          </UnstyledButton>
          {row.onShowOnMap ? (
            <UnstyledButton
              className={classes.rowMap}
              onClick={row.onShowOnMap}
              aria-label={t('showOnMapAria', { name: row.name })}
            >
              <span aria-hidden>🗺</span>
              {t('showOnMap')}
            </UnstyledButton>
          ) : (
            row.mapUrl && (
              <a
                className={classes.rowMap}
                href={row.mapUrl}
                target='_blank'
                rel='noreferrer'
                aria-label={t('openExternalMapAria', { name: row.name })}
              >
                <span aria-hidden>↗</span>
                {t('showOnMap')}
              </a>
            )
          )}
        </li>
      ))}
    </ul>
  );
}

/**
 * Credit for the data in a result list, as the provider's terms require. Google's is its logo rather than
 * the text the API sends.
 */
export function PlaceAttribution({ provider, attribution }: { provider: string; attribution: Attribution }) {
  if (provider === googleSource) {
    return (
      <p className={classes.attribution} data-google>
        <GoogleMapsLogo />
      </p>
    );
  }
  if (!attribution.text) {
    return null;
  }
  return (
    <p className={classes.attribution} translate='no'>
      {attribution.url ? (
        <a href={attribution.url} target='_blank' rel='noreferrer'>
          {attribution.text}
        </a>
      ) : (
        attribution.text
      )}
    </p>
  );
}
