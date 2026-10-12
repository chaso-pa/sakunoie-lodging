'use client';

import { Button } from '@mantine/core';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import shared from '@/components/Community.module.scss';
import { PostCard } from '@/components/PostCard';
import { ErrorState, LoadingState } from '@/components/QueryState';
import { paths } from '@/config/paths';
import { ProfileEditCard } from '@/features/profile/components/ProfileEditCard';
import { useMe } from '@/hooks/use-me';
import { useMyPosts } from '@/hooks/use-my-posts';

type ProfilePageProps = {
  /** Sign-out control, supplied by the app layer because it belongs to the auth feature. */
  logoutButton: React.ReactNode;
};

/** My page: profile editing, account notes and the way into the read-only 佐久の家 page. */
export function ProfilePage({ logoutButton }: ProfilePageProps) {
  const t = useTranslations('community');
  const { data: member } = useMe();
  const mine = useMyPosts();
  const posts = mine.data ?? [];

  return (
    <>
      <div className={shared.heroCentered}>
        <div style={{ fontSize: 45, lineHeight: 1.2 }} aria-hidden>
          🏡
        </div>
        <div className={shared.eyebrow}>{t('profile.heroEyebrow')}</div>
        <h1 className={shared.h1}>{t('business.name')}</h1>
        <p>{t('profile.heroLead')}</p>
        <Button component={Link} href={paths.community.business.getHref()}>
          {t('profile.businessCta')}
        </Button>
      </div>

      <div className={shared.statRow}>
        <div className={shared.stat}>
          <strong>{mine.data ? posts.length : '–'}</strong>
          <small>{t('profile.statPosts')}</small>
        </div>
        <div className={shared.stat}>
          <strong>{mine.data ? posts.reduce((sum, post) => sum + post.like_count, 0) : '–'}</strong>
          <small>{t('profile.statLikes')}</small>
        </div>
        <div className={shared.stat}>
          <strong>{mine.data ? new Set(posts.flatMap((post) => (post.spot ? [post.spot.id] : []))).size : '–'}</strong>
          <small>{t('profile.statSpots')}</small>
        </div>
      </div>

      {member && <ProfileEditCard member={member} />}

      <div className={shared.card}>
        <div className={shared.cardBody}>
          <h2 className={shared.h2}>{t('profile.accountTitle')}</h2>
          <div className={shared.note}>
            <strong>{t('profile.transferTitle')}</strong>
            {t('profile.transferBody')}
          </div>
          <div style={{ marginTop: 14 }}>{logoutButton}</div>
          <p className={shared.muted} style={{ marginTop: 10 }}>
            {t('profile.logoutNote')}
          </p>
        </div>
      </div>

      <h2 className={shared.h2}>{t('profile.recentPosts')}</h2>
      {mine.isPending ? (
        <LoadingState />
      ) : mine.isError ? (
        <ErrorState error={mine.error} onRetry={() => mine.refetch()} retrying={mine.isRefetching} />
      ) : posts.length === 0 ? (
        <div className={shared.empty}>
          {t('trips.myNotesEmpty')}
          <div>
            <Button component={Link} href={paths.community.post.getHref()} mt={12}>
              {t('trips.writeFirst')}
            </Button>
          </div>
        </div>
      ) : (
        posts.slice(0, 2).map((post) => <PostCard key={post.id} post={post} />)
      )}
    </>
  );
}
