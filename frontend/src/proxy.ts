import { type NextRequest, NextResponse } from 'next/server';

// Headers the Go API could read a client address from.
const clientAddressHeaders = ['x-forwarded-for', 'x-real-ip'];

/**
 * Runs before the `/api/*` rewrite to the Go API (see next.config.ts).
 *
 * Next.js cannot tell this code who the peer really is: it fills `x-forwarded-for` from the socket only
 * when the client sent none, so a value seen here may be forged. These headers are therefore removed
 * before the request is proxied, and the API sees every request as coming from this server.
 *
 * Set TRUST_FORWARDED_HEADERS=true only when a reverse proxy you control sits in front of Next.js and
 * overwrites these headers itself; they are then passed through unchanged.
 */
export function proxy(request: NextRequest) {
  if (process.env.TRUST_FORWARDED_HEADERS === 'true') {
    return NextResponse.next();
  }

  const headers = new Headers(request.headers);
  for (const name of clientAddressHeaders) {
    headers.delete(name);
  }
  return NextResponse.next({ request: { headers } });
}

export const config = { matcher: '/api/:path*' };
