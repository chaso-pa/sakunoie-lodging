'use client';

import { useTranslations } from 'next-intl';
import classes from '@/components/Community.module.scss';

/** Marks seeded demo rows (`is_sample` from the API) so they are not mistaken for a guest's content. */
export function SampleBadge() {
  const t = useTranslations('community.common');
  return <span className={classes.sampleBadge}>{t('sampleBadge')}</span>;
}
