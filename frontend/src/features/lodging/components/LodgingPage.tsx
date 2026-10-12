import { useLocale, useTranslations } from 'next-intl';
import { LodgingHeader } from '@/components/LodgingHeader';
import classes from '@/features/lodging/components/LodgingContent.module.scss';

type LodgingPageProps = {
  /** Document title in Japanese, as in the source page. */
  title: string;
  children: React.ReactNode;
};

/**
 * Frame for one lodging document. The body is the guesthouse's Japanese text and is never machine translated;
 * only the header chrome follows the display language.
 */
export function LodgingPage({ title, children }: LodgingPageProps) {
  const locale = useLocale();
  const t = useTranslations('lodging');

  return (
    <>
      <LodgingHeader title={title} titleLang='ja' />
      {locale !== 'ja' && <p className={classes.notice}>{t('japaneseOnly')}</p>}
      <main className={classes.content} lang='ja'>
        {children}
      </main>
    </>
  );
}
