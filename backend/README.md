# Saku no Ie API (backend)

Go API (Gin + Huma v2) for the Saku no Ie guest-registration system. Prisma 7 owns the MariaDB
schema and migrations (there is no Prisma Client). The Go code uses GORM for CRUD only, with no
AutoMigrate. The implementation contract is the repository-root `CLAUDE.md`, and its §13 records
the decisions.

## Prerequisites

- Go 1.26
- Node.js 24 and npm (Prisma CLI only)
- Docker with Compose v2 (throwaway MariaDB for tests, E2E and migration authoring)
- `mysql` client (used by the scripts)
- Firebase CLI and Java for the Auth Emulator (admin endpoints in dev and test)

```bash
npm ci
go build ./...
```

## Databases: what may run where

| Server | Databases | Allowed |
|---|---|---|
| Shared dev server `192.168.3.100:23306` (many unrelated projects; user `dev` has global privileges) | **only `lodging-master-dev`** | `npm run db:deploy:dev` (`CREATE DATABASE IF NOT EXISTS` + `prisma migrate deploy` + seed), `npx prisma migrate status`, the dev server, `cmd/admin-grant` |
| Compose MariaDB `127.0.0.1:33306` (`compose.test.yaml`, tmpfs, root password `test`) | `lodging_test`, `lodging_e2e`, `lodging_migrate`, `lodging_shadow` | everything, including `migrate dev`/`diff`/`reset`, drops and truncates |

These rules are enforced by guards:

- `prisma.config.ts` refuses `migrate dev|diff|reset` and `db push` unless `DATABASE_URL` and
  `SHADOW_DATABASE_URL` both point at the compose DB.
- `scripts/assert-safe-db.sh`, which every DB script sources, allows destructive operations only on
  compose and additive ones only on compose or exactly `lodging-master-dev`.
- `internal/testutil` panics unless `TEST_DATABASE_URL` is compose `lodging_test`/`lodging_e2e`.
- `cmd/admin-grant` applies the same target rules.

Scripts pass DB passwords through `--defaults-extra-file` or the environment, never on a command line.

## Environment

**`cmd/server` does not load `.env`.** It reads only the process environment, so export the
variables or set them inline. `.env` is read only by Prisma (`prisma.config.ts` uses dotenv) and by
`scripts/db-deploy-dev.sh`. To run the server with your `.env`:

```bash
set -a; . ./.env; set +a      # exports into this shell only; nothing goes on a command line
go run ./cmd/server
```

`.env.example` lists every variable with placeholders. `.env.test.example` lists the committed
throwaway compose URLs.

