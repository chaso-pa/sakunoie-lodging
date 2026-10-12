'use client';

import { Button, TextInput, UnstyledButton } from '@mantine/core';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { AreaChips } from '@/components/AreaChips';
import shared from '@/components/Community.module.scss';
import { EmptyState, ErrorState, LoadingState } from '@/components/QueryState';
import { SampleBadge } from '@/components/SampleBadge';
import { SpotRow } from '@/components/SpotRow';
import { paths } from '@/config/paths';
import { FootprintRecorder } from '@/features/map/components/FootprintRecorder';
import classes from '@/features/map/components/Map.module.scss';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { useSpots } from '@/hooks/use-spots';
import type { SpotListItem } from '@/types/community';

// Pin positions on the illustrated map, in percent. This is a picture of a map, not a projection:
// spots are placed in list order, as in the prototype.
const pinSlots = [
  [9, 18],
  [48, 31],
  [58, 62],
  [14, 70],
  [36, 48],
  [60, 10],
  [8, 44],
  [44, 82]
] as const;

const pinPosition = (index: number) => {
  const [left, top] = pinSlots[index % pinSlots.length];
  const lap = Math.floor(index / pinSlots.length);
  return { left: `${Math.min(64, left + lap * 5)}%`, top: `${Math.min(88, top + lap * 7)}%` };
};

function SelectedSpot({ spot }: { spot: SpotListItem }) {
  const t = useTranslations('community.map');
  const { areaLabel } = useCommunityLabels();

  return (
    <div className={shared.card}>
      <div className={shared.cardBody}>
        <span className={shared.pill}>
          {areaLabel(spot.area)} · {spot.category}
        </span>{' '}
        {spot.is_sample && <SampleBadge />}
        <h2 className={shared.h2} style={{ marginTop: 10 }}>
          {spot.emoji} {spot.name}
        </h2>
        <p className={shared.body}>{spot.description}</p>
        <Button component={Link} href={paths.community.spot.getHref(spot.id)} fullWidth mt={10} size='md'>
          {t('detailCta')}
        </Button>
        <Button
          component={Link}
          href={paths.community.post.getHref(spot.id)}
          fullWidth
          mt={10}
          size='md'
          variant='light'
        >
          {t('postCta')}
        </Button>
      </div>
    </div>
  );
}

/** Map-style spot explorer: a drawn map with tappable pins, the footprint recorder and the spot list. */
export function MapPage() {
  const t = useTranslations('community');
  const { areaLabel } = useCommunityLabels();
  const spots = useSpots();
  const [area, setArea] = useState('');
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const all = spots.data ?? [];
  const keyword = search.trim().toLowerCase();
  const inArea = all.filter((spot) => !area || spot.area === area);
  const shown = inArea.filter(
    (spot) =>
      !keyword || [spot.name, spot.area, spot.category, spot.description].join(' ').toLowerCase().includes(keyword)
  );
  const selected = all.find((spot) => spot.id === selectedId);

  return (
    <>
      <div className={shared.eyebrow}>{t('map.eyebrow')}</div>
      <h1 className={shared.h1}>{t('map.title')}</h1>
      <p className={shared.muted} style={{ marginBottom: 14 }}>
        {t('map.lead')}
      </p>

      <FootprintRecorder spots={all} />

      <TextInput
        type='search'
        size='md'
        radius='md'
        mb={10}
        value={search}
        onChange={(event) => setSearch(event.currentTarget.value)}
        placeholder={t('common.searchPlaceholder')}
        aria-label={t('common.searchLabel')}
      />
      <AreaChips value={area} onChange={setArea} />

      {spots.isPending ? (
        <LoadingState />
      ) : spots.isError ? (
        <ErrorState error={spots.error} onRetry={() => spots.refetch()} retrying={spots.isRefetching} />
      ) : (
        <>
          {/* biome-ignore lint/a11y/useSemanticElements: a group of pin buttons over a drawing, not a form fieldset */}
          <div className={classes.mapBox} role='group' aria-label={t('map.mapLabel')}>
            <div className={classes.river} aria-hidden />
            {inArea.map((spot, index) => (
              <UnstyledButton
                key={spot.id}
                className={classes.pin}
                style={pinPosition(index)}
                aria-pressed={selectedId === spot.id}
                onClick={() => setSelectedId(spot.id)}
              >
                {spot.emoji} {spot.name}
              </UnstyledButton>
            ))}
          </div>
          <p className={shared.muted} style={{ margin: '-6px 0 14px' }}>
            {t('map.mapNote')}
          </p>
          {selected && <SelectedSpot spot={selected} />}

          <h2 className={shared.h2}>
            {t('map.listTitle')} <span className={shared.muted}>({shown.length})</span>
          </h2>
          {shown.length === 0 ? (
            <EmptyState>{t('map.empty')}</EmptyState>
          ) : (
            <div className={classes.spotList}>
              {shown.map((spot) => (
                <SpotRow
                  key={spot.id}
                  spot={spot}
                  href={paths.community.spot.getHref(spot.id)}
                  meta={`${areaLabel(spot.area)} · ${spot.category}`}
                />
              ))}
            </div>
          )}
          {shown.some((spot) => spot.is_sample) && <p className={shared.muted}>{t('common.sampleNote')}</p>}
        </>
      )}
    </>
  );
}
