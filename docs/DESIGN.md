# DESIGN.md — 佐久の家 / Saku no Ie

> Status: Approved visual direction, design specification v1.0 (2026-11-11). This document defines presentation and interaction, not application implementation. Read alongside `CLAUDE.md`; functional and security requirements in `CLAUDE.md` take precedence where applicable.

## 1. Design intent

**Concept:** A small boutique hotel's polish, with the warmth of a host-occupied homestay in Saku, Nagano. The public site should feel inviting and photographic; the guest portal should feel calm and effortless; the admin should feel efficient and trustworthy.

- Visual style: natural modern + hotel-like refinement; restrained Japanese editorial typography.
- Keywords: warm, quiet, hospitable, simple, spacious, trustworthy.
- **Truth in presentation:** Clearly state that this is a private room in a host-occupied home, not an entire-home rental. Never imply amenities, views, pricing, or services that are not confirmed.
- Photography in the approved mockups is **placeholder** imagery. Replace with licensed actual property photos before launch. Preserve crop/focal-point configuration so replacement is simple.
- Brand name: `佐久の家` / `SAKU NO IE`. Other facilities may later use different names, assets, and capacity settings.

## 2. Reference screens

Approved concepts (2026-11-11): desktop 4-panel mockup and mobile 4-phone mockup covering (1) homepage, (2) facility/room details, (3) guest registration, (4) admin guest ledger. Mockups are directional, **not** pixel-perfect specifications. Their fictional dates, reservation IDs, counts, and photography are not production data.

### Design principles
1. Show the property before explaining it: prominent room photo and a short, clear promise.
2. One primary action per view; public-site primary CTA leads to Airbnb.
3. Guest registration is mobile-first and broken into manageable steps.
4. Administrative information density increases on desktop, while mobile remains usable.
5. Warm neutrals and terracotta for public-facing surfaces; deep green for admin identity and secondary emphasis.
6. Explicit, readable labels and status messages; do not rely on color alone.

## 3. Design tokens

### Color palette

| Token | Hex | Use |
|---|---|---|
| `color.canvas` | `#F7F4EE` | Warm page background |
| `color.surface` | `#FFFFFF` | Cards, forms, content panels |
| `color.surfaceMuted` | `#F0EDE6` | Subtle fills, chips, notes |
| `color.terracotta` | `#B05D3F` | Primary booking/next buttons, active tab underline |
| `color.terracottaHover` | `#934B32` | Primary hover/pressed |
| `color.forest` | `#214A3B` | Admin nav, headings/accent |
| `color.forestDeep` | `#17382D` | Admin nav hover/background |
| `color.ink` | `#292D2A` | Primary text |
| `color.muted` | `#626B65` | Secondary text |
| `color.border` | `#DDDCD5` | Borders and separators |
| `color.success` | `#34734E` | Success indicators with text |
| `color.warning` | `#A86B17` | Incomplete registrations with text |
| `color.error` | `#B53B36` | Validation and failures |

Ensure WCAG AA contrast for all real text/control combinations; adjust token shades if necessary. White text on terracotta must be checked, not assumed accessible. Avoid color as the only indicator of status.

### Typography
- Japanese: `Noto Sans JP` (or available equivalent) for body/UI; optional `Noto Serif JP` for large editorial headlines.
- Latin: `Inter` or system sans; English hero can use a restrained serif if consistent.
- Base text: 16px mobile and desktop; body line height 1.6–1.8 in Japanese.
- Mobile hero: 30–36px; desktop hero: 44–56px; page title: 28–36px desktop / 24–28px mobile.
- Form labels: 14–16px, semibold; helper text >= 13px where feasible.
- Avoid tiny text over photography; use a dark gradient/scrim where necessary.

### Layout, shape, spacing
- Spacing scale: 4, 8, 12, 16, 24, 32, 48, 64, 96px.
- Content max width: 1200px public site; 760px registration form; admin adapts to full available width.
- Public sections: 64–96px vertical desktop; 40–56px mobile.
- Card radius: 12–16px; button/input radius: 8–12px; subtle shadow only.
- Input/control minimum target: 44px height; touch target >= 44x44px.
- Photos: hero desktop 16:9 or wide crop; mobile portrait 4:5/3:4; room gallery 4:3/3:2. Store `alt`, aspect ratio and focal point metadata.
- Breakpoints: mobile < 768px; tablet 768–1023px; desktop >= 1024px. Use Mantine breakpoints consistently; avoid fixed viewport-height sections on small devices.

## 4. Shared components

