'use client';

import { Button, TextInput, UnstyledButton } from '@mantine/core';
import { useDebouncedValue } from '@mantine/hooks';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { AreaChips } from '@/components/AreaChips';
import shared from '@/components/Community.module.scss';
import { PostCard, PostMedia } from '@/components/PostCard';
import { EmptyState, ErrorState, LoadingState } from '@/components/QueryState';
import { paths } from '@/config/paths';
import { useFeedPosts } from '@/features/feed/api/get-posts';
import classes from '@/features/feed/components/Feed.module.scss';
import { FilterPanel } from '@/features/feed/components/FilterPanel';
import { RankingSection } from '@/features/feed/components/RankingSection';

export function FeedPage() {
  const t = useTranslations('community');
  const [area, setArea] = useState('');
  const [weather, setWeather] = useState('');
  const [month, setMonth] = useState('');
  const [search, setSearch] = useState('');
  const [mode, setMode] = useState<'list' | 'grid'>('list');
  const [q] = useDebouncedValue(search.trim(), 300);

  const feed = useFeedPosts({ area, weather, month, q });
  const posts = feed.data?.pages.flatMap((page) => page.posts) ?? [];
  const total = feed.data?.pages[0]?.total ?? 0;

  const reset = () => {
    setArea('');
    setWeather('');
    setMonth('');
    setSearch('');
  };

  return (
    <>
      <div className={shared.hero}>
        <div className={shared.eyebrow}>{t('feed.eyebrow')}</div>
        <h1 className={shared.h1}>
          {t('feed.titleLine1')}
          <br />
          {t('feed.titleLine2')}
        </h1>
        <p>{t('feed.lead')}</p>
        <Button component={Link} href={paths.community.map.getHref()}>
          ⌖ {t('feed.exploreCta')}
        </Button>
      </div>

      <RankingSection />

      <TextInput
        className={classes.search}
        type='search'
        size='md'
        radius='md'
        value={search}
        onChange={(event) => setSearch(event.currentTarget.value)}
        placeholder={t('common.searchPlaceholder')}
        aria-label={t('common.searchLabel')}
        maxLength={100}
      />
      <AreaChips value={area} onChange={setArea} />
      <FilterPanel
        weather={weather}
        month={month}
        onWeatherChange={setWeather}
        onMonthChange={setMonth}
        onReset={reset}
      />

      <div className={classes.modeBar}>
        <div>
          <h2 className={shared.h2}>{t('feed.listTitle')}</h2>
          <output className={shared.muted}>{feed.isSuccess ? t('feed.listMeta', { count: total }) : ''}</output>
        </div>
        <div className={classes.feedSwitch}>
          <UnstyledButton
            className={classes.viewBtn}
            aria-label={t('feed.gridView')}
            aria-pressed={mode === 'grid'}
            onClick={() => setMode('grid')}
          >
            ▦
          </UnstyledButton>
          <UnstyledButton
            className={classes.viewBtn}
            aria-label={t('feed.listView')}
            aria-pressed={mode === 'list'}
            onClick={() => setMode('list')}
          >
            ☰
          </UnstyledButton>
        </div>
      </div>

      {feed.isPending ? (
        <LoadingState />
      ) : feed.isError ? (
        <ErrorState error={feed.error} onRetry={() => feed.refetch()} retrying={feed.isRefetching} />
      ) : posts.length === 0 ? (
        <EmptyState>{t('feed.empty')}</EmptyState>
      ) : mode === 'grid' ? (
        <div className={classes.feedGrid}>
          {posts.map((post) => {
            const label = `${post.spot?.name ?? post.member.nickname}　♡ ${post.like_count}`;
            const tile = (
              <>
                <PostMedia post={post} compact />
                <div className={classes.gridStats}>{label}</div>
              </>
            );
            return post.spot ? (
              <Link key={post.id} className={classes.gridPost} href={paths.community.spot.getHref(post.spot.id)}>
                {tile}
              </Link>
            ) : (
              <div key={post.id} className={classes.gridPost}>
                {tile}
              </div>
            );
          })}
        </div>
      ) : (
        posts.map((post) => <PostCard key={post.id} post={post} />)
      )}

      {feed.hasNextPage && (
        <Button variant='light' fullWidth loading={feed.isFetchingNextPage} onClick={() => feed.fetchNextPage()}>
          {t('feed.loadMore')}
        </Button>
      )}
      {feed.isFetchNextPageError && <ErrorState error={feed.error} onRetry={() => feed.fetchNextPage()} />}
      {posts.some((post) => post.is_sample) && (
        <p className={shared.muted} style={{ marginTop: 12 }}>
          {t('common.sampleNote')}
        </p>
      )}
    </>
  );
}
