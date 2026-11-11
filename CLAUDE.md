# CLAUDE.md — 佐久の家 / Multi-facility Guest Registration

## 0. Purpose and source of truth
This file is the implementation contract for the Saku no Ie multilingual website and guest-registration system. Read it before writing code. The MVP supports multiple **facilities** operated by one host. Do **not** introduce `organizations`, a reservation engine, payment processing, or cancellation workflow in the MVP. Keep future migration to organizations feasible.

Status: detailed-design baseline, 2026-11-10. Decisions marked **VERIFY** require operator/legal confirmation before production; do not silently assume them.

## 1. Product requirements (confirmed)
- Public website: facility introduction, facilities/amenities, guide, FAQ, house rules/terms, indicative rates with outbound Airbnb booking link, privacy policy, registration, anonymous post-stay survey.
- Languages: `ja`, `en`, `zh-CN`, `ko`. Localize UI, errors, consent and legal texts. Default Japanese; use locale-prefixed routes.
- Facility master and facility-specific capacity, name, time zone, content and terms. `guest_registrations.facility_id` is mandatory. Guest count is variable, capped by facility policy, not a hardcoded two.
- One submission creates one `guest_registration` plus 1..N `guests`, with exactly one representative. The representative can add companions and correct information after submission via a secure URL valid through local checkout date. Host performs identity verification in person; online submission does not imply verification.
- Register before check-in. Support partial registrations when planned count exceeds supplied guests; show incomplete count to host. The user may edit until the end of checkout day in facility timezone, subject to security policy.
- Optional `booking_source` and `booking_reference` on registration; no Airbnb import, reservation status machine, or cancellation handling.
- Admin: Firebase Auth login, facility-scoped list/search/detail/edit, identity verification record, CSV export, anonymous survey aggregate. No public exposure of personal data.
- Content: Markdown/JSON in Git. Survey is anonymous and has no FK to registration or booking. Existing Google Forms CSVs require import.
- Hosting is undecided. Local development via Docker Compose. Production architecture must remain portable.

## 2. Stack and boundaries
- Frontend: Next.js App Router, TypeScript strict, Mantine, next-intl. Form validation with a typed schema (e.g. Zod); never rely on client validation alone.
- API: Go REST, GORM, MariaDB (MySQL-compatible), OpenAPI 3.1 as API contract. Explicit DTOs; do not expose GORM entities directly.
- Admin authentication: Firebase Authentication ID tokens validated by Go (issuer, audience, signature, expiry); authorization is independent and enforced server-side by facility.
- Time: UTC for timestamps, `DATE` for local check-in/out dates, IANA timezone per facility. DB DATETIME(6) values represent UTC; enforce in application and document connection settings.
- IDs: BIGINT UNSIGNED auto-increment internally; do not expose enumerable IDs as authentication. Use opaque public IDs (UUID/ULID) where needed.

## 3. Relational model
### 3.1 facilities
`id`, unique `code`, `name`, `timezone`, `max_guests`, `is_active`, `created_at`, `updated_at`. Future `organization_id` can be added to this table; preserve facility IDs.

### 3.2 guest_registrations
`id`, `public_id` (unique opaque), `facility_id` (FK, mandatory), `check_in_date`, `check_out_date`, `estimated_arrival_time` (nullable), `planned_guest_count`, `capacity_snapshot`, `booking_source` (nullable), `booking_reference` (nullable), `terms_version`, `privacy_version`, `terms_accepted_at`, `privacy_accepted_at`, `submitted_at`, `identity_verified_at` (nullable), `identity_verified_by` (nullable), `created_at`, `updated_at`, `deleted_at` (nullable for controlled retention). Registration completeness is **derived** from `COUNT(guests)` vs `planned_guest_count`; no cancellation status.

### 3.3 guests
`id`, `registration_id` FK, `is_representative`, `full_name`, `name_kana` nullable, `address`, `phone` nullable, `email` nullable, `has_japan_address`, `nationality` nullable/conditional, `passport_number_ciphertext` nullable/conditional, `passport_number_last4` nullable, `created_at`, `updated_at`. `is_representative` must be exactly one per registration; enforce in transactional service logic and tests (MariaDB does not provide a simple portable partial unique index). Do not require contact details for each companion without a policy reason. Validate legally required fields against facility's applicable license/ordinance (**VERIFY**).

### 3.4 registration_access_tokens
`id`, `registration_id` FK, unique `token_hash` (SHA-256 of CSPRNG 32+ byte secret), `expires_at`, `revoked_at`, `created_at`, `last_used_at` nullable. Never store raw token. Token permissions are limited to a single registration; token does not grant admin rights. Reissue revokes earlier token(s). Rate-limit and audit use.

### 3.5 admin_users and admin_facility_access
`admin_users(id, firebase_uid UNIQUE, created_at, updated_at)` and `admin_facility_access(admin_user_id, facility_id, role, created_at)` with composite PK. Even if there is one host, always authorize against this relation. Future organization membership can replace/augment this relation.

### 3.6 audit_logs
`id`, `facility_id`, `registration_id` nullable, `actor_type` (`guest_token`/`admin`/`system`), `actor_admin_user_id` nullable, `action`, `changed_fields_json` nullable, `created_at`. Avoid plaintext passport numbers, full addresses, tokens, or exported CSV payloads in logs. Record CSV export event and relevant filters, not exported content. For sensitive corrections, retain limited protected change evidence under retention rules; avoid uncontrolled PII duplication.

### 3.7 survey_responses
`id`, `facility_id` FK (facility-level analysis is allowed), `survey_version`, `answers_json`, `submitted_at`, `created_at`. **No** `guest_id`, `registration_id`, booking reference, email, or identity linkage. Questions/options and translations live in versioned JSON in Git. Consider anti-spam, aggregate-only admin views, and small-cell disclosure risks.

### 3.8 import_batches and import_rows (recommended)
Track CSV source, checksum, imported timestamp, row index, outcome, linked record ID and non-sensitive error codes for reproducible migrations. Raw CSV should be encrypted/restricted outside application DB and deleted per import retention policy.

