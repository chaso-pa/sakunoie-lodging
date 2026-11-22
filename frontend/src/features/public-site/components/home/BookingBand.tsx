import { Container } from '@mantine/core';
import type { ReactNode } from 'react';
import classes from '@/features/public-site/components/home/BookingBand.module.scss';

type Props = {
  airbnb: { title: string; body: string };
  /** The client-side Airbnb listing iframe (`AirbnbEmbed`). */
  embed: ReactNode;
  /** The price/check-in card with the primary Airbnb CTA (`BookingCard`). */
  booking: ReactNode;
};

/**
 * The conversion band: social proof (the Airbnb listing with its reviews) beside the price, check-in
 * and the one primary booking CTA. Mobile: listing first, then the booking card as the closing block.
 */
export function BookingBand({ airbnb, embed, booking }: Props) {
  return (
    <div className={classes.band}>
      <Container size={1200} className={classes.grid}>
        <section aria-labelledby='airbnb-title' className={classes.reviews}>
          <div className={classes.heading}>
            <h2 id='airbnb-title' className={classes.title}>
              {airbnb.title}
            </h2>
            <p className={classes.body}>{airbnb.body}</p>
          </div>
          {embed}
        </section>
        <div className={classes.booking}>{booking}</div>
      </Container>
    </div>
  );
}
