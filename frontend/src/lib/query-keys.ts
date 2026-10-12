/**
 * Query keys shared across features. Everything that caches posts lives under `posts`, so a like, a save
 * or a delete can update every list at once.
 */
export const queryKeys = {
  posts: {
    all: ['posts'] as const,
    feed: (filters: Record<string, string>) => ['posts', 'feed', filters] as const,
    mine: ['posts', 'mine'] as const,
    bookmarked: ['posts', 'bookmarked'] as const
  },
  rankings: {
    all: ['rankings'] as const,
    byType: (type: string) => ['rankings', type] as const
  },
  spots: {
    all: ['spots'] as const,
    list: ['spots', 'list'] as const,
    detail: (id: string) => ['spots', 'detail', id] as const
  },
  gpsVisits: {
    all: ['gps-visits'] as const,
    mine: ['gps-visits', 'mine'] as const,
    public: ['gps-visits', 'public'] as const
  },
  weather: ['weather'] as const,
  aiSpots: (filters: Record<string, string>) => ['ai-spots', filters] as const
};
