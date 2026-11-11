import path from 'node:path';
import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

// Same-origin API access (plan Decision B): the browser calls /api/v1/*, Next forwards it to Go.
// Resolved when `next dev` / `next build` / `next start` starts.
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
  // Next 16 blocks dev resources (HMR) for other origins; E2E uses http://127.0.0.1:3100.
  allowedDevOrigins: ['127.0.0.1'],
  experimental: {
    // Root 404 for paths outside `[locale]` (the app has more than one root layout).
    globalNotFound: true
  },
  async rewrites() {
    return [{ source: '/api/:path*', destination: `${apiProxyTarget}/api/:path*` }];
  }
};

export default withNextIntl(nextConfig);
