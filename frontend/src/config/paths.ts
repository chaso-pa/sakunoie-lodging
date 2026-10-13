/**
 * Centralized route path definitions.
 * All navigation paths should be defined here — no hardcoded strings elsewhere.
 */
export const paths = {
  home: {
    getHref: () => '/'
  },
  guide: {
    getHref: () => '/guide'
  },
  pricing: {
    getHref: () => '/pricing'
  },
  terms: {
    getHref: () => '/terms'
  },
  // Guest community (invite-gated). One entry per bottom navigation tab, plus the screens reached from them.
  community: {
    feed: {
      getHref: () => '/community'
    },
    map: {
      getHref: () => '/community/map'
    },
    spot: {
      getHref: (id: string) => `/community/spots/${encodeURIComponent(id)}`
    },
    post: {
      /** `spotId` opens the composer with that spot already chosen. */
      getHref: (spotId?: string) => (spotId ? `/community/post?spot=${encodeURIComponent(spotId)}` : '/community/post')
    },
    trips: {
      getHref: () => '/community/trips'
    },
    profile: {
      getHref: () => '/community/profile'
    },
    business: {
      getHref: () => '/community/profile/business'
    },
    ai: {
      getHref: () => '/community/ai'
    }
  },
  external: {
    googleMapsSearch: {
      getHref: (query: string) => `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(query)}`
    },
    osmCopyright: {
      getHref: () => 'https://www.openstreetmap.org/copyright'
    }
  }
} as const;
