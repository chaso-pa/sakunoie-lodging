# Saku no Ie web (frontend)

Next.js app for the Saku no Ie website: the public site in four languages, guest registration
(wizard, completion page and edit link), the anonymous post-stay survey, and the Japanese-only admin
console. All data goes through the Go API in `../backend` and never directly to a database.

The implementation contract is the repository-root `CLAUDE.md`. Its §13 decisions log explains the
choices summarized here.

## Stack

- **Next.js 16** (App Router, Turbopack, `output: 'standalone'`) and React 19.
- **Mantine v9**, SCSS Modules and PostCSS.
- **next-intl 4.14.9** (pinned exactly). Locales are `ja` (default), `en`, `zh-CN` and `ko`, and every
  public route has a locale prefix (`/ja/...`).
- **TanStack Query** and one shared axios instance (`src/lib/api-client.ts`, base URL
  `${NEXT_PUBLIC_API_BASE_URL}/api/v1`, or `/api/v1` when that is unset), with `@mantine/form` and Zod
  for forms.
- **Firebase Auth JS SDK** (admin sign-in only; the Auth Emulator is used in dev and E2E).
- **Biome**, **TypeScript** (strict), **Vitest** for unit tests and **Playwright** for E2E tests.

The architecture follows Bulletproof React (`src/features/*`, no cross-feature imports, `@/` alias,
route strings in `src/config/paths.ts`). See `CLAUDE.md` in this directory.

## Prerequisites

- Node.js 24 and npm. The lockfile is `package-lock.json`.
- The Go API running somewhere reachable (see `../backend/README.md`).
- For the admin console in dev and for E2E: the Firebase CLI (`firebase`) and Java, which the Auth
  Emulator needs.
- For E2E: Docker with Compose v2, Go and the `mysql` client (used by the backend scripts).

```bash
npm ci
```

## Environment variables

The two groups of variables are read at different times, so it matters when you set them.

| Variable | Read at | Purpose |
|---|---|---|
| `NEXT_PUBLIC_API_BASE_URL` | **build time** (inlined; also read by `src/proxy.ts` and `next.config.ts`) | Origin of the Go API that the browser calls **directly** (cross-origin), e.g. `https://api.sakunoie.example`. The API client uses `${NEXT_PUBLIC_API_BASE_URL}/api/v1`, every page's CSP `connect-src` adds this origin, and the rewrite below is disabled. Origin only (`scheme://host[:port]`); anything else fails the build. The API must list this site's origin in `CORS_ALLOWED_ORIGINS`. **Required for production**; rebuild to change it. |
| `API_PROXY_TARGET` | **build time** (`next build`; also `next dev` start) | **Dev convenience only**, used when `NEXT_PUBLIC_API_BASE_URL` is unset: `next.config.ts` rewrites `/api/:path*` to `${API_PROXY_TARGET}/api/:path*`, so the browser stays same-origin and the API needs no CORS. The default is `http://127.0.0.1:8080`. `next build` writes the value into the routes manifest, so you must rebuild to change it. Production does not use it. |
| `NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST` | **build time** (inlined), also read by `src/proxy.ts` | `host:port` of the Auth Emulator, e.g. `127.0.0.1:9099`. When it is set, the admin client calls `connectAuthEmulator`, the API key may be any string, the project defaults to `demo-sakunoie`, and the admin CSP `connect-src` allows `http://<host>`. **Never set it for production.** |
| `NEXT_PUBLIC_FIREBASE_API_KEY` | build time (inlined) | Firebase web API key. Required when there is no emulator. |
| `NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN` | build time (inlined) | e.g. `<project>.firebaseapp.com`. Required when there is no emulator. |
| `NEXT_PUBLIC_FIREBASE_PROJECT_ID` | build time (inlined) | Must equal the backend `FIREBASE_PROJECT_ID`. Required when there is no emulator; with the emulator it defaults to `demo-sakunoie`. |
| `PORT`, `HOSTNAME` | runtime (standalone server) | Listen address of `node .next/standalone/server.js`. |

Without the emulator, admin pages put the Google endpoints (`identitytoolkit.googleapis.com`,
`securetoken.googleapis.com`) in `connect-src`. Every page's `connect-src` is `'self'` plus the
`NEXT_PUBLIC_API_BASE_URL` origin when it is set; the Firebase origins appear on admin pages only.

`.env.example` lists every variable with placeholders. Other `.env*` files are git-ignored: set the
variables in your shell or in an untracked `.env.local`.

