'use client';

import { useTranslations } from 'next-intl';
import { useId } from 'react';
import classes from '@/components/Community.module.scss';

type SpotArtProps = {
  emoji: string;
  title: string;
  /** Hide the "Nagano · TABINOTE" label and shrink the caption, for grid tiles. */
  compact?: boolean;
};

/** The prototype's illustrated placeholder, drawn when a post or a spot has no photo. */
export function SpotArt({ emoji, title, compact }: SpotArtProps) {
  const t = useTranslations('community.common');
  const gradientId = useId();

  return (
    <div className={classes.photo} style={compact ? { height: '100%' } : undefined}>
      <svg viewBox='0 0 500 250' preserveAspectRatio='xMidYMid slice' role='img' aria-label={t('artAlt')}>
        <defs>
          <linearGradient id={gradientId} x1='0' y1='0' x2='0' y2='1'>
            <stop stopColor='#b8d9d1' />
            <stop offset='1' stopColor='#edf0d9' />
          </linearGradient>
        </defs>
        <rect width='500' height='250' fill={`url(#${gradientId})`} />
        <circle cx='390' cy='45' r='26' fill='#fff4cc' />
        <path d='M0 170 L100 65 205 175 300 75 500 180 V250 H0Z' fill='#789e86' />
        <path d='M0 195 L150 115 260 210 370 125 500 195 V250 H0Z' fill='#477b63' />
        <path d='M0 220 Q130 180 250 220 T500 215 V250 H0Z' fill='#a8c6a1' />
        <path d='M80 250 Q160 190 240 250' stroke='#e4e6c9' strokeWidth='18' fill='none' />
      </svg>
      {!compact && <div className={classes.photoLabel}>{t('artLabel')}</div>}
      <div className={classes.photoText} style={compact ? { fontSize: 13, left: 7, bottom: 7, right: 7 } : undefined}>
        {emoji} {title}
      </div>
    </div>
  );
}
