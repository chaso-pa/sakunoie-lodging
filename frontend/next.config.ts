import path from 'node:path';
import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

const apiProxyTarget = (process.env.API_PROXY_TARGET || 'http://localhost:8080').replace(/\/+$/, '');

const nextConfig: NextConfig = {
  sassOptions: {
    implementation: 'sass-embedded',
    additionalData: `@use "${path.join(process.cwd(), '_mantine').replace(/\\/g, '/')}" as mantine;`
  },
  output: 'standalone',
  experimental: {
    // src/proxy.ts makes Next.js buffer /api/* request bodies, 10MB by default, and anything larger gets an
    // opaque 500 from Next.js. The Go API enforces the real 10MB upload cap and answers 422 with a structured
    // error, so the buffer is set above that cap to let the API be the one that answers.
    proxyClientMaxBodySize: '12mb'
  },
  // Proxy the Go API so the session cookie is same-origin (plan fork B1).
  async rewrites() {
    return [{ source: '/api/:path*', destination: `${apiProxyTarget}/api/:path*` }];
  }
};

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

export default withNextIntl(nextConfig);
