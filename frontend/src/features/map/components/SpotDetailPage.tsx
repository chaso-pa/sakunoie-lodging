'use client';

import { Button } from '@mantine/core';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import shared from '@/components/Community.module.scss';
import { MemberAvatar } from '@/components/MemberAvatar';
import { PostPhotos } from '@/components/PostPhotos';
import { EmptyState, ErrorState, LoadingState } from '@/components/QueryState';
import { SampleBadge } from '@/components/SampleBadge';
import { SpotArt } from '@/components/SpotArt';
import { SpotCredit } from '@/components/SpotCredit';
import { TranslatableText } from '@/components/TranslatableText';
import { paths } from '@/config/paths';
import { type SpotPost, useSpot } from '@/features/map/api/get-spot';
import classes from '@/features/map/components/Map.module.scss';
import { ReviewSection } from '@/features/map/components/ReviewSection';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { getApiErrorStatus } from '@/lib/api-client';

function SpotPostCard({ post, emoji, spotName }: { post: SpotPost; emoji: string; spotName: string }) {
  const photos = post.media.filter((media) => media.type === 'photo');

  return (
    <article className={shared.card}>
      {photos.length > 0 ? (
        <PostPhotos photos={photos} nickname={post.member.nickname} />
      ) : (
        <SpotArt emoji={emoji} title={spotName} />
      )}
      <div className={shared.cardBody}>
        <div className={classes.spotPostMeta}>
          <MemberAvatar src={post.member.avatar_url} />
          <div className={shared.grow}>
            <strong>{post.member.nickname}</strong> {post.is_sample && <SampleBadge />}
            <div className={shared.muted}>
              {post.created_at.slice(0, 10)} · ♡ {post.like_count}
            </div>
          </div>
        </div>
        {post.caption && <TranslatableText text={post.caption} />}
      </div>
    </article>
  );
}

/** Spot detail: description, reviews with a form to add one, and the posts filed under the spot. */
export function SpotDetailPage({ spotId }: { spotId: string }) {
  const t = useTranslations('community');
  const { areaLabel } = useCommunityLabels();
  const detail = useSpot(spotId);

  const back = (
    <Link className={shared.back} href={paths.community.map.getHref()}>
      ← {t('common.back')}
    </Link>
  );

  if (detail.isPending) {
    return (
      <>
        {back}
        <LoadingState />
      </>
    );
  }
  if (detail.isError) {
    const notFound = getApiErrorStatus(detail.error) === 404;
    return (
      <>
        {back}
        <ErrorState
          error={detail.error}
          message={notFound ? t('spot.notFound') : undefined}
          onRetry={notFound ? undefined : () => detail.refetch()}
          retrying={detail.isRefetching}
        />
      </>
    );
  }

  const { spot, reviews, posts } = detail.data;
  const average = reviews.length ? reviews.reduce((sum, review) => sum + review.stars, 0) / reviews.length : null;

  return (
    <>
      {back}
      <div className={shared.card}>
        <SpotArt emoji={spot.emoji} title={spot.name} />
      </div>
      <div style={{ padding: '1px 2px 0' }}>
        <span className={shared.pill}>{[areaLabel(spot.area), spot.category].filter(Boolean).join(' · ')}</span>{' '}
        {spot.is_sample && <SampleBadge />}
        <h1 className={shared.h1} style={{ marginTop: 10 }}>
          {spot.emoji} {spot.name}
        </h1>
        {spot.address && (
          <p className={shared.muted} translate='no'>
            {spot.address}
          </p>
        )}
        <SpotCredit source={spot.source} />
        <p className={shared.body} style={{ fontSize: 14 }}>
          {spot.description}
        </p>
        <div className={shared.statRow}>
          <div className={shared.stat}>
            <strong>{reviews.length}</strong>
            <small>{t('spot.reviewsStat')}</small>
          </div>
          <div className={shared.stat}>
            <strong>{average === null ? '–' : average.toFixed(1)}</strong>
            <small>{t('spot.ratingStat')}</small>
          </div>
          <div className={shared.stat}>
            <strong>{posts.length}</strong>
            <small>{t('spot.postsStat')}</small>
          </div>
        </div>
        <Button component={Link} href={paths.community.post.getHref(spot.id)} fullWidth size='md'>
          ＋ {t('spot.postCta')}
        </Button>
        <ReviewSection spotId={spot.id} reviews={reviews} />
        <h2 className={shared.sectionTitle}>{t('spot.postsTitle')}</h2>
        {posts.length === 0 ? (
          <EmptyState>{t('spot.noPosts')}</EmptyState>
        ) : (
          posts.map((post) => <SpotPostCard key={post.id} post={post} emoji={spot.emoji} spotName={spot.name} />)
        )}
        {(spot.is_sample || reviews.some((review) => review.is_sample) || posts.some((post) => post.is_sample)) && (
          <p className={shared.muted}>{t('common.sampleNote')}</p>
        )}
      </div>
    </>
  );
}
