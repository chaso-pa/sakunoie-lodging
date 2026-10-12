'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { BottomNav } from '@/components/BottomNav';
import classes from '@/components/Community.module.scss';
import { LanguageSwitcher } from '@/components/LanguageSwitcher';
import { MemberAvatar } from '@/components/MemberAvatar';
import { paths } from '@/config/paths';
import { useMe } from '@/hooks/use-me';

/** Header and bottom navigation around every signed-in community screen. */
export function CommunityShell({ children }: { children: React.ReactNode }) {
  const t = useTranslations();
  const { data: member } = useMe();

  return (
    <>
      <header className={classes.top}>
        <div className={classes.brand}>
          <Link href={paths.community.feed.getHref()} className={classes.logo}>
            {t('auth.brand')}
            <small>{t('community.tagline')}</small>
          </Link>
          <div className={classes.brandActions}>
            <LanguageSwitcher w={112} />
            <Link
              href={paths.community.profile.getHref()}
              className={classes.headerAvatar}
              aria-label={t('community.header.profileLink')}
            >
              <MemberAvatar src={member?.avatar_url} size={38} />
            </Link>
          </div>
        </div>
      </header>
      <main className={classes.main}>{children}</main>
      <BottomNav />
    </>
  );
}
