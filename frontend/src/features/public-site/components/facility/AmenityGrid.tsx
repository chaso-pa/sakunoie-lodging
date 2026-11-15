import type { FacilityAmenity } from '@/config/site';
import { AmenityChip } from '@/features/public-site/components/AmenityChip';
import classes from '@/features/public-site/components/facility/AmenityGrid.module.scss';
import type { Locale } from '@/i18n/routing';

/** Confirmed amenities not shown elsewhere on the page, as a list of icon chips. */
export function AmenityGrid({ amenities, locale }: { amenities: FacilityAmenity[]; locale: Locale }) {
  return (
    <ul className={classes.grid}>
      {amenities.map((a) => (
        <li key={a.code}>
          <AmenityChip icon={a.icon} label={a.label[locale]} />
        </li>
      ))}
    </ul>
  );
}
