'use client';

import { ActionIcon, Badge, UnstyledButton } from '@mantine/core';
import { useReducedMotion } from '@mantine/hooks';
import { IconChevronLeft, IconChevronRight, IconPlayerPauseFilled, IconPlayerPlayFilled } from '@tabler/icons-react';
import Image from 'next/image';
import { type PointerEvent, useEffect, useRef, useState } from 'react';
import classes from '@/features/public-site/components/PhotoHero.module.scss';

export type HeroSlide = { src: string; alt: string; focalPoint: string; placeholder: boolean };

type Labels = {
  region: string;
  roleDescription: string;
  pause: string;
  play: string;
  prev: string;
  next: string;
  goTo: string[];
  badge: string;
};

const INTERVAL_MS = 6000;
/** Horizontal travel (px) that turns a touch drag into a slide change. */
const SWIPE_THRESHOLD = 48;

/**
 * Horizontally sliding hero photos. Swipe on touch screens, arrows on wider screens, dots everywhere.
 * Auto-advances every 6 s with a pause control (WCAG 2.2.2), pauses while hovered or focused,
 * and never auto-advances or animates under prefers-reduced-motion.
 */
export function HeroSlideshow({ slides, labels }: { slides: HeroSlide[]; labels: Labels }) {
  const reducedMotion = useReducedMotion();
  const [index, setIndex] = useState(0);
  const [playing, setPlaying] = useState(true);
  const [hovered, setHovered] = useState(false);
  const [dragX, setDragX] = useState(0);
  const drag = useRef<{ id: number; startX: number; startY: number; dx: number; horizontal: boolean | null } | null>(
    null
  );
  const count = slides.length;
  const autoplay = playing && !hovered && !reducedMotion && dragX === 0 && count > 1;

  // biome-ignore lint/correctness/useExhaustiveDependencies: `index` restarts the 6 s timer after any change, including a dot click or swipe
  useEffect(() => {
    if (!autoplay) return;
    const timer = window.setTimeout(() => setIndex((i) => (i + 1) % count), INTERVAL_MS);
    return () => window.clearTimeout(timer);
  }, [autoplay, index, count]);

  const go = (step: number) => setIndex((i) => (i + step + count) % count);

  // Touch only: mouse users get the arrows, and vertical scrolling stays native (touch-action: pan-y).
  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    if (e.pointerType === 'mouse' || count < 2) return;
    drag.current = { id: e.pointerId, startX: e.clientX, startY: e.clientY, dx: 0, horizontal: null };
  };
  const onPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    const d = drag.current;
    if (!d || d.id !== e.pointerId) return;
    const dx = e.clientX - d.startX;
    const dy = e.clientY - d.startY;
    if (d.horizontal === null && Math.abs(dx) + Math.abs(dy) > 8) d.horizontal = Math.abs(dx) > Math.abs(dy);
    if (d.horizontal) {
      d.dx = dx;
      setDragX(dx);
    }
  };
  const onPointerEnd = (e: PointerEvent<HTMLDivElement>) => {
    const d = drag.current;
    if (!d || d.id !== e.pointerId) return;
    // Read the distance from the ref: a fast swipe can end before the dragX state re-renders.
    if (d.horizontal && Math.abs(d.dx) > SWIPE_THRESHOLD) go(d.dx < 0 ? 1 : -1);
    drag.current = null;
    setDragX(0);
  };

  const current = slides[index];

  return (
    <section
      className={classes.slides}
      aria-roledescription={labels.roleDescription}
      aria-label={labels.region}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onFocus={() => setHovered(true)}
      onBlur={() => setHovered(false)}
    >
      <div
        className={classes.viewport}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerEnd}
        onPointerCancel={onPointerEnd}
      >
        <div
          className={classes.track}
          data-dragging={dragX !== 0 || undefined}
          style={{ transform: `translate3d(calc(${-index * 100}% + ${dragX}px), 0, 0)` }}
        >
          {slides.map((s, i) => (
            <div key={s.src} className={classes.slide} aria-hidden={i !== index}>
              <Image
                src={s.src}
                alt={s.alt}
                fill
                priority={i === 0}
                unoptimized
                draggable={false}
                sizes='100vw'
                style={{ objectFit: 'cover', objectPosition: s.focalPoint }}
              />
            </div>
          ))}
        </div>
      </div>
      <div className={classes.scrim} aria-hidden />
      {current.placeholder ? (
        <Badge className={classes.badge} color='dark' variant='filled' radius='sm' style={{ textTransform: 'none' }}>
          {labels.badge}
        </Badge>
      ) : null}
      {count > 1 ? (
        <>
          <ActionIcon
            className={classes.arrow}
            data-side='prev'
            variant='white'
            color='dark'
            radius='xl'
            size={44}
            aria-label={labels.prev}
            onClick={() => go(-1)}
          >
            <IconChevronLeft size={22} aria-hidden />
          </ActionIcon>
          <ActionIcon
            className={classes.arrow}
            data-side='next'
            variant='white'
            color='dark'
            radius='xl'
            size={44}
            aria-label={labels.next}
            onClick={() => go(1)}
          >
            <IconChevronRight size={22} aria-hidden />
          </ActionIcon>
          <div className={classes.controls}>
            {slides.map((s, i) => (
              <UnstyledButton
                key={s.src}
                className={classes.dot}
                data-active={i === index || undefined}
                aria-label={labels.goTo[i]}
                aria-current={i === index ? 'true' : undefined}
                onClick={() => setIndex(i)}
              >
                <span aria-hidden />
              </UnstyledButton>
            ))}
            <ActionIcon
              variant='subtle'
              color='white'
              size={44}
              radius='xl'
              className={classes.playPause}
              aria-label={playing ? labels.pause : labels.play}
              onClick={() => setPlaying((p) => !p)}
            >
              {playing ? (
                <IconPlayerPauseFilled size={18} aria-hidden />
              ) : (
                <IconPlayerPlayFilled size={18} aria-hidden />
              )}
            </ActionIcon>
          </div>
        </>
      ) : null}
    </section>
  );
}