## 4. Core invariants and transaction rules
1. `check_out_date > check_in_date`; planned guest count >= 1; `1 <= supplied guests <= planned_guest_count <= facility.max_guests` at creation. Capacity is checked against **current facility policy** at time of submission and stored in `capacity_snapshot`. Capacity changes affecting later edits require explicit admin override or policy review; never silently invalidate historical data.
2. Exactly one guest has `is_representative=true`; guests belong to one registration; a registration must have at least one guest after commit.
3. All guest creation and registration creation in **one transaction**. On failure rollback both. Use idempotency key for create; store hash of key + request and cached result in a separate idempotency table or equivalent durable mechanism. Same key/different body -> conflict.
4. For additions/edits lock parent registration (`SELECT ... FOR UPDATE` in transaction) and re-count guests; prevent concurrent over-capacity and duplicate submissions.
5. `planned_guest_count` can be corrected within policy. Incomplete registration is allowed, but the host sees outstanding count. Do not label identity verified solely from form submission.
6. Editing requires valid unrevoked token and current facility-local time <= checkout date 23:59:59.999999 (convert to UTC for expiry); do not extend expiry automatically if checkout date is edited by guest without explicit policy. Admin may correct after expiry.
7. Editing representative or sensitive personal data requires step-up verification (**VERIFY exact method**). Until confirmed, guest-token access should be limited to companion additions and low-risk corrections, with admin-assisted sensitive changes. Never return raw passport data from guest manage endpoint.
8. All admin reads, edits and exports must check Firebase identity **and** facility access in Go, not merely `facility_id` query params. Query scoping belongs in repository/service layer.
9. Passport data: application-layer authenticated encryption (e.g. AES-256-GCM), keys outside DB, key rotation plan, no plaintext logging. Only collect/store passport data when legally necessary; passport-copy requirement is **VERIFY** for license/guest category.
10. Retention: lodging ledger generally subject to statutory retention; **VERIFY** applicable law, starting event, facility license, exceptions, and privacy notice. Separate no-stay submissions and anonymous survey retention. Implement scheduled eligible deletion with legal hold override, audit, backups expiry and test coverage; do not blindly purge after 3 years without legal review.

## 5. REST API contract (v1)
### Public
- `GET /api/v1/facilities/{code}/public` — public name, localized labels, capacity, current terms metadata; never secrets.
- `POST /api/v1/guest-registrations` — facility code, dates, planned count, optional booking fields, guests array, consent version/booleans; returns public registration ID, count/completeness, one-time edit token or separate secure token delivery. Require `Idempotency-Key`.
- `GET /api/v1/guest-registrations/manage` — token in Authorization header; return **minimal** registration summary, count, masked data; `Cache-Control: no-store`.
- `PATCH /api/v1/guest-registrations/manage` — safe field corrections after authorization, version/precondition support.
- `POST /api/v1/guest-registrations/manage/guests` — append one or more companions transactionally, return updated count.
- `PATCH /api/v1/guest-registrations/manage/guests/{guest_public_id}` — corrections within permitted scope; no passport plaintext returned.
- `POST /api/v1/surveys` — anonymous facility-scoped survey; no linkage.
### Admin
- `GET /api/v1/admin/facilities` — authorized facilities only.
- `GET /api/v1/admin/guest-registrations?facility_id=&from=&to=&q=&page=` — scoped paginated list.
- `GET /api/v1/admin/guest-registrations/{id}` — scoped detail.
- `PATCH /api/v1/admin/guest-registrations/{id}` — controlled corrections and audit.
- `POST /api/v1/admin/guest-registrations/{id}/verify-identity` — authenticated host verification event.
- `POST /api/v1/admin/guest-registrations/{id}/reissue-edit-link` — revoke old token, issue new with bounded expiry; secure delivery.
- `GET /api/v1/admin/guest-registrations/export?facility_id=&from=&to=` — authorized CSV with audit and spreadsheet formula-injection neutralization.
- `GET /api/v1/admin/surveys/summary?facility_id=&from=&to=` — aggregates only.

Define standard error envelope `{code,message,details,request_id}`. Suggested HTTP statuses: 400 invalid payload, 401 unauthenticated, 403 forbidden, 404 unavailable/not found, 409 duplicate or capacity conflict, 410 expired token, 422 validation error, 429 throttled. Never reveal whether a specific guest exists to unauthorized users.

### Create DTO example (illustrative, not exhaustive)
```json
{
  "facility_code": "saku-no-ie",
  "check_in_date": "2026-12-01",
  "check_out_date": "2026-12-03",
  "estimated_arrival_time": "16:00",
  "planned_guest_count": 3,
  "booking_source": "airbnb",
  "booking_reference": null,
  "terms_version": "2026-10-01",
  "privacy_version": "2026-10-01",
  "terms_accepted": true,
  "privacy_accepted": true,
  "guests": [
    {"is_representative": true, "full_name": "Example Representative", "address": "...", "has_japan_address": true},
    {"is_representative": false, "full_name": "Example Companion", "address": "...", "has_japan_address": true}
  ]
}
```
The backend determines timestamps, capacity snapshot, facility ID, and token expiry; never trust client-supplied values for these.

## 6. Pages and UX
- `/{locale}/` homepage; `/{locale}/facilities`, `/guide`, `/faq`, `/rates`, `/terms`, `/privacy`, `/survey`.
- `/{locale}/stays/{facilityCode}/registration` — select stay dates, planned count, representative, dynamically add/remove companions, conditional legal fields, consent, review, submit.
- `/{locale}/registration/complete` — show registered/planned count and secure edit-link instructions; avoid including token in analytics or referrer leakage.
- `/{locale}/registration/manage` — exchange one-time URL token for short-lived secure session where feasible; do not persist secret in localStorage; mask PII.
- `/admin` — authenticated dashboard; `/admin/guest-registrations`, `/admin/guest-registrations/{id}`, `/admin/surveys`.
- Each facility has a stable `code` and localized content; facility ID is resolved server-side. Do not hardcode two guests. Show remaining unregistered count clearly. Representative can select “same address/contact as representative” but copy only those fields with consent; do not copy nationality/passport automatically.
- Accessible labels, keyboard navigation, responsive mobile-first, clear localized validation and progress state. Avoid exposing tokens or PII to analytics, search crawlers, server logs or error tracking.

## 7. CSV migration
- Import old Google Forms guest register and survey separately; preserve original column mapping and source row number in restricted import metadata.
- Old guest rows are individual responses. Group into a registration only with strong evidence (e.g. confirmed shared booking identifier); same representative name and dates alone are insufficient. Ambiguous groups go to manual review; do not fabricate companion relationships.
- Normalize dates/timezones, Boolean answers, nationalities, consent text versions; distinguish blank vs false; log invalid rows without exposing personal data.
- Dry-run -> review -> import -> reconcile row counts, guest counts and representative uniqueness -> sign-off. Use idempotent import keys/checksums.
- Survey question choices mapped to stable codes and survey version; do not link to guest registrations.

