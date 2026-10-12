import Link from 'next/link';
import classes from '@/components/Community.module.scss';
import { SampleBadge } from '@/components/SampleBadge';
import type { Spot } from '@/types/community';

type SpotRowProps = {
  spot: Spot;
  href: string;
  /** Second line, e.g. "East · Nature". */
  meta?: string;
};

export function SpotRow({ spot, href, meta }: SpotRowProps) {
  return (
    <Link className={classes.spotRow} href={href}>
      <div className={classes.spotEmoji} aria-hidden>
        {spot.emoji}
      </div>
      <div className={classes.grow}>
        <strong>{spot.name}</strong> {spot.is_sample && <SampleBadge />}
        {meta && <div className={classes.muted}>{meta}</div>}
        <div className={classes.muted}>{spot.description}</div>
      </div>
      <span aria-hidden>›</span>
    </Link>
  );
}
