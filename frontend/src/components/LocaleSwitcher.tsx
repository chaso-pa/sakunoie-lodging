'use client';

import { NativeSelect } from '@mantine/core';
import { IconLanguage } from '@tabler/icons-react';
import { useLocale, useTranslations } from 'next-intl';
import { useTransition } from 'react';
import { usePathname, useRouter } from '@/i18n/navigation';
import { type Locale, routing } from '@/i18n/routing';

export function LocaleSwitcher() {
  const t = useTranslations('localeSwitcher');
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  return (
    <NativeSelect
      aria-label={t('label')}
      size='sm'
      leftSection={<IconLanguage size={18} aria-hidden />}
      value={locale}
      disabled={isPending}
      data={routing.locales.map((l) => ({ value: l, label: t(l) }))}
      onChange={(event) => {
        const next = event.currentTarget.value as Locale;
        // Keeps the current path; the URL fragment (edit token) is never part of `pathname`.
        startTransition(() => router.replace(pathname, { locale: next }));
      }}
    />
  );
}
