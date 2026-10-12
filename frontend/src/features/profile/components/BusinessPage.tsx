'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import shared from '@/components/Community.module.scss';
import { EmptyState, ErrorState, LoadingState } from '@/components/QueryState';
import { SpotArt } from '@/components/SpotArt';
import { SpotRow } from '@/components/SpotRow';
import { paths } from '@/config/paths';
import classes from '@/features/profile/components/Profile.module.scss';
import { useSpots } from '@/hooks/use-spots';

// The lodging itself is one of the seeded spots; it is not its own recommendation.
const lodgingCategory = '宿泊';

/**
 * The 佐久の家 host page. Display only: guests have no way to edit it and the API has no endpoint that
 * would accept an edit.
 */
export function BusinessPage() {
  const t = useTranslations('community');
  const spots = useSpots();
  const recommended = (spots.data ?? []).filter((spot) => spot.category !== lodgingCategory).slice(0, 3);

  return (
    <>
      <Link className={shared.back} href={paths.community.profile.getHref()}>
        ← {t('business.back')}
      </Link>
      <div className={shared.eyebrow}>{t('business.eyebrow')}</div>
      <h1 className={shared.h1}>{t('business.name')}</h1>
      <div className={shared.card}>
        <SpotArt emoji='🏡' title={t('business.name')} />
      </div>
      <p className={shared.body}>{t('business.description')}</p>

      <div className={shared.card}>
        <div className={shared.cardBody}>
          <h2 className={shared.h2}>{t('business.infoTitle')}</h2>
          <dl className={classes.infoList}>
            <div>
              <dt>{t('business.nameLabel')}</dt>
              <dd>{t('business.name')}</dd>
            </div>
            <div>
              <dt>{t('business.categoryLabel')}</dt>
              <dd>{t('business.category')}</dd>
            </div>
            <div>
              <dt>{t('business.addressLabel')}</dt>
              <dd>{t('business.address')}</dd>
            </div>
            <div>
              <dt>{t('business.hoursLabel')}</dt>
              <dd>{t('business.hours')}</dd>
            </div>
          </dl>
        </div>
      </div>

      <h2 className={shared.h2}>{t('business.recommendTitle')}</h2>
      {spots.isPending ? (
        <LoadingState />
      ) : spots.isError ? (
        <ErrorState error={spots.error} onRetry={() => spots.refetch()} retrying={spots.isRefetching} />
      ) : recommended.length === 0 ? (
        <EmptyState>{t('map.empty')}</EmptyState>
      ) : (
        <div className={classes.recommendations}>
          {recommended.map((spot) => (
            <SpotRow key={spot.id} spot={spot} href={paths.community.spot.getHref(spot.id)} />
          ))}
        </div>
      )}
    </>
  );
}