## 8. Security, operations and quality gates
- TLS, secure cookie flags, CSRF protection if cookie-based session, strict CORS, Content Security Policy, rate limits and bot defense on public endpoints.
- Redact PII and tokens in logs, traces, Sentry and metrics; limit production DB access; encrypted backups and restore drills.
- CSV export requires authorized facility, logs export metadata, applies UTF-8 BOM only if operationally needed, and escapes spreadsheet formula-leading cells (`=`, `+`, `-`, `@`, tabs/control chars).
- Database migrations must be versioned and reversible where practical; never use GORM AutoMigrate as sole production migration mechanism.
- Test: creation atomicity, one representative, 1..N companions, planned count incomplete, facility capacity, concurrent additions, token expiry across timezones, token revocation, access isolation, sensitive edits, idempotency, CSV import ambiguity, retention, translations.
- CI: Go unit/integration tests against MariaDB, frontend typecheck/lint/tests, OpenAPI validation, migration checks, dependency/security scans.

## 9. Suggested repo layout
```text
apps/web/                    # Next.js App Router
services/api/                # Go API
  cmd/server/
  internal/{auth,facility,registration,guest,survey,admin,storage}/
  api/openapi.yaml
  migrations/
content/{ja,en,zh-CN,ko}/    # Markdown and JSON
infra/docker-compose.yml
scripts/import-google-forms/
CLAUDE.md
```

## 10. Delivery order
1. Versioned MariaDB migrations and facility seed (`saku-no-ie`); admin authorization tables.
2. OpenAPI contract + DTO validation + transactional registration service.
3. Public registration UI and token-limited companion addition/correction.
4. Admin list/detail/verification/export with audit.
5. Anonymous survey and aggregates; public content and 4-language localization.
6. CSV migration rehearsal, threat review, legal/privacy checks, end-to-end tests, deployment selection.

## 11. Open decisions / mandatory checks
- **VERIFY** applicable lodging business license and local legal requirements: required ledger fields, passport inspection/copy, precise retention basis, privacy notice, and consent text.
- **VERIFY** whether checkout-day edit link remains valid until 23:59:59 local or should close earlier; currently design assumes end of local date.
- **VERIFY** step-up identity method for edits of existing personal data, especially representative changes; do not implement unrestricted token-only PII editing.
- **VERIFY** whether the representative's email is mandatory for safe edit-link recovery; otherwise admin reissues manually.
- **VERIFY** facility-specific terms versions, rate examples, Airbnb link, content and translations.
- Hosting/domain and email delivery provider remain undecided; do not couple application code to a vendor.
- **VERIFY** whether legal minimum guest fields vary by accommodation license; implement conditional requirements without changing schema.

## 12. Coding instructions for Claude Code
- Follow this contract. If a request conflicts with confirmed requirements, flag the conflict before changing schema or behavior.
- Generate migration SQL and OpenAPI before implementing endpoints. Keep DB schema, GORM models, API DTOs and UI validation in sync.
- Prefer small cohesive services, explicit transactions and context propagation; return typed errors and no sensitive error details.
- Never put PII, passport plaintext, edit tokens or credentials in fixtures, logs, snapshots or committed files. Use synthetic data only.
- Add tests for every changed invariant, especially multi-facility authorization and concurrency.
- Document new decisions in this file; do not silently introduce organizations or reservation management.

