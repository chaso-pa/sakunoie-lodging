'use client';

import { ActionIcon, Group, Text } from '@mantine/core';
import { useReducedMotion } from '@mantine/hooks';
import { IconChevronLeft, IconChevronRight } from '@tabler/icons-react';
import { type KeyboardEvent, useRef, useState } from 'react';
import type { FacilityPhoto } from '@/config/site';
import classes from '@/features/public-site/components/PhotoGallery.module.scss';
import { PlaceholderPhoto } from '@/features/public-site/components/PlaceholderPhoto';

export type GalleryPhoto = { photo: FacilityPhoto; alt: string };

/** Plain strings (server → client props); `positionTemplate` contains `{current}` and `{total}`. */
type Labels = { region: string; prev: string; next: string; positionTemplate: string };
type Props = { photos: GalleryPhoto[]; badge: string; labels: Labels };

const format = (template: string, current: number, total: number) =>
  template.replace('{current}', String(current)).replace('{total}', String(total));

/**
 * Mobile: swipeable carousel with arrows, "n / m" and keyboard arrows. Desktop: mosaic (one large
 * room photo + supporting images). Smooth scrolling is disabled under prefers-reduced-motion.
 */
export function PhotoGallery({ photos, badge, labels }: Props) {
  const track = useRef<HTMLDivElement>(null);
  const [index, setIndex] = useState(0);
  const reduceMotion = useReducedMotion();
  const total = photos.length;

  const go = (next: number) => {
    const el = track.current;
    if (!el) return;
    const i = Math.max(0, Math.min(total - 1, next));
    el.scrollTo({ left: i * (el.scrollWidth / total), behavior: reduceMotion ? 'auto' : 'smooth' });
    setIndex(i);
  };

  const onScroll = () => {
    const el = track.current;
    if (!el) return;
    setIndex(Math.round(el.scrollLeft / (el.scrollWidth / total)));
  };

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      go(index + 1);
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      go(index - 1);
    }
  };

  if (total === 0) return null;
  const [main, ...rest] = photos;

  return (
    <>
      <div className={classes.mosaic}>
        <PlaceholderPhoto
          photo={main.photo}
          alt={main.alt}
          badge={badge}
          ratio={4 / 3}
          sizes='(min-width: 768px) 66vw'
        />
        <div className={classes.mosaicSide}>
          {rest.slice(0, 3).map((p) => (
            <PlaceholderPhoto
              key={p.photo.id}
              photo={p.photo}
              alt={p.alt}
              badge={badge}
              ratio={16 / 9}
              sizes='(min-width: 768px) 33vw'
            />
          ))}
        </div>
      </div>

      <section className={classes.carousel} aria-roledescription='carousel' aria-label={labels.region}>
        {/* biome-ignore lint/a11y/noNoninteractiveTabindex: the focusable scroll track takes arrow keys */}
        {/* biome-ignore lint/a11y/noStaticElementInteractions: same; buttons below are the pointer alternative */}
        <div ref={track} className={classes.track} onScroll={onScroll} onKeyDown={onKeyDown} tabIndex={0}>
          {photos.map((p, i) => (
            <figure
              key={p.photo.id}
              className={classes.slide}
              aria-label={format(labels.positionTemplate, i + 1, total)}
            >
              <PlaceholderPhoto photo={p.photo} alt={p.alt} badge={badge} ratio={4 / 3} sizes='100vw' />
            </figure>
          ))}
        </div>
        <Group justify='space-between' mt='sm'>
          <ActionIcon
            variant='default'
            size={44}
            radius='xl'
            onClick={() => go(index - 1)}
            disabled={index === 0}
            aria-label={labels.prev}
          >
            <IconChevronLeft size={20} aria-hidden />
          </ActionIcon>
          <Text size='sm' fw={600} aria-live='polite'>
            {format(labels.positionTemplate, index + 1, total)}
          </Text>
          <ActionIcon
            variant='default'
            size={44}
            radius='xl'
            onClick={() => go(index + 1)}
            disabled={index === total - 1}
            aria-label={labels.next}
          >
            <IconChevronRight size={20} aria-hidden />
          </ActionIcon>
        </Group>
      </section>
    </>
  );
}