E2E-only variables are listed under [E2E tests](#e2e-tests-playwright).

## Security headers

`src/proxy.ts` (the Next 16 replacement for middleware) runs for every page. It:

- builds a per-request nonce CSP with `src/lib/security-headers.ts`: `script-src 'self' 'nonce-…' 'strict-dynamic'`
  (`'unsafe-eval'` in `next dev` only), `connect-src 'self'` plus the API origin (and the Firebase
  origins on `/admin/*`),
  `frame-ancestors 'none'`, `object-src 'none'` and `base-uri 'none'`;
- puts the CSP on the forwarded request headers, so Next nonces its inline scripts, and on the response;
- sets `X-Content-Type-Options: nosniff`, `Referrer-Policy`, and `X-Robots-Tag: noindex, nofollow` on
  registration, stay and admin pages. Their `Referrer-Policy: no-referrer` also applies to their
  cross-origin API calls, so those carry no `Referer`. The edit token travels only in the URL
  fragment and the `Authorization: Bearer` header, never in a request URL.

Because of the nonce, every page is rendered dynamically.

## Content (`content/`)

Public copy, facility data and the survey questions live in Git as Markdown and JSON:

```
content/
  site.json                    # defaultFacilityCode + facility list (survey facility source)
  facilities/<code>.json       # names per locale, area, airbnbUrl, amenities, photos (alt per locale)
  surveys/v1.json              # survey questions + labels in 4 locales
  {ja,en,zh-CN,ko}/
    home.md facilities.md guide.md faq.md rates.md terms.md privacy.md
```

- Each Markdown file has front matter (`title`, `description`, quoted `version` and `effectiveDate`,
  `placeholder`, `todo`, plus page-specific fields that Zod validates). It is read on the server with
  `gray-matter` and rendered with `react-markdown`, which allows no raw HTML. If a locale file is
  missing, the page falls back to `ja` with a warning, but the content test requires all 28 files.
- While `placeholder: true`, a page shows the "要差し替え / Placeholder content" banner.
- JSON is imported through the **`@content/*` alias** (`tsconfig.json` and `vitest.config.mts`), e.g.
  `import site from '@content/site.json'`. Facility JSON files are registered in `src/config/site.ts`.
- `next.config.ts` `outputFileTracingIncludes` ships `content/` with the standalone build.
- The terms and privacy `version` values must equal `../backend/internal/services/policy_versions.json`.
  A test enforces this.

### Survey definition must stay in sync

The questions exist twice:

- `../backend/internal/services/surveydef/v1.json` is embedded in the API and used to validate answers.
- `content/surveys/v1.json` holds the same structure plus the labels in four languages.

Codes, types, options, `required` and `maxLength` must be identical. `src/config/survey-definition.test.ts`
fails if they differ. Change both files in the same commit. To make an incompatible change, add a new
version (`v2.json`) on both sides instead of editing `v1`.

## Development

```bash
# API on :8080 (see ../backend/README.md), then:
npm run dev                         # next dev --turbopack on :3000
# Admin console with the Auth Emulator:
firebase emulators:start --only auth --project demo-sakunoie     # :9099
NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099 npm run dev
```

In development `NEXT_PUBLIC_API_BASE_URL` is usually unset, so the browser calls `/api/v1` on the
Next origin and the rewrite forwards it to `API_PROXY_TARGET`. To try the production (cross-origin)
path, set `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8080` and start the API with
`CORS_ALLOWED_ORIGINS=http://localhost:3000`.

If another project's `next dev` already uses :3000, use another port, e.g.
`API_PROXY_TARGET=http://127.0.0.1:8180 npx next dev --turbopack -p 3100`. `allowedDevOrigins`
permits `127.0.0.1`.

## Checks

```bash
npx tsc --noEmit     # npm run typecheck
npx biome check      # npm run lint
npx vitest run       # npm test (src/**/*.test.ts)
npx next build
```

## Production-mode run (standalone)

`output: 'standalone'` produces `.next/standalone/server.js`. `next start` does not support this mode.

```bash
NEXT_PUBLIC_API_BASE_URL=https://api.sakunoie.example \
NEXT_PUBLIC_FIREBASE_API_KEY=… NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=… NEXT_PUBLIC_FIREBASE_PROJECT_ID=… \
  npx next build
cp -a public .next/standalone/public
cp -a .next/static .next/standalone/.next/static
PORT=3000 HOSTNAME=127.0.0.1 node .next/standalone/server.js
```

In production, Next and the API are reached through Cloudflare Tunnel on two hostnames, and both bind
to `127.0.0.1` (`../backend/README.md` "Cloudflare Tunnel deployment"). There is no `/api` rewrite.

Next 16.4 may stop a build once with a `SecurityFatalError` while it checks for advisories. If it
does, run the build again. Upgrading Next is a separate decision (see the open items in the root
`CLAUDE.md`).

## E2E tests (Playwright)

The specs are in `e2e/`: `public`, `registration`, `manage`, `admin` and `survey`. Each spec seeds its
own synthetic data through the public API in `beforeAll`, using unique booking references, so any
spec passes when run alone. The config uses `workers: 1` and `fullyParallel: false`, because all
specs share one DB and one emulator. The `desktop-chromium` project (1280 px) runs every spec, and the
`mobile-chromium` project (390 px) runs the wizard spec.

The stack is started by a script rather than a Playwright `webServer`, so its PIDs are tracked:

```bash
npx playwright install chromium           # once
bash scripts/e2e-stack.sh start           # or: npm run e2e:stack -- start
npx playwright test                       # all specs; npm run e2e
npx playwright test e2e/manage.spec.ts    # one spec alone
for s in e2e/*.spec.ts; do npx playwright test "$s"; done
bash scripts/e2e-stack.sh stop            # stops only its own PIDs, then compose down -v
```

`scripts/e2e-stack.sh start`:

1. refuses to start if :3100, :8180 or :9099 is already in use (it never uses :3000 or :8080);
2. starts `../backend/compose.test.yaml` (MariaDB on `127.0.0.1:33306`) and recreates `lodging_e2e`
   with `db-reset-test.sh` (migrate deploy + seed);
3. starts the Firebase Auth Emulator on :9099 (project `demo-sakunoie`);
4. builds and starts the API on :8180 with `APP_ENV=test`, per-run random passport and HMAC keys
   (exported, never on a command line), the emulator, `RATE_LIMIT_*_PER_MIN=1000` and
   `SURVEY_MIN_RESPONSES=1`;
5. runs `next build` with `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8180` and
   `NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099`, copies `public` and `.next/static` into
   the standalone output, and runs it on :3100. The browser calls the API cross-origin
   (:3100 → :8180), so the API runs with `CORS_ALLOWED_ORIGINS=http://127.0.0.1:3100` and CORS is
   exercised by every spec;
6. records the PIDs and logs in `.e2e-stack/` (git-ignored).

`e2e/global-setup.ts` checks that the stack is reachable. It then clears the emulator accounts,
signs up `e2e-admin@example.test` and `e2e-nobody@example.test` with a random per-run password,
and grants the admin user the owner role through `go run ./cmd/admin-grant` against compose
`lodging_e2e`.

| Variable | Default | Purpose |
|---|---|---|
| `E2E_BASE_URL` | `http://127.0.0.1:3100` | Playwright `baseURL` |
| `E2E_API_URL` | `http://127.0.0.1:8180` | API origin for seeding and direct API checks (the web origin has no `/api` rewrite in this mode) |
| `GORM_DATABASE_URL_E2E` | compose `lodging_e2e` | API DSN for the stack and the admin grant; must be compose `lodging_e2e` |
| `TEST_DB_PORT` | `33306` | compose MariaDB host port |
| `E2E_STATE_DIR` | `.e2e-stack` | logs, PIDs and the API binary |
| `E2E_PID_LOG` | (unset) | extra file to which every started and stopped PID is appended |
| `E2E_SKIP_BUILD` | (unset) | `1` reuses the existing `.next` build, which must have been built with the E2E variables |

Traces and screenshots of failed tests are kept in `test-results/` and `playwright-report/`, both
git-ignored, because they may contain synthetic tokens. CI runs the same steps (`.github/workflows/ci.yml`, job `e2e`).

## Placeholders to replace before launch

- [ ] **Airbnb URL**: `content/facilities/saku-no-ie.json` `airbnbUrl` is the Airbnb top page. Set the
      listing URL and `airbnbUrlIsPlaceholder: false`.
- [ ] **Photos**: `public/images/placeholder/*.svg` are generated shapes with a "placeholder image"
      badge. Replace them with licensed photos and keep `alt` (4 locales), `aspectRatio` and `focalPoint`.
- [ ] **Survey questions**: `q01..q10` are placeholders for the existing Google Forms questions.
      Update both `content/surveys/v1.json` and `../backend/internal/services/surveydef/v1.json`.
- [ ] **Legal texts**: `content/*/terms.md` and `privacy.md` are placeholders (version `2026-11-01`).
      Get legal review, then bump `version` here and in `../backend/internal/services/policy_versions.json`.
- [ ] **Rates**: `content/*/rates.md` shows "being confirmed". Add the indicative rates. Airbnb stays
      the source of actual prices.
- [ ] **Capacity**: `max_guests = 4` for `saku-no-ie` (`../backend/prisma/seed.sql`, VERIFY). The seed is
      insert-only, so re-seeding does not change an existing DB. Update `facilities.max_guests`
      there deliberately; there is no admin screen for it yet.
- [ ] Every other `placeholder: true` / `要差し替え` page (home, facilities, guide, FAQ) and the
      amenity facts (Wi-Fi, parking, laundry, room size).
