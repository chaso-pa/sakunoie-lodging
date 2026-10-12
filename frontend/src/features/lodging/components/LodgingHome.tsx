import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { LodgingHeader } from '@/components/LodgingHeader';
import { paths } from '@/config/paths';
import classes from '@/features/lodging/components/LodgingContent.module.scss';

export function LodgingHome() {
  const t = useTranslations('lodging');

  const links = [
    { href: paths.guide.getHref(), label: t('nav.guide'), description: t('home.guideDescription') },
    { href: paths.terms.getHref(), label: t('nav.terms'), description: t('home.termsDescription') },
    { href: paths.pricing.getHref(), label: t('nav.pricing'), description: t('home.pricingDescription') },
    { href: paths.community.feed.getHref(), label: t('nav.community'), description: t('home.communityDescription') }
  ];

  return (
    <>
      <LodgingHeader title={t('home.title')} />
      <main className={classes.content}>
        <p className={classes.lead}>{t('home.lead')}</p>
        <ul className={classes.homeList}>
          {links.map((link) => (
            <li key={link.href}>
              <Link className={classes.homeLink} href={link.href}>
                <strong>{link.label}</strong>
                <span>{link.description}</span>
              </Link>
            </li>
          ))}
        </ul>
      </main>
    </>
  );
}
