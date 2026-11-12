import path from 'node:path';
import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';
import { parseApiBaseUrl } from './src/config/api';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

// Production (CLAUDE.md §13 "Decision B revised"): the browser calls NEXT_PUBLIC_API_BASE_URL directly
// (cross-origin, CORS allowlist on the API). Parsing it here fails the build on a malformed value.
const apiOrigin = parseApiBaseUrl(process.env.NEXT_PUBLIC_API_BASE_URL);
// Dev convenience only, when no API base URL is set: /api/* on this origin is rewritten to Go.
// Resolved when `next dev` / `next build` starts (baked into the routes manifest).
const apiProxyTarget = process.env.API_PROXY_TARGET ?? 'http://127.0.0.1:8080';

const nextConfig: NextConfig = {
  sassOptions: {
    implementation: 'sass-embedded',
    additionalData: `@use "${path.join(process.cwd(), '_mantine').replace(/\\/g, '/')}" as mantine;`
  },
  output: 'standalone',
  // Public pages read content/{locale}/*.md with fs at request time; ship them with the standalone build.
  outputFileTracingIncludes: { '/[locale]/**': ['./content/**/*'] },
  poweredByHeader: false,
  // All images are local placeholders; disabling the optimizer removes /_next/image (GHSA-2464-fhr4-w9xw surface).
  images: { unoptimized: true },
  // Next 16 blocks dev resources (HMR) for other origins; E2E uses http://127.0.0.1:3100.
  allowedDevOrigins: ['127.0.0.1'],
  experimental: {
    // Root 404 for paths outside `[locale]` (the app has more than one root layout).
    globalNotFound: true
  },
  async rewrites() {
    return apiOrigin ? [] : [{ source: '/api/:path*', destination: `${apiProxyTarget}/api/:path*` }];
  }
};

export default withNextIntl(nextConfig);
