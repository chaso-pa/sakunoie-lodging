/**
 * Airbnb listing embed, rendered as a plain iframe without Airbnb's JS SDK (root CLAUDE.md §13,
 * "Airbnb listing embed"). The SDK would be a third-party script on our origin; the iframe keeps
 * Airbnb's code in its own browsing context. The CSP allows this origin in `frame-src` on the
 * home page of each locale only (`src/lib/security-headers.ts`).
 */
/**
 * Airbnb origins the embed may load from. A browser with Airbnb's regional cookies is redirected
 * from airbnb.com to airbnb.jp, so both must be allowed in `frame-src`.
 */
export const AIRBNB_EMBED_ORIGINS = ['https://www.airbnb.com', 'https://www.airbnb.jp'] as const;

/** Japanese pages use airbnb.jp directly (Japanese listing card, no regional redirect); others airbnb.com. */
export const airbnbEmbedOrigin = (locale: string) =>
  locale === 'ja' ? AIRBNB_EMBED_ORIGINS[1] : AIRBNB_EMBED_ORIGINS[0];

/** Whether a postMessage comes from one of the embed origins. */
export const isAirbnbEmbedOrigin = (origin: string) => (AIRBNB_EMBED_ORIGINS as readonly string[]).includes(origin);

/** Message the embedded page posts to resize its frame (same contract the SDK listens for). */
export const AIRBNB_HEIGHT_ACTION = 'airbnb-update-height';

/** Listing id from an Airbnb room URL such as `https://www.airbnb.jp/rooms/123?…`. */
export const airbnbListingId = (url: string): string | undefined => {
  try {
    return new URL(url).pathname.match(/^\/rooms\/(\d+)/)?.[1];
  } catch {
    return undefined;
  }
};

/**
 * The URL the SDK would give its iframe for `data-view="home"`: every `data-*` attribute except
 * width/height/view becomes a sorted query parameter, plus `externalPageUrl`. The fragment is URI-encoded
 * JSON (the embedded page parses it) carrying the frame id it echoes in its resize messages; the SDK's
 * tracking fields (`tracking_PageUrl`, `tracking_widgetName`) are left out.
 */
export const buildAirbnbEmbedSrc = ({
  listingId,
  pageUrl,
  locale,
  frameId = 1
}: {
  listingId: string;
  pageUrl: string;
  locale: string;
  frameId?: number;
}) => {
  const params = [
    ['externalPageUrl', pageUrl],
    ['hide-price', 'true'],
    ['id', listingId]
  ]
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .sort()
    .join('&');
  return `${airbnbEmbedOrigin(locale)}/embeddable/home?${params}#${encodeURIComponent(JSON.stringify({ frameId }))}`;
};
