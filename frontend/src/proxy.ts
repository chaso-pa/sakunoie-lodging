import { NextRequest, NextResponse } from 'next/server';
import createMiddleware from 'next-intl/middleware';
import { routing } from '@/i18n/routing';
import { buildCsp, buildPageHeaders, createNonce, isAdminPath } from '@/lib/security-headers';

const handleI18nRouting = createMiddleware(routing);

export default function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const isAdmin = isAdminPath(pathname);
  const nonce = createNonce();
  const csp = buildCsp({
    nonce,
    isDev: process.env.NODE_ENV === 'development',
    isAdmin,
    firebaseAuthEmulatorHost: process.env.NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST || undefined
  });

  // Next nonces its inline bootstrap scripts by parsing the CSP *request* header;
  // server components read `x-nonce` for Mantine's ColorSchemeScript.
  const requestHeaders = new Headers(request.headers);
  requestHeaders.set('Content-Security-Policy', csp);
  requestHeaders.set('x-nonce', nonce);

  // The admin UI is Japanese-only and lives outside `[locale]`, so it skips i18n routing.
  // next-intl copies the incoming request headers into its rewrite/next response,
  // so handing it a request that already carries the CSP forwards both headers.
  const response = isAdmin
    ? NextResponse.next({ request: { headers: requestHeaders } })
    : handleI18nRouting(new NextRequest(request, { headers: requestHeaders }));

  response.headers.set('Content-Security-Policy', csp);
  for (const [key, value] of Object.entries(buildPageHeaders(pathname))) {
    response.headers.set(key, value);
  }
  return response;
}

export const config = {
  // Everything except API rewrites, Next internals (static, image, dev HMR socket) and files with an extension.
  matcher: ['/((?!api/|_next/|.*\\..*).*)']
};