- `SiteHeader`: logo, facility nav, locale selector (JA / EN / 简体中文 / 한국어), Airbnb CTA, mobile menu.
- `SiteFooter`: facility details, access, policies, registration link, survey link, legal/privacy links.
- `PrimaryCTA`: terracotta, strong focus ring, clear external-link semantics for Airbnb.
- `SectionHeading`: eyebrow, heading, optional short intro.
- `PhotoHero`, `PhotoGallery`, `AmenityChip`, `FeatureCard`, `InfoBanner`.
- `LocaleSwitcher`: preserves current route when translated route exists; otherwise safe locale landing page.
- `FormStepper`, `GuestCard`, `FormField`, `ValidationSummary`, `ConsentPanel`, `RegistrationStatus`.
- `AdminShell`, `MetricCard`, `FacilityFilter`, `LedgerTable`, `StatusChip`, `ExportAction`, `ConfirmDialog`.

All components need loading, empty, error, success, disabled and keyboard-focus states as relevant. Prefer native Mantine components and accessible semantics; avoid bespoke controls without need.

## 5. Screen specifications

### 5.1 Homepage — public

**Goal:** communicate the accommodation's appeal and drive verified booking to Airbnb.

**Desktop:** white header; large photographic hero with terracotta/cream room image, headline overlay, short description, prominent `宿泊を予約 (Airbnb)` button and secondary `施設について詳しく見る`; below: three benefits with line icons, room/equipment previews, shared-space section, ways to spend time, access/nearby travel, booking CTA, footer.

**Mobile:** logo + locale + hamburger in compact header; edge-to-edge room hero with legible text overlay; stacked full-width booking and details CTAs; three compact benefit tiles; vertical room photo cards; short sections with clear rhythm. Keep Airbnb CTA visible near the top; avoid intrusive sticky overlays.

Suggested Japanese hero copy: **「佐久で、暮らすように泊まる。」** Subcopy: **「ホテルライクな個室と、あたたかな交流があるホームステイの宿。」** Verify this matches actual service before launch.

**Important:** explain shared areas and host occupancy clearly; do not imply an entire-house rental. Airbnb remains the booking system; no local checkout/payment UI.

### 5.2 Facility details — public

**Desktop:** heading and introductory copy; photo mosaic (large guest room image + 3 supporting images), room description, amenities, shared-space descriptions, house rules, access and booking CTA.

**Mobile:** tabs or anchor navigation (`客室`, `共用スペース`, `設備・アメニティ`), swipeable room gallery with visible image position and arrows, amenity chips in two columns, supporting image cards and clear shared/private distinction.

Use facility-specific content data (name, max occupancy, photo list, room amenities, check-in information). Gallery must support alt text, keyboard controls and reduced-motion preference. All photos remain placeholders until real assets are provided.

### 5.3 Guest registration — guest portal

**Goal:** representative submits lodging details and all guests in one flow, with later additions/corrections permitted through a protected link.

Four-step mobile-first wizard:

1. **宿泊情報:** facility name (preselected from facility route), check-in/out dates, estimated arrival, planned guest count, optional booking source/reference. Show facility-specific capacity; do not hard-code 2 guests.
2. **代表者情報:** full name, kana if applicable, address, contact information, domestic address status, nationality and conditional passport fields as required by applicable law/policy.
3. **同行者情報:** dynamic cards for `planned_guest_count - 1` companions; `同行者を追加` when needed; allow shared address/contact shortcuts but independently confirm each person's legally required fields.
4. **確認・同意:** review by person, display applicable terms/privacy links and versions, collect explicit consent, submit once.

**Mobile:** white canvas, narrow centered content, horizontal numbered progress indicator (1–4), single-column inputs, date pickers, generous vertical spacing, full-width bottom `次へ進む` / `登録する` action (avoid keyboard overlap). **Desktop:** centered max-width 760px form; optional subtle right-side lifestyle image, but never sacrifice form width.

**Validation:** show inline error below each field and a summary at submission; retain inputs across steps. Date ranges, headcount/capacity, representative exactly once, required fields and conditional passport rules validated server-side too. Support partial completion when actual registered count is below planned count, with an explicit warning and a follow-up action; do not silently call it complete.

**Completion screen:** registration receipt/summary (no unnecessarily exposed sensitive personal details), `登録内容を確認・訂正`, `同行者を追加`, and instructions for safely keeping the edit URL. The edit URL expires at checkout in facility local time; a host can reissue. Do not expose passport numbers or full personal details merely because a URL token is present; require appropriate additional verification for sensitive edits.

**Error/edge states:** expired/invalid token, session lost, duplicate submit, facility full, dates invalid, network failure, incomplete guest count, and save success. Display actionable recovery guidance, not raw backend errors.

### 5.4 Admin guest ledger

**Goal:** securely review registrations, identify missing guests, confirm identity, correct data and export CSV.

