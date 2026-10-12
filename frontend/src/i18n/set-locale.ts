'use server';

import { cookies } from 'next/headers';
import { isLocale, localeCookieMaxAge, localeCookieName } from '@/i18n/config';

/** Stores the display language. Setting a cookie in a server action re-renders the current route. */
export async function setLocale(locale: string) {
  if (!isLocale(locale)) {
    return;
  }
  (await cookies()).set(localeCookieName, locale, { path: '/', maxAge: localeCookieMaxAge, sameSite: 'lax' });
}
