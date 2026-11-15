'use client';

import { IconExternalLink } from '@tabler/icons-react';
import { useLocale } from 'next-intl';
import { useEffect, useRef, useState } from 'react';
import { AIRBNB_HEIGHT_ACTION, buildAirbnbEmbedSrc, isAirbnbEmbedOrigin } from '@/config/airbnb';
import classes from '@/features/public-site/components/AirbnbEmbed.module.scss';

type Props = {
  listingId: string;
  /** Listing URL for the plain fallback link (facility JSON `airbnbUrl`). */
  listingUrl: string;
  labels: { frameTitle: string; open: string; newTab: string };
};

const INITIAL_HEIGHT = 300;
const MIN_HEIGHT = 200;
const MAX_HEIGHT = 1200;

/**
 * The Airbnb listing widget as a plain iframe, **without** Airbnb's JS SDK (no third-party script on
 * our origin; root CLAUDE.md §13). It mirrors what the SDK does: the same iframe URL, and a resize on
 * `airbnb-update-height` messages, accepted only from the Airbnb origin and from this frame's window.
 * The CSP allows the frame on the home page only. A plain link is always shown as the fallback.
 */
export function AirbnbEmbed({ listingId, listingUrl, labels }: Props) {
  const locale = useLocale();
  const frameRef = useRef<HTMLIFrameElement>(null);
  const [src, setSrc] = useState<string | null>(null);
  const [height, setHeight] = useState(INITIAL_HEIGHT);

  useEffect(() => {
    // Like the SDK's `externalPageUrl`, minus the query and fragment.
    setSrc(buildAirbnbEmbedSrc({ listingId, locale, pageUrl: `${window.location.origin}${window.location.pathname}` }));

    const onMessage = (event: MessageEvent) => {
      if (!isAirbnbEmbedOrigin(event.origin)) return;
      if (!frameRef.current || event.source !== frameRef.current.contentWindow) return;
      const data: unknown = event.data;
      if (typeof data !== 'object' || data === null) return;
      const { action, height: next } = data as { action?: unknown; height?: unknown };
      if (action !== AIRBNB_HEIGHT_ACTION || typeof next !== 'number' || !Number.isFinite(next)) return;
      setHeight(Math.min(MAX_HEIGHT, Math.max(MIN_HEIGHT, Math.round(next))));
    };
    window.addEventListener('message', onMessage);
    return () => window.removeEventListener('message', onMessage);
  }, [listingId, locale]);

  return (
    <div className={classes.wrap}>
      <div className={classes.frame} style={{ height }}>
        {src ? (
          <iframe
            ref={frameRef}
            src={src}
            title={labels.frameTitle}
            loading='lazy'
            referrerPolicy='strict-origin-when-cross-origin'
            // The widget runs its own scripts and opens Airbnb in a new tab; it never sees our origin.
            sandbox='allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox'
            className={classes.iframe}
          />
        ) : null}
      </div>
      <a href={listingUrl} target='_blank' rel='noopener noreferrer' className={classes.link}>
        {labels.open}
        <IconExternalLink size={16} aria-hidden />
        <span className='visually-hidden'>{labels.newTab}</span>
      </a>
    </div>
  );
}
