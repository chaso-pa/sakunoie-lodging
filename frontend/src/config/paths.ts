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
  // Guest community (invite-gated). One entry per bottom navigation tab.
  community: {
    feed: {
      getHref: () => '/community'
    },
    map: {
      getHref: () => '/community/map'
    },
    post: {
      getHref: () => '/community/post'
    },
    trips: {
      getHref: () => '/community/trips'
    },
    profile: {
      getHref: () => '/community/profile'
    },
    ai: {
      getHref: () => '/community/ai'
    }
  }
} as const;
