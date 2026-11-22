import { Container } from '@mantine/core';
import type { ReactNode } from 'react';
import { PlaceholderMark } from '@/features/public-site/components/facility/FacilitySection';
import classes from '@/features/public-site/components/home/HomeSection.module.scss';

type Props = {
  id: string;
  /** Band colour: the page alternates canvas and surface; `forest` is the booking band. */
  tone: 'canvas' | 'surface' | 'forest' | 'muted';
  title: string;
  eyebrow?: string;
  intro?: string;
  /** Visible "仮の内容" tag beside the heading for unconfirmed content. */
  placeholderLabel?: string;
  /** Right side of the heading row, e.g. a "see details" link (desktop). */
  aside?: ReactNode;
  /** Tighter vertical padding for short sections. */
  compact?: boolean;
  /** The heading is rendered by the children (e.g. a two-column intro). */
  hideHeader?: boolean;
  children: ReactNode;
};

/** One full-width band of the home page with a real h2 (`#{id}-title`). */
export function HomeSection({
  id,
  tone,
  title,
  eyebrow,
  intro,
  placeholderLabel,
  aside,
  compact,
  hideHeader,
  children
}: Props) {
  return (
    <section
      id={id}
      aria-labelledby={`${id}-title`}
      className={classes.band}
      data-tone={tone}
      data-compact={compact || undefined}
    >
      <Container size={1200}>
        {hideHeader ? null : (
          <header className={classes.header}>
            <div className={classes.heading}>
              {eyebrow ? <p className={classes.eyebrow}>{eyebrow}</p> : null}
              <div className={classes.titleRow}>
                <h2 id={`${id}-title`} className={classes.title}>
                  {title}
                </h2>
                {placeholderLabel ? <PlaceholderMark label={placeholderLabel} /> : null}
              </div>
              {intro ? <p className={classes.intro}>{intro}</p> : null}
            </div>
            {aside ? <div className={classes.aside}>{aside}</div> : null}
          </header>
        )}
        {children}
      </Container>
    </section>
  );
}
