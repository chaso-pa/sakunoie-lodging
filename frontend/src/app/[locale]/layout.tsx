import '@mantine/core/styles.css';
import '@/app/globals.scss';

import { Box, mantineHtmlProps } from '@mantine/core';
import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { connection } from 'next/server';
import { hasLocale, NextIntlClientProvider } from 'next-intl';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';
import { inter, notoSansJp } from '@/app/fonts';
import { AppProviders } from '@/components/AppProviders';
import { SiteFooter } from '@/components/SiteFooter';
import { SiteHeader } from '@/components/SiteHeader';
import { routing } from '@/i18n/routing';

type LayoutProps = { children: ReactNode; params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Omit<LayoutProps, 'children'>): Promise<Metadata> {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) return {};
  const t = await getTranslations({ locale, namespace: 'common' });
  return { title: { default: t('brand'), template: `%s | ${t('brand')}` } };
}

export default async function LocaleLayout({ children, params }: LayoutProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) notFound();
  setRequestLocale(locale);

  // The nonce CSP from src/proxy.ts only works on dynamically rendered pages: Next nonces its
  // inline scripts per request from the CSP request header. Server code that needs the nonce
  // can read the `x-nonce` request header.
  await connection();
  const t = await getTranslations({ locale, namespace: 'common' });

  return (
    <html lang={locale} {...mantineHtmlProps} className={`${inter.variable} ${notoSansJp.variable}`}>
      {/* No ColorSchemeScript: the scheme is forced to light and set by mantineHtmlProps. */}
      <head />
      <body>
        <a href='#main' className='skip-link'>
          {t('skipToContent')}
        </a>
        <NextIntlClientProvider>
          <AppProviders>
            <Box mih='100dvh' style={{ display: 'flex', flexDirection: 'column' }}>
              <SiteHeader />
              <Box component='main' id='main' style={{ flex: 1 }}>
                {children}
              </Box>
              <SiteFooter />
            </Box>
          </AppProviders>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
