'use client';

import { UnstyledButton } from '@mantine/core';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import shared from '@/components/Community.module.scss';
import { EmptyState, ErrorState, LoadingState } from '@/components/QueryState';
import { SampleBadge } from '@/components/SampleBadge';
import { paths } from '@/config/paths';
import { type RankingItem, type RankingType, useRankings } from '@/features/feed/api/get-rankings';
import classes from '@/features/feed/components/Feed.module.scss';
import { useCommunityLabels } from '@/hooks/use-community-labels';

const types: RankingType[] = ['spots', 'posts'];

function RankRow({ item, type }: { item: RankingItem; type: RankingType }) {
  const t = useTranslations('community.feed');
  const { areaLabel } = useCommunityLabels();
  const spotId = type === 'spots' ? item.id : item.spot_id;

  const content = (
    <>
      <span className={classes.rankNo} data-rank={item.rank}>
        {item.rank}
      </span>
      {type === 'spots' && (
        <span className={shared.spotEmojiSmall} aria-hidden>
          {item.emoji}
        </span>
      )}
      <span className={classes.rankMain}>
        <strong>{item.name}</strong>
        <small>
          {type === 'spots'
            ? `${[item.area && areaLabel(item.area), t('relatedPosts', { count: item.post_count })].filter(Boolean).join(' · ')}`
            : item.detail}{' '}
          {item.is_sample && <SampleBadge />}
        </small>
      </span>
      <span className={classes.rankScore}>{type === 'spots' ? t('popular') : `♡ ${item.like_count}`}</span>
    </>
  );

  return (
    <li>
      {spotId ? (
        <Link className={classes.rankItem} href={paths.community.spot.getHref(spotId)}>
          {content}
        </Link>
      ) : (
        <div className={classes.rankItem}>{content}</div>
      )}
    </li>
  );
}

/** Top three spots or posts, computed by the API from likes, saves and public footprints. */
export function RankingSection() {
  const t = useTranslations('community.feed');
  const [type, setType] = useState<RankingType>('spots');
  const rankings = useRankings(type);
  const items = rankings.data ?? [];

  return (
    <section style={{ margin: '0 0 18px' }} aria-labelledby='ranking-title'>
      <h2 id='ranking-title' className={shared.h2} style={{ margin: '0 0 3px' }}>
        🏆 {t('rankingTitle')}
      </h2>
      <span className={shared.muted}>{t('rankingLead')}</span>
      <div className={classes.rankTabs}>
        {types.map((value) => (
          <UnstyledButton
            key={value}
            className={classes.rankTab}
            aria-pressed={type === value}
            onClick={() => setType(value)}
          >
            {value === 'spots' ? t('rankSpots') : t('rankPosts')}
          </UnstyledButton>
        ))}
      </div>
      {rankings.isPending ? (
        <LoadingState />
      ) : rankings.isError ? (
        <ErrorState error={rankings.error} onRetry={() => rankings.refetch()} retrying={rankings.isRefetching} />
      ) : items.length === 0 ? (
        <EmptyState>{t('rankEmpty')}</EmptyState>
      ) : (
        <>
          <ol className={classes.rankList}>
            {items.map((item) => (
              <RankRow key={item.id} item={item} type={type} />
            ))}
          </ol>
          {items.some((item) => item.is_sample) && <p className={shared.muted}>{t('sampleNote')}</p>}
        </>
      )}
    </section>
  );
}
