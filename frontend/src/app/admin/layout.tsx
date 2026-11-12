import '@mantine/core/styles.css';
import '@/app/globals.scss';

import { mantineHtmlProps } from '@mantine/core';
import type { Metadata } from 'next';
import { connection } from 'next/server';
import type { ReactNode } from 'react';
import { inter, notoSansJp } from '@/app/fonts';
import { AppProviders } from '@/components/AppProviders';

export const metadata: Metadata = {
  title: { default: '管理画面 | 佐久の家', template: '%s | 佐久の家 管理' },
  robots: { index: false, follow: false }
};

/**
 * Second root layout: the admin console is Japanese-only and lives outside `[locale]`
 * (src/proxy.ts skips i18n routing for /admin and adds the Firebase origins to connect-src).
 */
export default async function AdminRootLayout({ children }: { children: ReactNode }) {
  // Dynamic rendering, required by the per-request nonce CSP (see [locale]/layout.tsx).
  await connection();
  return (
    <html lang='ja' {...mantineHtmlProps} className={`${inter.variable} ${notoSansJp.variable}`}>
      <head />
      <body>
        <a href='#main' className='skip-link'>
          本文へスキップ
        </a>
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
