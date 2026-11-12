/**
 * Security headers for page responses, applied by `src/proxy.ts`.
 * Pure functions so the policy can be unit-tested without a Next runtime.
 */

const FIREBASE_AUTH_ORIGINS = ['https://identitytoolkit.googleapis.com', 'https://securetoken.googleapis.com'];

type CspOptions = {
  nonce: string;
  isDev: boolean;
  isAdmin: boolean;
  /** API origin from `NEXT_PUBLIC_API_BASE_URL` (cross-origin mode); allowed in `connect-src` on every page. */
  apiOrigin?: string;
  /** `host:port` of the Firebase Auth Emulator, when used (dev/E2E). */
  firebaseAuthEmulatorHost?: string;
};

export const isAdminPath = (pathname: string) => pathname === '/admin' || pathname.startsWith('/admin/');

/**
 * Guest-registration pages carry an edit token or PII: no referrer, no indexing. The page policy also
 * covers their cross-origin API calls, which therefore carry no `Referer`.
 */
export const isSensitivePath = (pathname: string) =>
  isAdminPath(pathname) || /^\/[^/]+\/(registration|stays)(\/|$)/.test(pathname);

export const buildCsp = ({ nonce, isDev, isAdmin, apiOrigin, firebaseAuthEmulatorHost }: CspOptions) => {
  const connectSrc = ["'self'"];
  if (apiOrigin) connectSrc.push(apiOrigin);
  if (isAdmin) {
    if (firebaseAuthEmulatorHost) {
      connectSrc.push(`http://${firebaseAuthEmulatorHost}`);
    } else {
      connectSrc.push(...FIREBASE_AUTH_ORIGINS);
    }
  }
  const directives = [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${isDev ? " 'unsafe-eval'" : ''}`,
    // Mantine injects inline <style> tags for its CSS variables.
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: blob:",
    "font-src 'self'",
    `connect-src ${connectSrc.join(' ')}`,
    "frame-ancestors 'none'",
    "object-src 'none'",
    "base-uri 'none'",
    "form-action 'self'"
  ];
  return directives.join('; ');
};

/** Response headers for a page, besides the CSP. */
export const buildPageHeaders = (pathname: string): Record<string, string> => {
  const headers: Record<string, string> = {
    'X-Content-Type-Options': 'nosniff',
    'Referrer-Policy': 'strict-origin-when-cross-origin'
  };
  if (isSensitivePath(pathname)) {
    headers['Referrer-Policy'] = 'no-referrer';
    headers['X-Robots-Tag'] = 'noindex, nofollow';
  }
  return headers;
};

/** Base64 of 16 random bytes; matches Next's nonce pattern `[A-Za-z0-9+/_-]+={0,2}`. */
export const createNonce = () => {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return btoa(String.fromCharCode(...bytes));
};
