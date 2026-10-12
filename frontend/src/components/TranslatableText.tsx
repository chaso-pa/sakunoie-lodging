'use client';

import { UnstyledButton } from '@mantine/core';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';
import classes from '@/components/Community.module.scss';
import { useTranslate } from '@/hooks/use-translate';
import { isLocale, type Locale } from '@/i18n/config';
import { getApiErrorStatus } from '@/lib/api-client';

type TranslatableTextProps = {
  text: string;
  /** Language the text was written in, when recorded. Anything else lets the server detect it. */
  sourceLang?: string;
};

/**
 * Guest-written text with a "translate" button. The translation comes from the Go API and is rendered as
 * plain text. On failure the original stays in place and the reason is shown next to the button.
 */
export function TranslatableText({ text, sourceLang }: TranslatableTextProps) {
  const t = useTranslations('community.translate');
  const locale = useLocale();
  const translate = useTranslate();
  // A translation belongs to the language it was requested in; switching language shows the original again.
  const [shown, setShown] = useState<{ locale: Locale; text: string; same: boolean } | null>(null);

  const source = isLocale(sourceLang) ? sourceLang : 'auto';
  const translated = shown?.locale === locale ? shown : null;
  const canTranslate = text.trim() !== '' && source !== locale;

  const errorMessage = () => {
    if (!translate.isError || translate.variables?.target_lang !== locale) {
      return '';
    }
    switch (getApiErrorStatus(translate.error)) {
      case 503:
        return t('errors.notConfigured');
      case 502:
        return t('errors.failed');
      case 429:
        return t('errors.rateLimited');
      default:
        return t('errors.generic');
    }
  };

  const toggle = () => {
    if (translated) {
      setShown(null);
      return;
    }
    translate.mutate(
      { text: text.slice(0, 2000), source_lang: source, target_lang: locale },
      {
        onSuccess: (result) => setShown({ locale, text: result.translated_text, same: result.provider === 'none' })
      }
    );
  };

  const error = errorMessage();

  return (
    <>
      <p className={classes.body} lang={translated && !translated.same ? locale : undefined}>
        {translated ? translated.text : text}
      </p>
      {canTranslate && (
        <div className={classes.translateRow}>
          <UnstyledButton className={classes.action} disabled={translate.isPending} onClick={toggle}>
            {translate.isPending ? t('translating') : translated ? t('showOriginal') : t('button')}
          </UnstyledButton>
          <output className={classes.muted}>{translated?.same ? t('sameLanguage') : error}</output>
        </div>
      )}
    </>
  );
}
