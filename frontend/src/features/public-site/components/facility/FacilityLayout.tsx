import type { ReactNode } from 'react';
import classes from '@/features/public-site/components/facility/FacilityLayout.module.scss';

type Props = {
  facts: ReactNode;
  booking: ReactNode;
  children: ReactNode;
  /** Booking reminder at the end of the content; mobile only (the desktop card stays in view). */
  endCta: ReactNode;
};

/**
 * Facility page body. Mobile: key facts, the booking card, the sections, then the CTA again.
 * Desktop (≥ md): the booking card sits in a right column and sticks while the sections scroll,
 * like a listing's booking card. One booking card in the DOM, so `#booking` works at every width.
 */
export function FacilityLayout({ facts, booking, children, endCta }: Props) {
  return (
    <div className={classes.layout}>
      <div className={classes.facts}>{facts}</div>
      <div className={classes.booking}>
        <div className={classes.bookingSticky}>{booking}</div>
      </div>
      <div className={classes.main}>
        {children}
        <div className={classes.endCta}>{endCta}</div>
      </div>
    </div>
  );
}
