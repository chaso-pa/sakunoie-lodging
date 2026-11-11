/**
 * Centralized route path definitions. No hardcoded path strings elsewhere.
 * Public and registration hrefs are locale-less: pass them to the next-intl `Link`/router
 * from `@/i18n/navigation`, which adds the `/{locale}` prefix. Admin paths have no locale.
 */
export const paths = {
  public: {
    home: { getHref: () => '/' },
    facilities: { getHref: () => '/facilities' },
    guide: { getHref: () => '/guide' },
    faq: { getHref: () => '/faq' },
    rates: { getHref: () => '/rates' },
    terms: { getHref: () => '/terms' },
    privacy: { getHref: () => '/privacy' },
    survey: { getHref: () => '/survey' }
  },
  registration: {
    wizard: { getHref: (facilityCode: string) => `/stays/${encodeURIComponent(facilityCode)}/registration` },
    complete: { getHref: () => '/registration/complete' },
    manage: {
      getHref: () => '/registration/manage',
      /** Absolute edit URL. The token travels only in the fragment, which browsers never send to servers. */
      getAbsoluteHref: (origin: string, locale: string, token: string) =>
        `${origin}/${locale}/registration/manage#token=${encodeURIComponent(token)}`
    }
  },
  admin: {
    root: { getHref: () => '/admin' },
    login: { getHref: () => '/admin/login' },
    registrations: { getHref: () => '/admin/guest-registrations' },
    registration: { getHref: (publicId: string) => `/admin/guest-registrations/${encodeURIComponent(publicId)}` },
    surveys: { getHref: () => '/admin/surveys' }
  }
} as const;