## 13. Decisions log (2026-11-11)
Implementation decisions that refine or deviate from this contract, recorded during MVP Phases 1–6 (plan `.omc/plans/sakunoie-mvp.md`). Add new decisions at the end. Items that need the owner before launch are listed in **Open items before production** at the end of this section.
- **Schema and migrations: Prisma migrate + GORM.** §2/§8/§12 "migration SQL" is read as Prisma 7 migrations in `backend/prisma/migrations/` generated from `backend/prisma/schema.prisma`. Go accesses the DB through GORM for CRUD only, with no AutoMigrate. Models are PascalCase with `@@map`/`@map` to the snake_case names of `docs/schema.sql`. There is no Prisma Client (no generator block).
- **CHECK constraints** (`chk_facility_capacity`, `chk_registration_dates`, `chk_registration_counts`) live in the `*_add_check_constraints` migration, created with `prisma migrate dev --create-only` and filled by hand. It is the only sanctioned hand-edited migration.
- **JSON columns** (`audit_logs.changed_fields_json`, `survey_responses.answers_json`, `api_idempotency_keys.response_json`) are `String @db.LongText` in Prisma, with `json_valid()` CHECKs (`chk_audit_changed_fields_json`, `chk_survey_answers_json`, `chk_idempotency_response_json`) in the CHECK migration. In MariaDB, `JSON` is an alias for exactly that. Prisma's `Json` type made `migrate diff --from-migrations … --exit-code` report false drift on MariaDB 11.1. The columns use the table collation (`utf8mb4_unicode_ci`) instead of the `utf8mb4_bin` that the JSON alias would give. The Go API validates and marshals the JSON.
- **`public_id` collation:** `CHAR(36)` in the table's utf8mb4 collation. The `ascii_bin` override from `docs/schema.sql` is dropped, because UUIDs are lowercase hex and the override risks Prisma drift.
- **Index and FK names** follow `docs/schema.sql`. Inline `UNIQUE` columns keep MariaDB's implicit names (`code`, `public_id`, `firebase_uid`, `token_hash`, `key_hash`). Implicit FK indexes are declared explicitly under the FK name. All FKs are `ON DELETE RESTRICT ON UPDATE RESTRICT`.
- **Databases.** Shared dev server `192.168.3.100:23306`: only `lodging-master-dev`, and only `migrate deploy`, `migrate status` and the seed (`backend/scripts/db-deploy-dev.sh`). Everything destructive or shadow-creating runs on the throwaway compose MariaDB `backend/compose.test.yaml` (`127.0.0.1:33306`; `lodging_migrate`, `lodging_shadow`, `lodging_test`, `lodging_e2e`). Guards: the `backend/prisma.config.ts` argv guard and `backend/scripts/assert-safe-db.sh`.
- **`lodging-master-dev` default collation** is `utf8mb4_general_ci`, because the database already existed before Phase 1 deployment, so `CREATE DATABASE IF NOT EXISTS` was a no-op. Every table is created with `utf8mb4_unicode_ci` explicitly, so this has no effect on the schema. It is left as is because `ALTER DATABASE` is not an allowed operation on the shared server.
- **Seed** `backend/prisma/seed.sql` is insert-only (`ON DUPLICATE KEY UPDATE code = code`), so re-running it never overwrites an operator's later capacity change (§4.1). `max_guests = 4` is a placeholder (**VERIFY**).
- **Error envelope (Phase 2).** One constructor (`internal/models/error.go` `NewEnvelope`) is installed as both `huma.NewError` and `huma.NewErrorWithContext`. It keeps only `location`/`message` of each detail and drops Huma's `ErrorDetail.Value` and any non-detail error text, because those can echo input such as passport numbers. A Huma transformer stamps `request_id` on envelopes built without a context. Huma's default `$schema` link transformer is removed (`CreateHooks = nil`), so bodies and the spec carry no `$schema`. Unknown routes, 429s and panics use the same envelope. `gin.Recovery` is replaced because it dumps the request line, which includes the query string.
- **Error codes (stable, for frontend i18n):** `validation_error` 422, `facility_unavailable` 422 (create with an unknown or inactive facility; `GET …/public` returns 404 `not_found`), `capacity_exceeded` 409, `idempotency_conflict` 409, `token_invalid` 401 (missing, malformed, unknown or revoked token, or soft-deleted registration), `token_expired` 410, `field_not_editable` 403, `not_found` 404, `precondition_failed` 412, `rate_limited` 429, `internal_error` 500. Plain Huma errors default by status (`bad_request`, …).
- **Create status.** A new registration returns 201. An idempotent replay returns 200 with an additional token, or `edit_token: null` and `code: "edit_link_unavailable"`. Create and every `/manage` response send `Cache-Control: no-store`.
- **Edit-token transport.** The manage operations declare an `editToken` HTTP bearer security scheme. A Huma middleware reads `Authorization: Bearer`. `adminBearer` is declared now for Phase 4.
- **Optimistic concurrency on manage PATCH (deviation from plan "conditional.Params").** `If-Match` is a required header parameter (a missing header gives 422 from Huma validation). The ETag is opaque (`"r<base36 of updated_at µs>"`) and is also returned as `version` in the body. `If-Match` must be exactly that ETag, quoted or as the bare `version`; `*`, weak `W/` tags and lists get 412. Every manage mutation, including a companion edit or addition, bumps `guest_registrations.updated_at`, so one ETag covers the whole registration. Reason: `conditional.Params` cannot make the header required and adds four unrelated headers to the contract.
- **Non-editable fields are declared in the PATCH DTOs** (`check_in_date`, `check_out_date`, `facility_code`; for guests `is_representative`, `has_japan_address`, `nationality`, `passport_number`). Sending one returns 403 `field_not_editable` instead of a generic 422 for an unknown property. Any PATCH of the representative returns 403.
- **Transactions.** Create Tx1 uses the default REPEATABLE READ. The replay (Tx3) and all manage mutations use READ COMMITTED with the registration row `FOR UPDATE`, and they count guests with a locking read, so concurrent additions never see a stale snapshot. Token resolution locks the registration row, then the token row, and records `last_used_at` (see the lock-order decision below).
- **Validation additions.** `check_out_date` must not be before today in the facility time zone (422). A passport number sent for a guest with `has_japan_address=true` is discarded and not stored (data minimization). Create-time audit rows use `actor_type = guest_token` and action `registration.created`. Manage audit rows store field-name arrays (`["guests"]`, `["guest.phone", …]`, `["planned_guest_count", …]`).
- **DB clock and logging.** `internal/database` logs the single `loc overridden` WARN whenever the DSN lacks `loc=UTC` **or** `time_zone='+00:00'`, so the compose test DSN triggers it too. GORM logs at Warn with `ParameterizedQueries` (no bound values). Service errors are logged by kind (`mysql_<n>`/`other`), never by message, because MySQL duplicate-key messages quote values.
- **Env loading.** `cmd/server` reads the process environment only and does not load `backend/.env`. Export the variables or set them inline (README follow-up in Phase 6).
- **Frontend i18n (Phase 3).** `next-intl` is pinned exactly to **4.14.9**: it is the newest 4.x release older than the npm `min-release-age=7` policy (4.14.10 and 4.15.0 were too new). Next 16 `src/proxy.ts` composes `createMiddleware(routing)` from `next-intl/middleware`; see `node_modules/next/dist/docs/01-app/02-guides/content-security-policy.md` and https://next-intl.dev/docs/routing/middleware. Locales `ja` (default), `en`, `zh-CN`, `ko` with `localePrefix: 'always'`. Messages live in `frontend/messages/{locale}.json` and are type-checked against `ja.json` (`src/types/next-intl.d.ts`); a Vitest parity test checks keys and ICU arguments across all four.
- **Nonce CSP (Phase 3).** `src/proxy.ts` builds the policy with `src/lib/security-headers.ts`, puts `Content-Security-Policy` and `x-nonce` on the forwarded **request** headers and the CSP on every response. For locale routes it hands next-intl a `new NextRequest(request, { headers })`; next-intl 4.14.9 copies the incoming request headers into its rewrite/next response, so no internal `x-middleware-*` headers are touched. The proxy also sets `X-Content-Type-Options: nosniff`, `Referrer-Policy` (`no-referrer` plus `X-Robots-Tag: noindex, nofollow` on `/{locale}/registration/*`, `/{locale}/stays/*` and `/admin/*`, otherwise `strict-origin-when-cross-origin`). These security headers live in the proxy, not in `next.config.ts` `headers()`. The matcher excludes `api/`, all of `_next/` (static, image and the dev HMR socket) and files with an extension.
- **No Mantine `ColorSchemeScript` (deviation from plan).** The color scheme is forced to light, and `mantineHtmlProps` already sets the attribute, so the inline script is dropped and nothing needs the nonce in server code. Pages stay dynamic through `await connection()` in `[locale]/layout.tsx`, which the nonce CSP requires. `x-nonce` is still forwarded for later server code.
- **Zod runs `jitless`** (`z.config({ jitless: true })` in `features/registration/types/schemas.ts`). Otherwise Zod 4 probes `new Function`, which the production CSP (no `'unsafe-eval'`) reports as a violation.
- **Rewrite target is fixed at build time.** `next build` bakes `API_PROXY_TARGET` (default `http://127.0.0.1:8080`) into the routes manifest, so an E2E or production build must be built with the right value. Use `node .next/standalone/server.js` for production-mode runs: `next start` warns it does not support `output: 'standalone'`. `allowedDevOrigins: ['127.0.0.1']` lets E2E drive `next dev` at `http://127.0.0.1:3100` (Next 16 blocks dev resources for other origins).
- **Guest edit-token handling (Decision C, as built).** `features/registration/stores/manage-token-store.ts` moves `#token=` into `sessionStorage['sakunoie.manage.token']` and strips the fragment with `history.replaceState`; the registration feature registers it with `registerAuthTokenProvider('/guest-registrations/manage', …)`. The token is cleared on 401/410 and on "finish editing". The completion page builds the edit URL from the in-memory create result only; after a reload it shows "ask the host to reissue".
- **Wizard idempotency (as built).** One `crypto.randomUUID()` key per wizard session in the in-memory zustand store (no `persist`). Network failures and timeouts are retried twice with the same key. A 409 `idempotency_conflict`, or a 200 replay with `edit_token: null`, shows "already submitted; contact the host" and keeps the key. The key is dropped only when the completion page has displayed the result, or when the guest presses "start a different registration". The draft (including passport numbers) is cleared at success.
- **Manage UI scope.** Stay corrections send only changed fields (an emptied optional field is sent as `""` to clear it). Companion corrections cannot prefill masked values, so an empty field means "unchanged" and clearing a companion's phone/email/kana is not offered in the UI. A 412 reloads the summary and asks the guest to retry. Guest form inputs use `autoComplete="off"`.
- **Known issue: `global-not-found.tsx`** (experimental in Next 16.4) is built, but unmatched non-locale URLs (`/admin`, `/foo.txt`) render Next's built-in 404 page in production (status 404 is correct). Unknown localized paths render the localized `[locale]/not-found.tsx` through the `[...rest]` catch-all, and other unprefixed paths are redirected into a locale by next-intl. Phase 4's `app/admin/not-found.tsx` covers `/admin/*`.
- **Firebase spike outcome (Phase 4, Decision A).** firebase-admin-go v4.22.0. Without ADC, `firebase.NewApp` succeeds but `app.Auth()` fails ("could not find default credentials"), because the auth client builds a signer. With `option.WithoutAuthentication()` it initializes, fetches Google's public certs without credentials, and rejects forged RS256 and unsigned tokens. The server always uses `WithoutAuthentication`, so **production needs only `FIREBASE_PROJECT_ID`; no service account or ADC.** Consequence: `VerifyIDTokenAndCheckRevoked` and user management are unavailable (not used). In emulator mode `VerifyIDToken` skips the signature but always calls the emulator's `accounts:lookup` (user exists and is enabled), so the "offline" claim tests (`admin_auth_test.go`) use a loopback `httptest` stub as the emulator; they never skip. `FIREBASE_PROJECT_ID` is now required at startup, and `NewFirebaseVerifier` re-checks the emulator/`demo-` rule.
- **Live wrong-`aud` test (deviation).** The emulator signs client tokens only for its default project, so a `demo-other` token cannot be minted through its REST API. The live test verifies a real `demo-sakunoie` emulator token with a verifier for `demo-other` (the same `aud` check); the offline test covers wrong `aud`/`iss`, expiry, empty `sub`, future `iat` and missing `aud` with hand-built tokens.
- **Admin authorization.** `AdminAuth` Huma middleware on every `/admin` operation (before input parsing): missing/invalid token → 401 `unauthenticated` (+ `WWW-Authenticate: Bearer`), unknown uid or no `owner` row → 403 `not_authorized`; all admin responses send `Cache-Control: no-store`. Rows with any other role grant nothing. An empty `AdminScope` fails closed in every service method. `facility_id` outside the scope → 403 `forbidden` (list, export); a registration of another facility → 404 `not_found` (detail, PATCH, verify, reissue). Admins never see soft-deleted rows.
- **Ledger semantics.** `from`/`to` filter the **check-in date** (inclusive, facility-local civil dates). `q` is a substring search over any guest's `full_name`/`name_kana` and `booking_reference`, with LIKE wildcards escaped. 50 rows per page, newest check-in first; `metrics` cover the whole filter, not the page. `facility_id` is the numeric facility ID from `GET /admin/facilities` (not an authorization credential).
- **Detail masking.** Address, phone and email are masked server-side by default; `?reveal=true` returns them and writes `admin.registration_revealed` (field names only). Passport numbers: last 4 only, everywhere.
- **Admin PATCH.** Requires `If-Match` (same ETag as manage). Dates, arrival, counts, booking fields and guest fields including the representative (`is_representative: true` moves it) and passport (re-encrypted with the guest public_id AAD) are editable. `planned_guest_count` must be ≥ registered and ≤ current `max_guests` unless `capacity_override: true` (audited as `capacity_override`; `capacity_snapshot` is never changed). Check-out in the past is allowed for the host. `has_japan_address: true` deletes the stored passport. Masked values are never prefilled in the UI; empty means unchanged. Edit-token expiry is not recomputed by a date correction; a reissue uses the current checkout date.
- **Verify identity** is idempotent: the first verification (time and admin) is kept and audited once. **Reissue** revokes every token and returns the new one once with `expired: true` when the checkout day has already ended.
- **CSV export (as built).** `facility_id` is required (one facility per file). The file is built in memory, then the `registrations.exported` audit row (`{facility_id, from, to, registration_count, row_count, format}`) is committed in the same transaction **before** any byte is sent, so there is no file without an audit row (deviation from "stream": MVP volumes are small). UTF-8 with BOM, CRLF, one row per guest, English snake_case headers. `utils.CSVSafe` prefixes `'` to cells starting with `= + - @`, their full-width forms, or any control character (tab, CR, LF, …). Filename `registrations-<code>-<from|start>-<to|end>.csv`.
- **Audit actions (admin).** `admin.registration_updated`, `admin.registration_revealed`, `registration.identity_verified`, `registration.edit_link_reissued`, `registrations.exported`. `writeAuditMeta` stores non-PII metadata objects (export only); everything else stores field-name arrays.
- **`cmd/admin-grant`** (`--uid --facility`, role always `owner`, idempotent, upgrades another role to owner) reads `GORM_DATABASE_URL` and refuses every target except compose `127.0.0.1:${TEST_DB_PORT:-33306}` `lodging_test`/`lodging_e2e` and `192.168.3.100:23306/lodging-master-dev`, before connecting and without printing the DSN.
- **Admin frontend (Phase 4).** `firebase` JS SDK 13.x (Auth only). `initializeAuth` with `browserSessionPersistence` and no popup/redirect resolver (email/password needs no iframe, so the CSP needs no `frame-src`); the sign-in ends with the tab. `NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST` drives both `connectAuthEmulator` and the admin `connect-src` origin; it is inlined at build time, so an E2E build must set it (with `API_PROXY_TARGET`). Without it, `NEXT_PUBLIC_FIREBASE_API_KEY`, `_AUTH_DOMAIN` and `_PROJECT_ID` are required. Console pages live in the route group `src/app/admin/(console)/` behind `AdminConsole` (signed out → login; 403 → "権限がありません"); `admin/[...rest]` renders `admin/not-found.tsx`, which is a client component (Mantine `Anchor component={Link}` cannot be rendered from a server component). Ledger filters live in the URL. The query cache is cleared on sign-out, and revealed details are not cached (`gcTime: 0`).
- **Survey (Phase 5).** `POST /api/v1/surveys` (exact path; `ClassifyRoute` applies `RATE_LIMIT_SURVEY_PER_MIN`) returns 202 with no body. The question set is versioned JSON: `backend/internal/services/surveydef/v1.json` (embedded; codes `q01..q10`, types `rating_1_5`/`single_choice`/`multi_choice`/`text`, `required`, `maxLength`) validates, and `frontend/content/surveys/v1.json` holds the same structure plus labels in 4 locales; a Vitest test fails if codes, types, options, required flags or `maxLength` differ. Questions are **placeholders** (要差し替え: Google Forms の既存10問); only `q01` is required. Answers are normalized (ratings as integers, text trimmed, empty answers dropped) and stored as `{"q01":5,…}`; 422 details never echo submitted keys or values (an unknown code is reported at `body.answers`). A non-empty honeypot `website` gets 202 and nothing is stored. Unknown or inactive facility → 422 `facility_unavailable`; any other `survey_version` → 422.
- **Survey privacy.** `submitted_at` and `created_at` are both stored as 00:00:00 UTC of the submission's UTC day, so rows cannot be matched to a stay by time; the auto-increment `id` still implies order but is never exposed. `survey_responses` has exactly `id, facility_id, survey_version, answers_json, submitted_at, created_at` (integration test). Free-text answers are stored but never returned by any API; the form asks guests not to write identifying details.
- **Survey facility source.** `/{locale}/survey` uses `?facility=<code>` when it is in `content/site.json` `facilities`, else `defaultFacilityCode` (an unknown query value silently falls back); a select is shown only when more than one facility is listed. The server re-validates the code.
- **Survey summary.** `GET /api/v1/admin/surveys/summary?facility_id=&from=&to=` uses the `adminOp` wrapper and `facilitiesFilter` (other facility → 403, omitted → all facilities in scope). Only the **current** version is aggregated. `from`/`to` are inclusive **UTC** submission days (that is what is stored). Per question: `answered`, rating `mean` (2 decimals) and 1–5 `counts`, option `counts` in definition order; text questions report `answered` only. **Small-cell rule:** a question with `answered < SURVEY_MIN_RESPONSES` (default 5) has `suppressed: true` and `mean`/`counts` null; `total` and `answered` are always shown. Aggregation runs in Go over the matching rows (MVP volumes).
- **Public content (Phase 5).** Markdown with front matter in `frontend/content/{ja,en,zh-CN,ko}/{home,facilities,guide,faq,rates,terms,privacy}.md` (`title`, `description`, quoted `version`/`effectiveDate`, `placeholder`, `todo` containing 要差し替え, plus page-specific structured fields validated with Zod), read on the server with `gray-matter` and rendered with `react-markdown` (no raw HTML). `next.config.ts` `outputFileTracingIncludes` ships `content/` with the standalone build. A missing locale file falls back to `ja` with a warning; the content test requires all 28. Every page shows a visible "要差し替え / Placeholder content" banner while `placeholder: true`. Terms/privacy `version` must equal `backend/internal/services/policy_versions.json` (`2026-11-01`; test). `defaultFacilityCode` moved from `src/config/site.ts` to `content/site.json`; facility data (`content/facilities/<code>.json`: names, area, `airbnbUrl` + `airbnbUrlIsPlaceholder`, amenities, photos with `alt` per locale, `aspectRatio`, `focalPoint`) is registered by static import in `src/config/site.ts`, importable through the `@content/*` alias.
- **Truth in presentation.** Copy states that guests stay in a private room in the host's home, not an entire-home rental, with some shared spaces. Amenities list only those facts; Wi-Fi, parking, laundry, capacity, room size and all prices say "being confirmed". The rates page shows no amounts and says Airbnb sets actual prices. The Airbnb URL is the Airbnb top page until the real listing replaces it. Photos are generated, textless SVG shapes in `public/images/placeholder/` with a visible "placeholder image" badge.
- **Navigation.** Header: desktop (≥ Mantine `md`) shows facilities/guide/FAQ/rates and the Airbnb CTA; below that a burger opens a drawer with the same links and the CTA. External links use `target=_blank rel="noopener noreferrer"` with a visually hidden "opens in a new tab". Footer links all public pages, registration (default facility) and the survey.
- **RSC caveat.** Server components must use Mantine's flat exports (`TableTr`, `ListItem`, `AccordionItem`, …): compound members such as `Table.Tr` are undefined on a client reference in RSC and crash the page at runtime (the build does not catch it).
- **Admin survey UI.** `/admin/surveys` (ja only) shows total, the threshold, a banner when total is below it, and one card per question with counts as text plus decorative bars; suppressed questions and free text show an explanatory sentence instead of values. Filters (facility when more than one, UTC date range) are component state, not URL state.
- **Repository layout (replaces §9).** The code lives in the existing `backend/` (Go: Gin + Huma v2, GORM, Prisma 7 for schema/migrations only) and `frontend/` (Next 16 App Router, Mantine v9, next-intl) directories, each with its own README and `CLAUDE.md`. There is no `apps/`/`services/` split.
- **API contract.** Huma code-first OpenAPI 3.1, committed as `backend/api/openapi.yaml` and drift-tested (`backend/api/openapi_test.go`). `/docs`, `/openapi.*` and `/schemas/*` are served only when `APP_ENV` is `development` or `test`.
- **Decision B (frontend → API).** Same origin through the Next rewrite `/api/:path*` → `API_PROXY_TARGET`. Go installs no CORS middleware. The client IP for the rate limiter comes from `X-Real-IP` only (see the client-IP decision below).
- **Decision D (idempotent create), as built.** No raw token is stored. A valid replay mints an **additional** token while fewer than 3 are active and no admin reissue has happened; otherwise it returns `edit_token: null` + `edit_link_unavailable`. Admin reissue revokes all tokens. The request hash is HMAC-SHA256, key rows expire after 24 h, and a janitor deletes them.
- **Audit deviation from §3.6 (VERIFY with legal).** `audit_logs.changed_fields_json` stores field names only (plus non-PII export metadata), not protected before/after evidence of sensitive changes.
- **Admin UI is Japanese-only** and lives outside `[locale]` (`src/app/admin`, its own root layout with `lang="ja"`). The MVP role is `owner` only; `staff` is a follow-up.
- **`cacheComponents` stays off.** If it is enabled later, the `headers()`/`connection()` reads that obtain the CSP nonce (and every other request-time read) must sit inside `<Suspense>` boundaries.
- **E2E (Phase 6).** Playwright (`@playwright/test` 1.64, Chromium) in `frontend/e2e/`: `public`, `registration`, `manage`, `admin` and `survey` specs. The `desktop-chromium` project (1280 px) runs all of them, and `mobile-chromium` (390×844) runs the wizard spec. The settings are `workers: 1` and `fullyParallel: false`. Each spec seeds its own synthetic rows through the public API in `beforeAll`, with unique `E2E-<spec>-<uuid8>` booking references and a random far-future stay date. Every spec passes alone. `e2e/global-setup.ts` checks the stack, resets the emulator accounts, signs up an admin user and an ungranted user with a random per-run password (kept only in env), and grants the owner role through `go run ./cmd/admin-grant` on compose `lodging_e2e`.
- **E2E runs against the production build (deviation from plan "next dev").** `frontend/scripts/e2e-stack.sh start|stop|status` does the following:
  - starts compose and recreates `lodging_e2e`;
  - starts the Auth Emulator on :9099;
  - builds and starts the API on :8180 with `APP_ENV=test`, per-run `openssl rand` keys in exported variables, `RATE_LIMIT_*_PER_MIN=1000` and `SURVEY_MIN_RESPONSES=1`;
  - runs `next build` with `API_PROXY_TARGET=http://127.0.0.1:8180` and `NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099`, then runs the standalone server on :3100, so the nonce CSP is tested without dev-only `'unsafe-eval'`.

  The script refuses ports that are already bound, starts every process with `setsid` and stops only those process groups, and finishes with `compose down -v`. The CI `e2e` job runs the same script.
