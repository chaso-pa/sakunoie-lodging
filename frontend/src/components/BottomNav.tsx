'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import classes from '@/components/Community.module.scss';
import { paths } from '@/config/paths';

const tabs = [
  { key: 'feed', icon: '▦', href: paths.community.feed.getHref() },
  { key: 'map', icon: '⌖', href: paths.community.map.getHref() },
  { key: 'post', icon: '＋', href: paths.community.post.getHref() },
  { key: 'trips', icon: '▤', href: paths.community.trips.getHref() },
  { key: 'profile', icon: '☻', href: paths.community.profile.getHref() },
  { key: 'ai', icon: '✦', href: paths.community.ai.getHref() }
] as const;

/** The six tabs, fixed to the bottom of the phone-width column. */
export function BottomNav() {
  const t = useTranslations('community.nav');
  const pathname = usePathname();

  return (
    <nav className={classes.bottom} aria-label={t('label')}>
      {tabs.map((tab) => (
        <Link
          key={tab.key}
          href={tab.href}
          className={classes.navBtn}
          aria-current={pathname === tab.href ? 'page' : undefined}
        >
          <span aria-hidden>{tab.icon}</span>
          {t(tab.key)}
        </Link>
      ))}
    </nav>
  );
}
