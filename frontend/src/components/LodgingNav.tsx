'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import classes from '@/components/LodgingHeader.module.scss';
import { paths } from '@/config/paths';

export function LodgingNav() {
  const t = useTranslations('lodging.nav');
  const pathname = usePathname();

  const links = [
    { href: paths.guide.getHref(), label: t('guide') },
    { href: paths.terms.getHref(), label: t('terms') },
    { href: paths.pricing.getHref(), label: t('pricing') },
    { href: paths.community.feed.getHref(), label: t('community') }
  ];

  return (
    <nav className={classes.nav} aria-label={t('label')}>
      {links.map((link) => (
        <Link key={link.href} href={link.href} aria-current={pathname === link.href ? 'page' : undefined}>
          {link.label}
        </Link>
      ))}
    </nav>
  );
}