- **E2E coverage notes.** The suite checks:
  - the CSP header and zero CSP violations (console text plus `securitypolicyviolation` events) on `/ja` and `/admin/login`;
  - that the edit token never appears in any request URL;
  - the lost-response retry (the server commits, the browser sees a network failure) and the 409 "already submitted" path with one Idempotency-Key throughout;
  - a server 422 with client validation bypassed (the request is rewritten in the route handler);
  - that reissue makes the old token return 401;
  - the CSV BOM, header and `'=SUM(1)` neutralization.

  The expired-token (410) state is covered twice: Go clock-injection tests check the expiry itself, and the E2E `manage` spec (AC6) replaces the manage API answer with the server's 410 `token_expired` envelope through `page.route` and checks the recovery guidance and that the token is dropped, because a token cannot expire in real time during a run.
- **E2E waits.** No `networkidle`: the CSP checks wait for a concrete hydrated element (`waitForHydrated`, React props on the DOM node) and then `expect.poll` the violations. Text locators are scoped to a role or region instead of `.first()`.
- **Hydration timing (observed).** A language change made before React hydrates the header is ignored, because the native `<select>` has no handler yet. The E2E waits for hydration (React props on the select) before switching. This is a minor UX gap on slow connections, not a correctness bug.
- **OpenAPI lint (Phase 6).** `npx @redocly/cli lint` (recommended rules) originally reported `no-empty-servers` and three `security-defined` errors. The public create, facility and survey operations now declare `security: []`, and the spec has the same-origin server `/`. Only the `info-license` warning remains (no license chosen).
- **Client IP behind the Next rewrite (security review HIGH-1).** Next 16's external rewrite (`proxy-request.js`, httpxy without `xfwd`) forwards the client's `X-Forwarded-For` unchanged and does not append the peer, so Go (peer `127.0.0.1`, trusted) would have keyed the rate limiter on a client-chosen XFF, or put every client in one bucket when XFF is absent. `middlewares.ConfigureClientIP` sets `RemoteIPHeaders = ["X-Real-IP"]` with `SetTrustedProxies(TRUSTED_PROXIES)`; XFF is never read. **Deployment requirement:** the edge (nginx/Caddy/LB) overwrites `X-Real-IP` with the peer address and strips any client value (or routes `/api/` straight to Go), and `TRUSTED_PROXIES` lists only the hop that connects to Go (`backend/README.md` "Client IP and the edge proxy"). Without an edge, all clients share one bucket (local dev only). Test: `TestRateLimitClientIPTrustBoundary`.
- **API response headers and limiter memory.** `middlewares.SecurityHeaders` (first in the Gin chain) sets `X-Content-Type-Options: nosniff` on every Go response, including 404/429/panic envelopes. **HSTS is the edge's job** (TLS terminates there). The rate limiter caps its bucket map at 100,000 entries: when full it sweeps expired buckets, then drops arbitrary ones down to 90% and logs a WARN (an evicted client just gets a fresh bucket).
- **Lock order: registration, then tokens (deadlock 1213 fix).** Manage used to lock the token `FOR UPDATE` and then the registration, while an admin reissue locks the registration and then UPDATEs the tokens, so a concurrent guest PATCH and reissue deadlocked (reproduced by `TestIntegrationConcurrentManagePatchAndReissue`). `TokenService.Resolve` now reads the token without a lock, locks the registration `FOR UPDATE`, then re-reads the token `FOR UPDATE` and re-validates it (revocation, expiry); the manage mutations reuse that locked row. As a backstop, the manage and admin transaction helpers (`readCommittedTx`) retry the whole transaction once on MariaDB error 1213. The lock order alone passes the race test without the retry.
- **Contact format checks (server).** Create, add-companions, manage guest PATCH and admin PATCH check `phone` (digits incl. full-width, optional leading `+`, spaces, hyphens, dots, parentheses; at least one digit) and `email` (one `@`, a dot in the domain, no spaces or `<>,;"`) after trimming, through one helper (`validateContact`). The free-text guest corrections (full_name/address required if sent, optional fields trimmed with `""` clearing) are shared by both PATCH paths (`guestEdits`). Errors are 422 `validation_error` at `body.phone`/`body.guests[i].email`, never echoing the value.

