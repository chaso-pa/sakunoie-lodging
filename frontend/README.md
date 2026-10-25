# sakunoie-lodging-frontend

Frontend for the 佐久の家 service pages (lodging information and the guest community), following [Bulletproof React](https://github.com/alan2207/bulletproof-react) architecture conventions.

## Stack

- **Next.js** (App Router) — framework
- **Mantine v9** — UI component library
- **SCSS Modules** + PostCSS — styling
- **Leaflet** — map of the place picker (OpenStreetMap tiles by default, configurable)
- **Maps JavaScript API** via `@googlemaps/js-api-loader` — the place picker's map instead, when a browser key is set
- **TanStack Query + axios + zod** — API layer (no direct DB access; data comes from the Go API)
- **next-intl** — UI strings in ja / en / ko / zh (`messages/*.json`), locale stored in a cookie
- **Biome** — linting & formatting
- **TypeScript** (strict mode)

## Getting Started

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

`/api/*` is proxied to the Go API by the `rewrites` in `next.config.ts`. The target defaults to
`http://localhost:8080`; set `API_PROXY_TARGET` to change it (see `.env.example`). The value is read when
the dev server starts and is fixed at build time for production builds.

### Client address headers

Next.js does not tell the API who the client is: its proxy adds no `X-Forwarded-For`, and it passes a
client-supplied one through as is. `src/proxy.ts` therefore deletes `X-Forwarded-For` and `X-Real-IP` from
every `/api/*` request, so they cannot be forged through this app. The API sees all requests as coming
from the Next.js server's address.

If a reverse proxy you control runs in front of Next.js and overwrites these headers itself, start the
server with `TRUST_FORWARDED_HEADERS=true` to pass them through unchanged. Never enable it when Next.js is
reachable directly. It is read at server start, not at build time.

Because `src/proxy.ts` handles `/api/*`, Next.js buffers those request bodies. The buffer is set to 12MB
(`experimental.proxyClientMaxBodySize` in `next.config.ts`), above the API's 10MB upload cap, so that the
API is the one that rejects an oversized upload with a proper error. A body over 12MB is answered with a
plain 500 by Next.js and does not reach the API.

A post carries up to five photos in one request, so that 10MB is the limit for all of them together; the
API also caps each photo at 4MB (see "投稿の写真" in `backend/README.md`). The composer downscales every
photo to a JPEG of at most 1600px before it is added (`src/utils/downscale-image.ts`), which keeps a photo
at a few hundred kilobytes, and checks the same three limits itself: five photos (`postPhotoLimit`), 4MB
each (`postPhotoMaxBytes`) and 9.5MB in total (`uploadMaxBytes`, leaving room for the other form fields).
`getPostPhotoProblem` in `src/lib/api-client.ts` turns the API's refusals into the composer's messages.

### Place picker map

The post composer's place picker draws its map with [Leaflet](https://leafletjs.com) and raster tiles. With
no configuration it uses the public openstreetmap.org tile server, which needs no key or account. Set
`NEXT_PUBLIC_MAP_TILE_URL`, `NEXT_PUBLIC_MAP_TILE_ATTRIBUTION` and `NEXT_PUBLIC_MAP_TILE_MAX_ZOOM` (see
`.env.example`) to use another tile source; they are fixed at build time. The tile policy, the required
attribution and how the picker talks to the API are described in `../docs/spot-search.md`.

#### Google map

Set `NEXT_PUBLIC_GOOGLE_MAPS_API_KEY` (a **browser** key) and `NEXT_PUBLIC_GOOGLE_MAPS_MAP_ID` to draw the
picker's map with the Maps JavaScript API instead. Both are inlined into the browser bundle at build time,
so rebuild after changing them. This is the map to use when the API runs with `PLACE_PROVIDER=google`:
Google's terms allow Google's results only on a Google map, so on the Leaflet map they stay in the list.

- Restrict the key in the Cloud console to the **Maps JavaScript API** only and to **HTTP referrers**: the
  production origin and every origin the app is opened from in development, e.g. `http://localhost:3000/*`,
  `http://192.168.3.101:3000/*` and `http://localhost:3107/*`. A page whose origin is not in the list gets
  `RefererNotAllowedMapError`, and the picker falls back to the Leaflet map.
- The Map ID is created under "Map management" (JavaScript, raster or vector). The picker's markers are
  Advanced Markers, which need one; without it Google's `DEMO_MAP_ID` is used, which is only for trying out.
- Never put the server key (`GOOGLE_MAPS_API_KEY` in `backend/.env`) here. The browser key is read only in
  `src/features/post/utils/load-google-maps.ts`, so it ends up only in the chunk that loads the map.
- The API is loaded when the member opens the map, not with the composer. Each opening creates one map,
  which Google bills as one Dynamic Maps load (10,000 a month free, then $7 per 1,000, as of 2026-10-14).
- If the key is refused (`gm_authFailure`), the script cannot load (offline, blocked by an extension) or it
  does not answer within 15 seconds, the picker switches to the Leaflet map for the rest of the page's life
  and says so. Search, near-me and posting never depend on the map.
- The map's labels follow the display language of the first opening; the Maps JavaScript API cannot be
  loaded twice in one page, so after switching languages they change on the next page load.

## Project Structure

```
src/
  app/            # Next.js App Router pages and layouts
  assets/         # Static files (images, fonts, icons)
  components/     # Shared UI components (used across features)
  config/
    paths.ts      # Centralized route path strings
  features/       # Feature-based modules (primary application code)
  hooks/          # Shared React hooks
  lib/            # Pre-configured library instances (e.g., api-client)
  stores/         # Global Zustand state stores
  testing/        # Test utilities and helpers
  types/          # Shared TypeScript types
  utils/          # Shared utility functions
```

### Feature Module

Each feature under `src/features/<name>/` is self-contained:

```
src/features/<name>/
  api/            # API calls + TanStack Query hooks
  components/     # Components scoped to this feature
  hooks/          # Hooks scoped to this feature
  stores/         # Zustand stores scoped to this feature
  types/          # Types scoped to this feature
  utils/          # Utils scoped to this feature
  index.ts        # Public barrel export
```

## Architecture Principles

### Unidirectional Dependency Flow

```
shared (components, hooks, utils, lib, stores, types, config)
    ↓
features
    ↓
app
```

- Features can import from the shared layer
- The app layer can import from features and shared
- **No cross-feature imports** — features are isolated from each other

### File Naming

- React components: `PascalCase.tsx`
- All other TypeScript files: `kebab-case.ts`
- Folders: `kebab-case`

### Imports

Always use `@/` alias. Relative imports are not allowed.

```ts
import { UserCard } from '@/components/UserCard';
import { paths } from '@/config/paths';
import { useAuth } from '@/features/auth';
```

### Route Paths

All paths are defined in `src/config/paths.ts`:

```ts
import { paths } from '@/config/paths';
href={paths.home.getHref()}
```

## Scripts

```bash
npm run dev        # Start dev server (Turbopack)
npm run build      # Production build
npm run lint       # Biome lint check
npm run format     # Biome format (write)
```

## Reference

- [Bulletproof React](https://github.com/alan2207/bulletproof-react)
- [Mantine](https://mantine.dev)
- [Next.js App Router](https://nextjs.org/docs/app)
- [next-intl](https://next-intl.dev)
- [Biome](https://biomejs.dev)
