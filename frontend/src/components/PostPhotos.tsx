'use client';

import { Image, UnstyledButton, VisuallyHidden } from '@mantine/core';
import { useReducedMotion } from '@mantine/hooks';
import { useTranslations } from 'next-intl';
import { useRef, useState } from 'react';
import shared from '@/components/Community.module.scss';
import classes from '@/components/PostPhotos.module.scss';
import type { Media } from '@/types/community';

type PostPhotosProps = {
  /** In the order the author gave them. */
  photos: Media[];
  nickname: string;
  /** A tile: only the first photo, with a count when there are more. */
  compact?: boolean;
};

/**
 * The photos of a post. One photo is a plain picture; several are a carousel that is swiped on a phone,
 * moved with the arrow buttons or the arrow keys, and whose later photos are fetched only when the reader
 * gets near them.
 */
export function PostPhotos({ photos, nickname, compact }: PostPhotosProps) {
  const t = useTranslations('community.post');
  const total = photos.length;

  if (total === 0) {
    return null;
  }
  if (compact || total === 1) {
    return (
      <div className={shared.mediaBox} style={compact ? { height: '100%' } : undefined}>
        <Image src={photos[0].url} alt={t('photoAlt', { nickname })} loading='lazy' h='100%' fit='cover' />
        {total > 1 && (
          <span className={classes.countBadge}>
            <span aria-hidden>▣ {total}</span>
            <VisuallyHidden>{t('photoCount', { count: total })}</VisuallyHidden>
          </span>
        )}
      </div>
    );
  }
  return <PhotoCarousel photos={photos} nickname={nickname} />;
}

function PhotoCarousel({ photos, nickname }: { photos: Media[]; nickname: string }) {
  const t = useTranslations('community.post');
  const reduceMotion = useReducedMotion();
  const track = useRef<HTMLDivElement>(null);
  const total = photos.length;
  const [active, setActive] = useState(0);
  // Photos up to this index have an <img>. It starts one ahead of the first, so the next photo is ready
  // when the swipe begins, and only grows.
  const [reached, setReached] = useState(1);
  // Nothing is announced until the reader has moved.
  const [moved, setMoved] = useState(false);

  const goTo = (index: number) => {
    const element = track.current;
    if (!element) {
      return;
    }
    const target = Math.min(Math.max(index, 0), total - 1);
    setReached((current) => Math.max(current, target + 1));
    element.scrollTo({ left: target * element.clientWidth, behavior: reduceMotion ? 'auto' : 'smooth' });
  };

  // Swipes, the buttons and the keys all end up here: the scroll position is the single source of truth.
  const onScroll = () => {
    const element = track.current;
    if (!element || element.clientWidth === 0) {
      return;
    }
    const index = Math.min(Math.max(Math.round(element.scrollLeft / element.clientWidth), 0), total - 1);
    if (index !== active) {
      setActive(index);
      setMoved(true);
      setReached((current) => Math.max(current, index + 1));
    }
  };

  const onKeyDown = (event: React.KeyboardEvent) => {
    const target = { ArrowLeft: active - 1, ArrowRight: active + 1, Home: 0, End: total - 1 }[event.key];
    if (target !== undefined) {
      event.preventDefault();
      goTo(target);
    }
  };

  return (
    // biome-ignore lint/a11y/useSemanticElements: a labelled group of photos, not a form's fieldset
    <div className={classes.carousel} role='group' aria-label={t('photosLabel', { nickname, count: total })}>
      {/* biome-ignore lint/a11y/noNoninteractiveTabindex: a scrolling region has to be reachable by keyboard */}
      {/* biome-ignore lint/a11y/noStaticElementInteractions: the arrow keys scroll it one photo at a time */}
      <div className={classes.track} ref={track} tabIndex={0} onScroll={onScroll} onKeyDown={onKeyDown}>
        {photos.map((photo, index) => (
          <div className={classes.slide} key={photo.url}>
            {index <= reached && (
              <Image
                src={photo.url}
                alt={t('photoAltNumbered', { nickname, index: index + 1, total })}
                loading='lazy'
                h='100%'
                fit='cover'
              />
            )}
          </div>
        ))}
      </div>
      <UnstyledButton
        className={classes.nav}
        data-side='prev'
        aria-label={t('previousPhoto')}
        aria-disabled={active === 0}
        onClick={() => goTo(active - 1)}
      >
        ‹
      </UnstyledButton>
      <UnstyledButton
        className={classes.nav}
        data-side='next'
        aria-label={t('nextPhoto')}
        aria-disabled={active === total - 1}
        onClick={() => goTo(active + 1)}
      >
        ›
      </UnstyledButton>
      <div className={classes.counter} aria-hidden>
        {active + 1}/{total}
      </div>
      <div className={classes.dots} aria-hidden>
        {photos.map((photo, index) => (
          <span key={photo.url} className={classes.dot} data-active={index === active} />
        ))}
      </div>
      <VisuallyHidden aria-live='polite'>
        {moved ? t('photoPosition', { current: active + 1, total }) : ''}
      </VisuallyHidden>
    </div>
  );
}
