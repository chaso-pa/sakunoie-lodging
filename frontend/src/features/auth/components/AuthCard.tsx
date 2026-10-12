'use client';

import { Title } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { LanguageSwitcher } from '@/components/LanguageSwitcher';
import classes from '@/features/auth/components/Auth.module.scss';

type AuthCardProps = {
  icon: string;
  eyebrow: string;
  title: string;
  description?: string;
  /** Position in the two-step invite → profile flow. Omit for screens outside it. */
  step?: 1 | 2;
  children: React.ReactNode;
};

/** Card frame shared by the invite, signup, transfer code and restore screens. */
export function AuthCard({ icon, eyebrow, title, description, step, children }: AuthCardProps) {
  const t = useTranslations('auth');

  return (
    <div className={classes.authWrap}>
      <section className={classes.authCard}>
        <div className={classes.cardTop}>
          <div className={classes.logo}>
            {t('brand')}
            <small>{t('brandSub')}</small>
          </div>
          <LanguageSwitcher />
        </div>
        <div className={classes.icon} aria-hidden>
          {icon}
        </div>
        <div className={classes.eyebrow}>{eyebrow}</div>
        <Title order={1} className={classes.title}>
          {title}
        </Title>
        {description && <p className={classes.description}>{description}</p>}
        {step && (
          <div className={classes.steps} aria-hidden>
            <span data-on />
            <span data-on={step === 2 ? true : undefined} />
          </div>
        )}
        {children}
      </section>
    </div>
  );
}
