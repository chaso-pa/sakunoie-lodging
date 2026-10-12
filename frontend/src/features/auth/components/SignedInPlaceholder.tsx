'use client';

import { Avatar, Title } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { LanguageSwitcher } from '@/components/LanguageSwitcher';
import { useMe } from '@/features/auth/api/use-me';
import classes from '@/features/auth/components/Auth.module.scss';
import { LogoutButton } from '@/features/auth/components/LogoutButton';

/** Temporary community home: only proves the member is signed in. The six-tab UI replaces it. */
export function SignedInPlaceholder() {
  const t = useTranslations('auth');
  const tCommunity = useTranslations('community');
  const { data: member } = useMe();

  if (!member) {
    return null;
  }

  return (
    <div className={classes.signedIn}>
      <div className={classes.signedInTop}>
        <div className={classes.logo}>
          {t('brand')}
          <small>{tCommunity('tagline')}</small>
        </div>
        <LanguageSwitcher />
      </div>
      <div className={classes.hero}>
        <Avatar className={classes.avatar} src={member.avatar_url || null} alt='' size={72} radius='50%' mx='auto'>
          🌿
        </Avatar>
        <div className={classes.eyebrow} style={{ marginTop: 12 }}>
          {t('signedIn.eyebrow')}
        </div>
        <Title order={1} className={classes.title}>
          {t('signedIn.title', { nickname: member.nickname })}
        </Title>
        <p className={classes.description}>{t('signedIn.description')}</p>
      </div>
      <LogoutButton fullWidth size='md' />
    </div>
  );
}
