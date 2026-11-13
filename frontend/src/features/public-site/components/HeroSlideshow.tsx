'use client';

import { ActionIcon, Badge, UnstyledButton } from '@mantine/core';
import { useReducedMotion } from '@mantine/hooks';
import { IconPlayerPauseFilled, IconPlayerPlayFilled } from '@tabler/icons-react';
import Image from 'next/image';
import { useEffect, useState } from 'react';
import classes from '@/features/public-site/components/PhotoHero.module.scss';

export type HeroSlide = { src: string; alt: string; focalPoint: string; placeholder: boolean };

type Labels = { region: string; roleDescription: string; pause: string; play: string; goTo: string[]; badge: string };

const INTERVAL_MS = 6000;

/**
 * Cross-fading hero photos. Auto-advances every 6 s with a pause control (WCAG 2.2.2),
 * pauses while hovered or focused, and never auto-advances under prefers-reduced-motion.
 */
export function HeroSlideshow({ slides, labels }: { slides: HeroSlide[]; labels: Labels }) {
  const reducedMotion = useReducedMotion();
  const [index, setIndex] = useState(0);
  const [playing, setPlaying] = useState(true);
  const [hovered, setHovered] = useState(false);
  const autoplay = playing && !hovered && !reducedMotion && slides.length > 1;

  // biome-ignore lint/correctness/useExhaustiveDependencies: `index` restarts the 6 s timer after any change, including a dot click
  useEffect(() => {
    if (!autoplay) return;
    const timer = window.setTimeout(() => setIndex((i) => (i + 1) % slides.length), INTERVAL_MS);
    return () => window.clearTimeout(timer);
  }, [autoplay, index, slides.length]);

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
      {slides.map((s, i) => (
        <div key={s.src} className={classes.slide} data-active={i === index || undefined} aria-hidden={i !== index}>
          <Image
            src={s.src}
            alt={s.alt}
            fill
            priority={i === 0}
            unoptimized
            sizes='100vw'
            style={{ objectFit: 'cover', objectPosition: s.focalPoint }}
          />
        </div>
      ))}
      <div className={classes.scrim} aria-hidden />
      {current.placeholder ? (
        <Badge className={classes.badge} color='dark' variant='filled' radius='sm' style={{ textTransform: 'none' }}>
          {labels.badge}
        </Badge>
      ) : null}
      {slides.length > 1 ? (
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
            {playing ? <IconPlayerPauseFilled size={18} aria-hidden /> : <IconPlayerPlayFilled size={18} aria-hidden />}
          </ActionIcon>
        </div>
      ) : null}
    </section>
  );
}
