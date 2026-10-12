'use client';

import { useTranslations } from 'next-intl';
import shared from '@/components/Community.module.scss';
import { EmptyState } from '@/components/QueryState';
import classes from '@/features/trips/components/Trips.module.scss';
import type { GpsVisit } from '@/types/community';

type Located = GpsVisit & { latitude: number; longitude: number };

const hasPosition = (visit: GpsVisit): visit is Located => visit.latitude !== null && visit.longitude !== null;

/**
 * The member's own footprints joined in visit order. A drawing scaled to the recorded points, with no
 * map tiles behind it. Only the member's own visits are passed in, so nobody else's position is drawn.
 */
export function FootprintMap({ visits }: { visits: GpsVisit[] }) {
  const t = useTranslations('community.trips');
  const points = visits
    .filter(hasPosition)
    .sort((a, b) => `${a.visited_date}${a.created_at}`.localeCompare(`${b.visited_date}${b.created_at}`));

  if (points.length === 0) {
    return <EmptyState>{t('mapEmpty')}</EmptyState>;
  }
  if (points.length === 1) {
    return (
      <div className={classes.singleBox}>
        <div className={classes.singleCard}>
          <div style={{ fontSize: 25 }} aria-hidden>
            📍
          </div>
          <strong>{points[0].place_name}</strong>
          <div className={shared.muted}>{t('mapSingle')}</div>
        </div>
      </div>
    );
  }

  const latitudes = points.map((point) => point.latitude);
  const longitudes = points.map((point) => point.longitude);
  const minLat = Math.min(...latitudes);
  const maxLat = Math.max(...latitudes);
  const minLon = Math.min(...longitudes);
  const maxLon = Math.max(...longitudes);
  const xy = points.map((point) => ({
    id: point.id,
    x: 22 + (maxLon === minLon ? 0.5 : (point.longitude - minLon) / (maxLon - minLon)) * 256,
    y: 18 + (maxLat === minLat ? 0.5 : (maxLat - point.latitude) / (maxLat - minLat)) * 124
  }));
  const line = xy.map((point) => `${point.x},${point.y}`).join(' ');

  return (
    <>
      <div className={classes.routeBox}>
        <svg viewBox='0 0 300 160' role='img' aria-label={t('mapLabel', { count: points.length })}>
          <rect width='300' height='160' fill='#e8f0e5' />
          {[30, 70, 110].map((y) => (
            <path key={y} d={`M0 ${y} Q120 ${y - 12} 300 ${y + 5}`} stroke='#d0dfcf' strokeWidth='2' fill='none' />
          ))}
          {[55, 115, 175, 235].map((x) => (
            <path key={x} d={`M${x} 0 L${x - 15} 160`} stroke='#d0dfcf' strokeWidth='2' />
          ))}
          <polyline
            points={line}
            fill='none'
            stroke='#ffffff'
            strokeWidth='9'
            strokeLinecap='round'
            strokeLinejoin='round'
          />
          <polyline
            points={line}
            fill='none'
            stroke='#1e6a58'
            strokeWidth='4'
            strokeLinecap='round'
            strokeLinejoin='round'
          />
          {xy.map((point, index) => (
            <g key={point.id}>
              <circle cx={point.x} cy={point.y} r='7' fill='#fff' stroke='#1e6a58' strokeWidth='3' />
              <text x={Math.min(255, point.x + 9)} y={Math.max(12, point.y - 9)} fontSize='9' fill='#20312c'>
                {index + 1}
              </text>
            </g>
          ))}
        </svg>
      </div>
      <p className={shared.muted}>{t('mapNote')}</p>
    </>
  );
}
