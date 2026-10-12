'use client';

import { UnstyledButton } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslations } from 'next-intl';
import shared from '@/components/Community.module.scss';
import { MemberAvatar } from '@/components/MemberAvatar';
import { EmptyState } from '@/components/QueryState';
import { useDeleteGpsVisit, useUpdateGpsVisitVisibility } from '@/features/trips/api/update-gps-visit';
import classes from '@/features/trips/components/Trips.module.scss';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import type { GpsVisit } from '@/types/community';

/** Weather, temperature and, only when the API returned them, the coordinates of a footprint. */
function VisitFacts({ visit, showCoordinates }: { visit: GpsVisit; showCoordinates: boolean }) {
  const t = useTranslations('community');
  const { weatherLabel } = useCommunityLabels();
  const facts = [
    visit.weather ? `🌤️ ${weatherLabel(visit.weather)}` : '',
    visit.temperature !== null ? t('common.temperature', { temp: visit.temperature }) : '',
    showCoordinates && visit.latitude !== null && visit.longitude !== null
      ? t('trips.coordinates', { latitude: visit.latitude.toFixed(4), longitude: visit.longitude.toFixed(4) })
      : ''
  ].filter(Boolean);

  return facts.length > 0 ? <div className={shared.muted}>{facts.join(' · ')}</div> : null;
}

function OwnVisit({ visit }: { visit: GpsVisit }) {
  const t = useTranslations('community.trips');
  const tErrors = useTranslations('errors');
  const update = useUpdateGpsVisitVisibility();
  const remove = useDeleteGpsVisit();
  const hasPosition = visit.latitude !== null && visit.longitude !== null;
  const showFailure = () => notifications.show({ color: 'red', message: tErrors('generic') });

  return (
    <div className={shared.card}>
      <div className={shared.cardBody}>
        <div className={shared.postMeta}>
          <div className={shared.spotEmoji} aria-hidden>
            {visit.emoji || '📍'}
          </div>
          <div className={shared.grow}>
            <strong>{visit.place_name}</strong>
            <div className={shared.muted}>
              {visit.visited_date} · {visit.is_public ? t('public') : t('private')}
            </div>
          </div>
        </div>
        {visit.note && <p className={classes.visitNote}>{visit.note}</p>}
        <VisitFacts visit={visit} showCoordinates />
        {hasPosition && (
          <div className={shared.muted}>
            {visit.is_public && visit.exact_public ? t('exactShared') : t('exactHidden')}
          </div>
        )}
        <div className={shared.actions}>
          <UnstyledButton
            className={shared.action}
            disabled={update.isPending}
            onClick={() => update.mutate({ id: visit.id, is_public: !visit.is_public }, { onError: showFailure })}
          >
            {visit.is_public ? t('makePrivate') : t('makePublic')}
          </UnstyledButton>
          {hasPosition && visit.is_public && (
            <UnstyledButton
              className={shared.action}
              disabled={update.isPending}
              onClick={() =>
                update.mutate(
                  { id: visit.id, is_public: true, exact_public: !visit.exact_public },
                  { onError: showFailure }
                )
              }
            >
              {visit.exact_public ? t('hideExact') : t('shareExact')}
            </UnstyledButton>
          )}
          <UnstyledButton
            className={shared.action}
            data-danger
            disabled={remove.isPending}
            onClick={() =>
              remove.mutate(visit.id, {
                onSuccess: () => notifications.show({ message: t('visitDeleted') }),
                onError: showFailure
              })
            }
          >
            {t('deleteVisit')}
          </UnstyledButton>
        </div>
      </div>
    </div>
  );
}

export function OwnVisitList({ visits }: { visits: GpsVisit[] }) {
  const t = useTranslations('community.trips');
  if (visits.length === 0) {
    return <EmptyState>{t('visitsEmpty')}</EmptyState>;
  }
  return visits.map((visit) => <OwnVisit key={visit.id} visit={visit} />);
}

/** Other members' public footprints. Coordinates appear only when the API included them. */
export function PublicVisitList({ visits }: { visits: GpsVisit[] }) {
  const t = useTranslations('community.trips');
  if (visits.length === 0) {
    return <EmptyState>{t('publicEmpty')}</EmptyState>;
  }
  return visits.map((visit) => (
    <div key={visit.id} className={shared.card}>
      <div className={shared.cardBody}>
        <div className={shared.postMeta}>
          <MemberAvatar src={visit.member.avatar_url} />
          <div className={shared.grow}>
            <strong>{visit.member.nickname}</strong>
            <div className={shared.muted}>
              {visit.emoji || '📍'} {visit.place_name} · {visit.visited_date}
            </div>
          </div>
        </div>
        {visit.note && <p className={classes.visitNote}>{visit.note}</p>}
        <VisitFacts visit={visit} showCoordinates />
      </div>
    </div>
  ));
}
