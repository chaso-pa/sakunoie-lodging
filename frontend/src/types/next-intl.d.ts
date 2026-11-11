import type messages from '../../messages/ja.json';
import type { Locale } from '@/i18n/routing';

declare module 'next-intl' {
  interface AppConfig {
    Locale: Locale;
    Messages: typeof messages;
  }
}
