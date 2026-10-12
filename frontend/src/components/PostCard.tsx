'use client';

import { Button, Group, Image, Modal, UnstyledButton } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import classes from '@/components/Community.module.scss';
import { MemberAvatar } from '@/components/MemberAvatar';
import { SampleBadge } from '@/components/SampleBadge';
import { SpotArt } from '@/components/SpotArt';
import { TranslatableText } from '@/components/TranslatableText';
import { paths } from '@/config/paths';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { useDeletePost, useToggleBookmark, useToggleLike } from '@/hooks/use-post-actions';
import type { Post } from '@/types/community';
import { getWeatherIcon } from '@/utils/community-labels';

/** Date a post is filed under: the visit date when given, otherwise the day it was posted. */
export const getPostDate = (post: Post) => post.visited_at ?? post.created_at.slice(0, 10);

export function PostMedia({ post, compact }: { post: Post; compact?: boolean }) {
  const t = useTranslations('community');
  const photo = post.media.find((media) => media.type === 'photo');
  const placeName = post.spot?.name ?? t('common.unknownSpot');

  if (photo) {
    return (
      <div className={classes.mediaBox} style={compact ? { height: '100%' } : undefined}>
        <Image
          src={photo.url}
          alt={t('post.photoAlt', { nickname: post.member.nickname })}
          loading='lazy'
          h='100%'
          fit='cover'
        />
      </div>
    );
  }
  return <SpotArt emoji={post.spot?.emoji ?? '📍'} title={placeName} compact={compact} />;
}

/** One post with its like, save, translate and (for the author) delete controls. */
export function PostCard({ post }: { post: Post }) {
  const t = useTranslations('community');
  const tErrors = useTranslations('errors');
  const { areaLabel, weatherLabel } = useCommunityLabels();
  const like = useToggleLike();
  const bookmark = useToggleBookmark();
  const remove = useDeletePost();
  const [confirming, setConfirming] = useState(false);

  const showFailure = () => notifications.show({ color: 'red', message: tErrors('generic') });
  const date = getPostDate(post);

  return (
    <article className={classes.card}>
      <PostMedia post={post} />
      <div className={classes.cardBody}>
        <div className={classes.postMeta}>
          <MemberAvatar src={post.member.avatar_url} />
          <div className={classes.grow}>
            <strong>{post.member.nickname}</strong> {post.is_sample && <SampleBadge />}
            <div className={classes.muted}>
              {post.area ? `${areaLabel(post.area)} · ` : ''}
              {date}
            </div>
          </div>
          <UnstyledButton
            className={classes.action}
            data-save
            aria-pressed={post.is_bookmarked}
            onClick={() =>
              bookmark.mutate(post.id, {
                onSuccess: (result) =>
                  notifications.show({ message: result.bookmarked ? t('post.savedToast') : t('post.unsavedToast') }),
                onError: showFailure
              })
            }
          >
            {post.is_bookmarked ? t('post.saved') : t('post.save')}
          </UnstyledButton>
        </div>
        {post.caption && <TranslatableText text={post.caption} sourceLang={post.language} />}
        {(post.weather || post.temperature !== null) && (
          <div className={classes.weather}>
            {post.weather && (
              <span>
                {getWeatherIcon(post.weather)} {weatherLabel(post.weather)}
              </span>
            )}
            {post.temperature !== null && <span>{t('common.temperature', { temp: post.temperature })}</span>}
            <span aria-hidden style={{ color: '#aab5af' }}>
              |
            </span>
            <span>📅 {date}</span>
          </div>
        )}
        {post.tags.length > 0 && (
          <div className={classes.tagRow}>
            {post.tags.map((tag) => (
              <span key={tag} className={classes.tag}>
                #{tag}
              </span>
            ))}
          </div>
        )}
        <div className={classes.actions}>
          <UnstyledButton
            className={classes.action}
            data-like
            aria-pressed={post.is_liked}
            onClick={() => like.mutate(post.id, { onError: showFailure })}
          >
            {post.is_liked ? '♥' : '♡'} {t('post.like')} {post.like_count}
          </UnstyledButton>
          {post.spot && (
            <Link className={classes.action} href={paths.community.spot.getHref(post.spot.id)}>
              ⌖ {t('post.viewSpot')}
            </Link>
          )}
          {post.is_mine && (
            <UnstyledButton className={classes.action} data-danger onClick={() => setConfirming(true)}>
              {t('post.delete')}
            </UnstyledButton>
          )}
        </div>
      </div>
      {post.is_mine && (
        <Modal opened={confirming} onClose={() => setConfirming(false)} title={t('post.deleteTitle')} centered>
          <p className={classes.muted}>{t('post.deleteBody')}</p>
          <Group justify='flex-end' mt='md'>
            <Button variant='default' onClick={() => setConfirming(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              color='red'
              loading={remove.isPending}
              onClick={() =>
                remove.mutate(post.id, {
                  onSuccess: () => {
                    setConfirming(false);
                    notifications.show({ message: t('post.deleted') });
                  },
                  onError: showFailure
                })
              }
            >
              {t('post.delete')}
            </Button>
          </Group>
        </Modal>
      )}
    </article>
  );
}
