import { z } from 'zod';

const isHttpUrl = (value: string) => {
  try {
    const { protocol } = new URL(value);
    return protocol === 'http:' || protocol === 'https:';
  } catch {
    return false;
  }
};

/**
 * A URL from the API that ends up in an `href` or a `src`. Anything that is not an absolute http(s) URL
 * (`javascript:`, `data:`, a relative path, an empty string) becomes null, so the place it belongs to is
 * still shown, only without the link or image. Use it for every URL a provider or a member could have
 * influenced.
 */
export const httpUrlOrNull = z
  .string()
  .nullable()
  .transform((value) => (value !== null && isHttpUrl(value) ? value : null));
