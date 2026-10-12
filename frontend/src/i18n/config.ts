export const locales = ['ja', 'en', 'ko', 'zh'] as const;
export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = 'ja';

/** The locale is a user preference kept in this cookie; URLs carry no locale prefix. */
export const localeCookieName = 'sakunoie_locale';
export const localeCookieMaxAge = 60 * 60 * 24 * 365;

/** Shown in the language switcher. Each language is named in itself, so these are not translated. */
export const localeLabels: Record<Locale, string> = {
  ja: '🇯🇵 日本語',
  en: '🇺🇸 English',
  ko: '🇰🇷 한국어',
  zh: '🇨🇳 简体中文'
};

const htmlLangs: Record<Locale, string> = { ja: 'ja', en: 'en', ko: 'ko', zh: 'zh-CN' };

export const getHtmlLang = (locale: Locale) => htmlLangs[locale];

export const isLocale = (value: unknown): value is Locale => locales.includes(value as Locale);
