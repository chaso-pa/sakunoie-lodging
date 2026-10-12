import { Title } from '@mantine/core';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { LanguageSwitcher } from '@/components/LanguageSwitcher';
import classes from '@/components/LodgingHeader.module.scss';
import { LodgingNav } from '@/components/LodgingNav';
import { paths } from '@/config/paths';

type LodgingHeaderProps = {
  title: string;
  /** Set when the title is fixed document text rather than a translated label. */
  titleLang?: string;
};

/** Header shared by the lodging information pages: site name, page title and the links between pages. */
export function LodgingHeader({ title, titleLang }: LodgingHeaderProps) {
  const t = useTranslations('common');

  return (
    <header className={classes.header}>
      <div className={classes.top}>
        <Link className={classes.site} href={paths.home.getHref()}>
          {t('siteName')}
        </Link>
        <LanguageSwitcher />
      </div>
      <Title order={1} className={classes.title} lang={titleLang}>
        {title}
      </Title>
      <LodgingNav />
    </header>
  );
}