| Variable | Required | Purpose |
|---|---|---|
| `APP_ENV` | yes, no default | `development`, `test` or `production`. Anything else makes the server refuse to start. `/docs`, `/openapi.*` and `/schemas/*` are served only outside production. |
| `GORM_DATABASE_URL` | yes | go-sql-driver DSN. `loc` and `time_zone` are forced to UTC in code (see below). |
| `DATABASE_URL` | Prisma | Prisma URL. In `.env` it is the shared `lodging-master-dev` (deploy, status and seed only). |
| `SHADOW_DATABASE_URL` | Prisma, optional | compose `lodging_shadow`, set inline for `migrate dev`/`diff` only |
| `TEST_DATABASE_URL` | integration tests | compose `lodging_test` (or `lodging_e2e`). Without it, the integration tests skip. |
| `PORT` | no (8080) | listen port |
| `TRUSTED_PROXIES` | no (`127.0.0.1,::1`) | peers whose `X-Real-IP` is trusted as the client IP. List only the hop that sits between the edge and Go (see [Client IP and the edge proxy](#client-ip-and-the-edge-proxy)). `X-Forwarded-For` is never read. |
| `PASSPORT_ENC_KEYS` | yes | `1:<base64 32 bytes>[,2:<…>]`, AES-256-GCM keys by ID |
| `PASSPORT_ENC_ACTIVE_KEY_ID` | yes | the key ID used for new encryptions |
| `IDEMPOTENCY_HMAC_KEY` | yes | base64 of 32 bytes (HMAC of create request bodies) |
| `FIREBASE_PROJECT_ID` | yes | ID-token audience, e.g. `demo-sakunoie` with the emulator |
| `FIREBASE_AUTH_EMULATOR_HOST` | dev/test only | e.g. `127.0.0.1:9099`. It disables signature checks, so the server refuses it with `APP_ENV=production` or a project ID that does not start with `demo-`. |
| `RATE_LIMIT_CREATE_PER_MIN`, `RATE_LIMIT_MANAGE_PER_MIN`, `RATE_LIMIT_SURVEY_PER_MIN` | no (10/60/10) | per-IP limits. The E2E stack uses 1000. |
| `SURVEY_MIN_RESPONSES` | no (5) | small-cell threshold of the survey summary |

### Client IP and the edge proxy

The per-IP rate limiter keys on `c.ClientIP()`. Go reads **only `X-Real-IP`**, and only when the TCP
peer is in `TRUSTED_PROXIES`; `X-Forwarded-For` is ignored. Reason: the Next rewrite
(`/api/:path*` → `API_PROXY_TARGET`) forwards the client's `X-Forwarded-For` unchanged and does not
append the peer address, so behind it XFF is entirely client-controlled. With no trusted header,
every request through Next arrives from `127.0.0.1` and all clients share one bucket.

Production deployment requirements:

- The edge (nginx, Caddy or the load balancer) **must overwrite `X-Real-IP` with the peer address
  and drop any client-supplied value**, e.g. nginx `proxy_set_header X-Real-IP $remote_addr;`, Caddy
  `header_up X-Real-IP {remote_host}`. Next passes the header through to Go unchanged.
- Alternatively, route `/api/` from the edge straight to Go (bypassing Next) with the same header.
- `TRUSTED_PROXIES` must list **only** the hop that connects to Go (the Next server, or the edge when
  `/api/` bypasses Next). Never list a range that clients can reach directly.
- HSTS is set at the edge (TLS terminates there), not by Go. Go sets
  `X-Content-Type-Options: nosniff` on every response (`middlewares.SecurityHeaders`).
- The limiter keeps at most 100,000 buckets in memory; past that it drops expired and then
  arbitrary buckets down to 90% and logs one WARN.

### Generating keys

```bash
openssl rand -base64 32     # one key; e.g. PASSPORT_ENC_KEYS="1:<output>", IDEMPOTENCY_HMAC_KEY="<output>"
```

Keep production keys in a secret store, never in Git. Each key must decode to exactly 32 bytes.
To rotate the passport key, add `2:<new>` and set `PASSPORT_ENC_ACTIVE_KEY_ID=2`. Keep `1:` until a
re-encryption job exists (follow-up), because old rows still need it.

### Expected startup WARN

```
{"level":"WARN","msg":"db: loc overridden to UTC per contract §2"}
```

This is expected. The DSN in `.env` keeps `loc=Local` (and the compose DSNs have no `time_zone`),
but `internal/database` forces `loc=UTC` and session `time_zone='+00:00'`, so `DATETIME(6)` always
holds UTC. The WARN appears whenever the DSN lacks `loc=UTC` or `time_zone='+00:00'`. It never
contains the DSN.

## Running

```bash
set -a; . ./.env; set +a
go run ./cmd/server        # :8080; GET /health (liveness), GET /ready (DB ping)
```

For the admin endpoints in dev, start the Auth Emulator from `../frontend`:

```bash
(cd ../frontend && firebase emulators:start --only auth --project demo-sakunoie)   # :9099
```

## Database workflow

### Shared dev DB (`lodging-master-dev`)

```bash
npm run db:deploy:dev        # reads .env; asserts the target is exactly lodging-master-dev
npx prisma migrate status    # "Database schema is up to date!"
```

### Compose test DB

```bash
docker compose -f compose.test.yaml up -d --wait          # MariaDB 11.1.2 on 127.0.0.1:33306 (tmpfs)
bash scripts/db-reset-test.sh lodging_test                # drop/create + migrate deploy + seed
# ... tests ...
docker compose -f compose.test.yaml down -v
```

`TEST_DB_PORT` changes the host port. The compose file and every guard read it.

### Authoring a migration (compose only)

```bash
docker compose -f compose.test.yaml up -d --wait
export DATABASE_URL=mysql://root:test@127.0.0.1:33306/lodging_migrate
export SHADOW_DATABASE_URL=mysql://root:test@127.0.0.1:33306/lodging_shadow
# migrate dev needs a TTY; in a non-interactive shell wrap it with `script`:
bash -c '(sleep 8; printf "y\n"; sleep 15) | script -qec "npx prisma migrate dev --name <name>" /dev/null'
npx prisma migrate diff --from-migrations prisma/migrations --to-schema prisma/schema.prisma --exit-code
```

CHECK constraints live in the hand-written `*_add_check_constraints` migration, because Prisma cannot
express them. That migration is the only sanctioned hand edit. The facility seed is `prisma/seed.sql`
(`npm run db:seed`). It is insert-only and idempotent.

## Admin access (Firebase + `cmd/admin-grant`)

The admin API (`/api/v1/admin/*`) verifies Firebase ID tokens with firebase-admin-go
`VerifyIDToken` and authorizes every request against `admin_users` + `admin_facility_access`.
The MVP has only the `owner` role.

- **No credentials are needed.** The app uses `option.WithoutAuthentication()`, and verification uses
  Google's public certificates, so production needs only `FIREBASE_PROJECT_ID` (Phase 4 spike in
  `CLAUDE.md` §13).
- There is no auto-provisioning. A Firebase user can use the console only after an owner grant:

```bash
# Compose (tests / E2E): the DSN goes in the environment.
GORM_DATABASE_URL='root:test@tcp(127.0.0.1:33306)/lodging_e2e?parseTime=True' \
  go run ./cmd/admin-grant --uid <firebase uid> --facility saku-no-ie
```

### Granting yourself on `lodging-master-dev`

1. Start the Auth Emulator (above) and the frontend with
   `NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099`.
2. Create your user, either through the emulator REST API or by signing in once on `/admin/login`
   after creating the account:
   ```bash
   curl -s -XPOST 'http://127.0.0.1:9099/identitytoolkit.googleapis.com/v1/accounts:signUp?key=demo' \
     -H 'content-type: application/json' --data @signup.json   # {"email":"…","password":"…"} in a file, not argv
   ```
   The response's `localId` is the uid. Delete `signup.json` afterwards.
3. Grant, with the `.env` DSN exported rather than typed:
   ```bash
   set -a; . ./.env; set +a
   go run ./cmd/admin-grant --uid <localId> --facility saku-no-ie
   ```
   It refuses every target except compose `lodging_test`/`lodging_e2e` and
   `192.168.3.100:23306/lodging-master-dev`, and it never prints the DSN. It is idempotent.

Emulator accounts disappear when the emulator stops unless you run it with
`--import <dir> --export-on-exit`. A new account has a new uid, so you must grant it again. With a
real Firebase project, the uid is shown in the Firebase console under Authentication → Users.

## Testing

```bash
go vet ./...
APP_ENV=test go test ./...                 # unit tests; DB and emulator tests skip with a message
```

Full run. The compose DB and the Auth Emulator must be up. Use `-p 1`, because DB tests truncate
tables:

```bash
docker compose -f compose.test.yaml up -d --wait && bash scripts/db-reset-test.sh lodging_test
(cd ../frontend && firebase emulators:start --only auth --project demo-sakunoie) &
APP_ENV=test FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099 FIREBASE_PROJECT_ID=demo-sakunoie \
TEST_DATABASE_URL='root:test@tcp(127.0.0.1:33306)/lodging_test?parseTime=True' \
  go test -p 1 -count=1 -race ./...
```

The only allowed skip is the forged-signature test, and only when Google's certificate endpoint is
unreachable (offline).

## OpenAPI

`api/openapi.yaml` (OpenAPI 3.1) is generated from the Huma operations and committed:

```bash
go run ./cmd/openapi && git diff --exit-code api/openapi.yaml   # api/openapi_test.go checks drift in go test
npx --yes @redocly/cli lint api/openapi.yaml                     # one-off lint (only the info-license warning remains)
```

Errors use one envelope, `{code, message, details[{location, message}], request_id}`. It never
echoes input values.

## Project structure

```
api/                 openapi.yaml + drift test
cmd/server           API server
cmd/openapi          spec exporter
cmd/admin-grant      owner grant (guarded targets)
internal/config      env loading and startup refusals
internal/database    GORM open, UTC enforcement
internal/entities    GORM table structs
internal/models      Huma request/response DTOs, error envelope
internal/handlers    parse → service → map
internal/routes      operation registration (/api/v1)
internal/middlewares request ID, logging, rate limit, edit-token and admin auth
internal/services    business logic and transactions
internal/testutil    compose-only DB helpers for integration tests
internal/utils       crypto, CSV safety, masking, local time
prisma/              schema, migrations, seed.sql
scripts/             assert-safe-db.sh, db-reset-test.sh, db-deploy-dev.sh
```
