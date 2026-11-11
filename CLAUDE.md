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
