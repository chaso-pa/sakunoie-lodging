'use client';

import { Button } from '@mantine/core';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import shared from '@/components/Community.module.scss';
import { PostCard } from '@/components/PostCard';
import { EmptyState, ErrorState, LoadingState } from '@/components/QueryState';
import { paths } from '@/config/paths';
import { useBookmarkedPosts } from '@/features/trips/api/get-bookmarked-posts';
import { useMyGpsVisits, usePublicGpsVisits } from '@/features/trips/api/get-gps-visits';
import { FootprintMap } from '@/features/trips/components/FootprintMap';
import classes from '@/features/trips/components/Trips.module.scss';
import { OwnVisitList, PublicVisitList } from '@/features/trips/components/VisitList';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { useMyPosts } from '@/hooks/use-my-posts';
import { useSpots } from '@/hooks/use-spots';

type AnyQuery = { isPending: boolean; isError: boolean; error: unknown; isRefetching: boolean; refetch: () => unknown };

/** Loading, error-with-retry or the section's content, so one failing list leaves the others usable. */
function Section({ query, children }: { query: AnyQuery; children: () => React.ReactNode }) {
  if (query.isPending) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} retrying={query.isRefetching} />;
  }
  return children();
}

const count = (query: { data?: unknown[] }) => (query.data ? query.data.length : '–');

export function TripsPage() {
  const t = useTranslations('community');
  const { areaLabel } = useCommunityLabels();
  const mine = useMyPosts();
  const saved = useBookmarkedPosts();
  const visits = useMyGpsVisits();
  const publicVisits = usePublicGpsVisits();
  const spots = useSpots();

  const visitedPlaces =
    mine.data && visits.data
      ? new Set([
          ...mine.data.flatMap((post) => (post.spot ? [post.spot.name] : [])),
          ...visits.data.map((visit) => visit.place_name)
        ]).size
      : '–';
  const others = (publicVisits.data ?? []).filter((visit) => !visit.is_mine);

  return (
    <>
      <div className={shared.eyebrow}>{t('trips.eyebrow')}</div>
      <h1 className={shared.h1}>{t('trips.title')}</h1>
      <p className={shared.muted}>{t('trips.lead')}</p>
      <div className={shared.statRow}>
        <div className={shared.stat}>
          <strong>{count(mine)}</strong>
          <small>{t('trips.statPosts')}</small>
        </div>
        <div className={shared.stat}>
          <strong>{visitedPlaces}</strong>
          <small>{t('trips.statPlaces')}</small>
        </div>
        <div className={shared.stat}>
          <strong>{count(saved)}</strong>
          <small>{t('trips.statSaved')}</small>
        </div>
      </div>

      <h2 className={shared.h2}>{t('trips.myNotes')}</h2>
      <Section query={mine}>
        {() =>
          mine.data?.length ? (
            mine.data.map((post) => <PostCard key={post.id} post={post} />)
          ) : (
            <EmptyState>
              {t('trips.myNotesEmpty')}
              <div>
                <Button component={Link} href={paths.community.post.getHref()} mt={12}>
                  {t('trips.writeFirst')}
                </Button>
              </div>
            </EmptyState>
          )
        }
      </Section>

      <h2 className={shared.sectionTitle}>{t('trips.saved')}</h2>
      <Section query={saved}>
        {() =>
          saved.data?.length ? (
            saved.data.map((post) => <PostCard key={post.id} post={post} />)
          ) : (
            <EmptyState>{t('trips.savedEmpty')}</EmptyState>
          )
        }
      </Section>

      <h2 className={shared.sectionTitle}>🗺️ {t('trips.mapTitle')}</h2>
      <Section query={visits}>{() => <FootprintMap visits={visits.data ?? []} />}</Section>

      <h2 className={shared.sectionTitle}>
        📍 {t('trips.visitsTitle')} <span className={shared.muted}>({count(visits)})</span>
      </h2>
      <Section query={visits}>{() => <OwnVisitList visits={visits.data ?? []} />}</Section>

      <h2 className={shared.sectionTitle}>{t('trips.publicTitle')}</h2>
      <p className={shared.muted} style={{ margin: '-6px 0 10px' }}>
        {t('trips.publicLead')}
      </p>
      <Section query={publicVisits}>{() => <PublicVisitList visits={others} />}</Section>

      <h2 className={shared.sectionTitle}>{t('trips.wantToGo')}</h2>
      <Section query={spots}>
        {() =>
          spots.data?.length ? (
            <div className={classes.spotGrid}>
              {spots.data.slice(0, 4).map((spot) => (
                <Link key={spot.id} className={classes.spot} href={paths.community.spot.getHref(spot.id)}>
                  <div className={classes.spotArt} aria-hidden>
                    <span>{spot.emoji}</span>
                  </div>
                  <div className={classes.spotText}>
                    <strong>{spot.name}</strong>
                    <br />
                    <small>
                      {areaLabel(spot.area)} · {spot.category}
                    </small>
                  </div>
                </Link>
              ))}
            </div>
          ) : (
            <EmptyState>{t('map.empty')}</EmptyState>
          )
        }
      </Section>
    </>
  );
}
