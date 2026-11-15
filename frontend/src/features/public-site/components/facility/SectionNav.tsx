'use client';

import type { MouseEvent } from 'react';
import classes from '@/features/public-site/components/facility/SectionNav.module.scss';

type Props = { label: string; items: { id: string; title: string }[] };

/** Smooth-scrolls to the section (instant under prefers-reduced-motion) and moves focus there for keyboard users. */
function scrollToSection(event: MouseEvent<HTMLAnchorElement>, id: string) {
  const target = document.getElementById(id);
  if (!target) return;
  event.preventDefault();
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  target.scrollIntoView({ behavior: reduced ? 'auto' : 'smooth', block: 'start' });
  window.history.pushState(null, '', `#${id}`);
  // Move focus too, so the next Tab continues from the section (a programmatic focus target only).
  if (!target.hasAttribute('tabindex')) target.setAttribute('tabindex', '-1');
  target.focus({ preventScroll: true });
}

/** In-page jump links to the facility sections (anchors; 44px targets, wraps on narrow screens). */
export function SectionNav({ label, items }: Props) {
  return (
    <nav aria-label={label}>
      <ul className={classes.list}>
        {items.map((s) => (
          <li key={s.id}>
            <a href={`#${s.id}`} className={classes.link} onClick={(e) => scrollToSection(e, s.id)}>
              {s.title}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}
