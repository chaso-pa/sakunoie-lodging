/**
 * Public environment variables. Only NEXT_PUBLIC_* values belong here: they are inlined into the browser bundle.
 */
export const env = {
  apiBaseUrl: (process.env.NEXT_PUBLIC_API_BASE_URL ?? '').replace(/\/+$/, '')
} as const;
