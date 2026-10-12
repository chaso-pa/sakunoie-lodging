'use client';

import { Button, Title } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { LanguageSwitcher } from '@/components/LanguageSwitcher';
import classes from '@/features/auth/components/Auth.module.scss';
import { WelcomeArt } from '@/features/auth/components/WelcomeArt';

export function WelcomePage({ onStart }: { onStart: () => void }) {
  const t = useTranslations('auth');

  return (
    <div className={classes.welcomeWrap}>
      <div className={classes.welcomeTop}>
        <div className={classes.welcomeBrand}>
          {t('brand')}
          <small>{t('brandSub')}</small>
        </div>
        <LanguageSwitcher />
      </div>
      <div className={classes.welcomeArt}>
        <WelcomeArt label={t('welcome.artLabel')} />
      </div>
      <div className={classes.welcomeCopy}>
        <div className={classes.welcomeKicker}>{t('welcome.kicker')}</div>
        <Title order={1} className={classes.welcomeTitle}>
          {t('welcome.titleLine1')}
          <br />
          <em>{t('welcome.titleLine2')}</em>
        </Title>
        <p className={classes.welcomeDesc}>{t('welcome.description')}</p>
        <Button className={classes.welcomeCta} size='lg' radius={15} fullWidth onClick={onStart}>
          {t('welcome.cta')}
        </Button>
        <div className={classes.welcomeFoot}>{t('welcome.foot')}</div>
      </div>
    </div>
  );
}