### Open items before production
1. **VERIFY items of §11:** lodging-license ledger fields and passport inspection/copy rules, retention basis, privacy notice and consent text, the checkout-day edit window (currently 23:59:59 local), step-up identity for edits, whether the representative's email is mandatory, and the facility terms versions, rates, Airbnb link, content and translations. Also the §3.6 audit-evidence deviation above.
2. **Q2:** the real `max_guests` for `saku-no-ie` (seeded as the placeholder 4).
3. **Q6:** confirm the token-editable allowlist (companion `full_name`/`name_kana`/`address`/`phone`/`email`; registration arrival time, booking source/reference, planned count; nothing on the representative).
4. **Q7:** confirm that the CSV (UTF-8 with BOM) may carry passport **last 4 only**. The legal ledger may require full numbers.
5. **Next upgraded to 16.4.5 (was 16.4.0).** The user-approved target was `^16.4.6`, but 16.4.6 (published 2026-11-09) is still blocked by the npm `min-release-age=7` policy, so `next` is `^16.4.5` (16.4.5 installed, the newest allowed patch; `@next/env` and the SWC binaries follow). All three `next` advisories (GHSA-2464-fhr4-w9xw image-optimizer RCE, GHSA-p3m5-59mp-jv5r Server Actions DoS, GHSA-673q-mjm6-r5xr `use cache` RCE) cover only `<=16.4.0`, so `npm audit --omit=dev` no longer lists `next`. `images: { unoptimized: true }` additionally removes `/_next/image` (all images are local placeholders). Move to 16.4.6 once it is older than 7 days. The Go toolchain is pinned to `toolchain go1.26.11` (govulncheck: 29 reachable → 0; quic-go v0.59.1); one unreachable `golang.org/x/crypto` advisory (GO-2026-5932, fix only in a pseudo-version) remains.
6. **`@grpc/grpc-js` advisory via `firebase`.** It comes in through `@firebase/firestore` (high; e.g. GHSA-m9gg-hp2v-232j). The app uses Firebase **Auth only**, so Firestore code is not imported, but the dependency remains until `firebase` ships a fixed range. npm's suggested "fix" is a semver-major downgrade to 9.x and is not appropriate.
7. **Collation mismatch.** The shared `lodging-master-dev` database default is `utf8mb4_general_ci` (it existed before Phase 1), while `docs/schema.sql` and every table use `utf8mb4_unicode_ci`. The tables are explicit, so behavior is correct, but a production DB should be created with `utf8mb4_unicode_ci`.
8. **Redocly lint:** valid. The `info-license` warning stays until a license is chosen.
9. **AC1 guard check left for the owner.** The `prisma.config.ts` argv guard check (`DATABASE_URL=<shared .env URL> npx prisma migrate dev` must exit non-zero with the refusal before connecting) was not run against the real shared URL by the agents. The owner should run it once (read-only).
10. **M2: admin revocation.** `VerifyIDToken` does not check revocation (the server runs `WithoutAuthentication`, so `VerifyIDTokenAndCheckRevoked` needs credentials), so a deleted or disabled Firebase admin keeps a valid ID token for up to 1 h. **Immediate revocation procedure:** delete the admin's `admin_facility_access` row(s) (`DELETE FROM admin_facility_access WHERE admin_user_id = …`); the next request gets 403 `not_authorized`, because `AdminAuth` checks the rows on every request. Follow-up: `VerifyIDTokenAndCheckRevoked` with a service account once hosting is chosen.
11. **Security review follow-ups (low):**
    - **L1** the anonymous survey can be poisoned by rotating IPs; consider a per-facility daily cap or a challenge.
    - **L5** the `prisma.config.ts` argv guard does not cover `prisma db execute --file` or `migrate resolve` against the shared DB (by design for the seed). `db execute` runs only through `scripts/db-deploy-dev.sh`; give the shared-server DB user no DROP rights.
    - **L6** GitHub Actions are pinned by tag, not by commit SHA (`permissions: contents: read` is done). Pin by SHA before the repository is public or uses secrets.
    - **L7** CSP `style-src 'unsafe-inline'` is required by Mantine's inline styles. Accepted.
    - **L3 (rest)** Go now sends `nosniff` and the Next proxy sends it on pages, but `_next/*` static responses (outside the proxy matcher) do not; set it, and HSTS, at the edge.
12. **Survey small-cell differencing.** Suppression is per query, so an admin can compare two overlapping date ranges (e.g. `from=D` vs `from=D+1`) whose totals are both above `SURVEY_MIN_RESPONSES` and subtract them to recover a single day's answers. Acceptable for the MVP (one owner per facility, coarse UTC-day storage, no free text returned). Options later: fixed reporting periods (whole months), or a minimum difference between overlapping queries.
13. **Deploy checklist: client IP (required).** The edge proxy must overwrite `X-Real-IP` with the peer address and strip any client-supplied value (or route `/api/` straight to Go), and `TRUSTED_PROXIES` must list only the hop that connects to Go. If Next is exposed without such an edge, a client-supplied `X-Real-IP` passes through the rewrite and bypasses the rate limiter. Follow-up (low): key IPv6 clients by /64, so address churn within one prefix cannot evict buckets at the 100k cap.
14. Follow-ups from the ADR: a C2 cookie session once hosting is chosen; CSP `report-to`; a `staff` role; a key-rotation re-encryption job; retention/deletion jobs; CSV import (§10-6); a distributed rate limiter for more than one API instance; hosting, domain and email provider.
