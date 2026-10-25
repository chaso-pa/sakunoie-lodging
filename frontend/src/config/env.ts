/**
 * Public environment variables. Only NEXT_PUBLIC_* values belong here: they are inlined into the browser bundle.
 */
export const env = {
  apiBaseUrl: (process.env.NEXT_PUBLIC_API_BASE_URL ?? '').replace(/\/+$/, ''),
  /**
   * Raster tiles of the place picker's map (see docs/spot-search.md). The default is the public
   * openstreetmap.org tile server, which needs no key but has a usage policy; a deployment with real
   * traffic should point these at a commercial or self-hosted source.
   */
  mapTileUrl: process.env.NEXT_PUBLIC_MAP_TILE_URL || 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
  /** Shown on the map. HTML, written by whoever deploys the app; the tile source's licence decides its wording. */
  mapTileAttribution:
    process.env.NEXT_PUBLIC_MAP_TILE_ATTRIBUTION ||
    '&copy; <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">OpenStreetMap</a> contributors',
  mapTileMaxZoom: Number(process.env.NEXT_PUBLIC_MAP_TILE_MAX_ZOOM) || 19,
  /**
   * Whether the place picker draws a Google map instead of the tiles above. Only this flag lives here: the
   * browser key itself is read in `features/post/utils/load-google-maps.ts`, so that it is inlined into the
   * map's own chunk and not into every page that imports this file.
   */
  googleMapsEnabled: Boolean(process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY)
} as const;