**Desktop:** forest-green left sidebar with dashboard, guest ledger, filters/search, CSV export, survey analytics, facility settings, account. Main white area: page heading, facility selector, date range, search by guest name/reference, summary metrics (registrations, registered guests, incomplete registrations, identity verified), ledger table, row actions.

**Mobile:** forest-green top bar; compact filter controls and 2–3 KPI tiles; guest registrations shown as stacked list cards with dates, representative, planned/registered count, identity-check state and overflow actions; bottom navigation is optional for the 4–5 core sections, but avoid duplicating menu routes.

**Table columns:** stay dates, facility, optional booking reference, representative, planned headcount, registered headcount, identity verification, registration completeness, actions. `未登録あり` / `確認済み` etc. use text plus icon, not color alone. Search and CSV exports must respect authenticated facility access rights. Sensitive fields masked by default; export action should require appropriate permission and be audited.

**Admin states:** empty date range, no matching results, partial registrations, verification pending, verification complete, CSV export preparing/failure/success, insufficient permission. Never display real guest PII in design fixtures/screenshots.

### 5.5 Additional pages, same design system

- **Guide / FAQ:** searchable or categorized, clear answers for Wi-Fi, shared facilities, laundry, arrival and check-in; content from version-controlled Markdown/JSON.
- **Rates:** indicative pricing only, with disclaimer that Airbnb listing determines actual available rates; direct Airbnb CTA.
- **Terms / Privacy:** readable multilingual legal content with version and effective date; preserve consent-version references.
- **Survey:** anonymous, no registration linkage; 10 existing questions, mobile-friendly rating and choice controls, optional text answers, completion confirmation.
- **Guest registration management:** protected edit flow visually consistent with registration wizard; show only necessary summary until adequate verification.

## 6. Responsive and interaction rules

| Element | Mobile | Desktop |
|---|---|---|
| Main navigation | hamburger + locale | full nav + Airbnb CTA |
| Homepage hero | portrait crop, stacked CTAs | wide photo, overlay content |
| Facility gallery | swipeable carousel | mosaic/grid + lightbox |
| Guest form | one column, stepper | centered one/two-column as appropriate |
| Admin navigation | compact header/cards | persistent sidebar/table |
| Filters | stacked/collapsible | inline row |
| Data density | concise cards | table and summary metrics |

- Honor safe areas on iOS/Android, text scaling, 200% zoom, screen reader navigation and reduced motion.
- Locale may change text length substantially. Do not bake text into images; test English, Chinese and Korean at narrow widths.
- Date/time and local timezone shown unambiguously; store canonical data independently from locale-specific display.
- Skeleton loading for galleries/cards; descriptive empty states; image loading should avoid layout shifts.
- Keep forms recoverable on back navigation; prevent accidental loss of entered data.

## 7. Multi-facility branding

MVP is `facilities`-centric, not `organizations`-centric. Facility configuration supplies logo/name, copy, image set, capacity, amenities, booking URL, contact and policy references. Use the same core UI components across facilities. Support a **small optional theme override** per facility (hero photo, accent color) without duplicating page layouts or business logic. Keep administrative facility access scoped server-side. Future organization management is outside MVP.

## 8. Accessibility, privacy, and quality acceptance

- Target WCAG 2.2 AA: keyboard access, visible focus, labels, meaningful alt text, readable contrast, proper error announcements, no color-only status.
- Registration is a high-trust flow: no third-party analytics on PII form screens without review; never place passport numbers in URLs, logs, analytics events or screenshots.
- Use explicit confirmation for destructive/admin actions. Prevent duplicate submissions and preserve data on retry.
- Validate at widths 320, 375, 390, 768, 1024 and 1440px, including all four locales.
- Test representative-only and multiple-companion registrations, incomplete headcount, post-submission addition/edit, expired link, and facility capacity changes.
- Confirm all legal text, retention and identity-verification rules with applicable requirements before production; visual mockups do not establish legal compliance.

## 9. Design handoff checklist

- [ ] Replace placeholder images with actual approved photographs; set crop/focal points and alt text.
- [ ] Confirm final logo, facility name transliterations, copy, Airbnb booking URL and room/amenity claims.
- [ ] Check accessible color contrast, especially white text on terracotta.
- [ ] Review all four locales and responsive screens.
- [ ] Finalize terms/privacy copy and guest identity-verification UX.
- [ ] Ensure admin and registration states use realistic **fictional** test data only.
- [ ] Keep design tokens and shared components centralized in the eventual Mantine theme.

## 10. Out of scope

No local reservation engine, payment checkout, cancellation management, organization SaaS UI, or implementation code in this design deliverable. Hosting remains undecided. This document is a visual and interaction contract, not a substitute for the DB/API specifications in `CLAUDE.md`.
