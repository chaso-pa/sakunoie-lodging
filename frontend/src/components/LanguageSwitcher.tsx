'use client';

import { NativeSelect, type NativeSelectProps } from '@mantine/core';
import { useLocale, useTranslations } from 'next-intl';
import { useTransition } from 'react';
import { localeLabels, locales } from '@/i18n/config';
import { setLocale } from '@/i18n/set-locale';

const data = locales.map((locale) => ({ value: locale, label: localeLabels[locale] }));

type LanguageSwitcherProps = Omit<NativeSelectProps, 'data' | 'value' | 'onChange'> & {
  /** Stretch to the container instead of the compact header width. */
  fullWidth?: boolean;
};

/** Switches the display language. The choice is stored in a cookie, so it survives reloads. */
export function LanguageSwitcher({ fullWidth, label, ...props }: LanguageSwitcherProps) {
  const locale = useLocale();
  const t = useTranslations('common');
  const [isPending, startTransition] = useTransition();

  return (
    <NativeSelect
      size='xs'
      w={fullWidth ? '100%' : 128}
      label={label}
      aria-label={label ? undefined : t('languageLabel')}
      {...props}
      data={data}
      value={locale}
      disabled={isPending}
      onChange={(event) => {
        const next = event.currentTarget.value;
        startTransition(() => setLocale(next));
      }}
    />
  );
}
