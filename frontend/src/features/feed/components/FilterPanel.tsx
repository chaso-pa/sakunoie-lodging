'use client';

import { NativeSelect, UnstyledButton } from '@mantine/core';
import { useLocale, useTranslations } from 'next-intl';
import shared from '@/components/Community.module.scss';
import classes from '@/features/feed/components/Feed.module.scss';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { getHtmlLang } from '@/i18n/config';
import { weatherValues } from '@/utils/community-labels';

/** The last twelve months as YYYY-MM, newest first. */
const recentMonths = () => {
  const now = new Date();
  return Array.from({ length: 12 }, (_, index) => {
    const date = new Date(now.getFullYear(), now.getMonth() - index, 1);
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
  });
};

type FilterPanelProps = {
  weather: string;
  month: string;
  onWeatherChange: (value: string) => void;
  onMonthChange: (value: string) => void;
  onReset: () => void;
};

export function FilterPanel({ weather, month, onWeatherChange, onMonthChange, onReset }: FilterPanelProps) {
  const t = useTranslations('community.feed');
  const locale = useLocale();
  const { weatherLabel } = useCommunityLabels();
  const monthFormat = new Intl.DateTimeFormat(getHtmlLang(locale), { year: 'numeric', month: 'long' });

  return (
    <div className={classes.filterPanel}>
      <div className={classes.filterHeading}>🔎 {t('filterHeading')}</div>
      <div className={shared.grid2}>
        <NativeSelect
          label={t('weatherLabel')}
          size='sm'
          radius='sm'
          classNames={{ label: shared.fieldLabel }}
          value={weather}
          onChange={(event) => onWeatherChange(event.currentTarget.value)}
          data={[
            { value: '', label: t('allWeather') },
            ...weatherValues.map((value) => ({ value, label: weatherLabel(value) }))
          ]}
        />
        <NativeSelect
          label={t('monthLabel')}
          size='sm'
          radius='sm'
          classNames={{ label: shared.fieldLabel }}
          value={month}
          onChange={(event) => onMonthChange(event.currentTarget.value)}
          data={[
            { value: '', label: t('allMonths') },
            ...recentMonths().map((value) => ({
              value,
              label: monthFormat.format(new Date(Number(value.slice(0, 4)), Number(value.slice(5)) - 1, 1))
            }))
          ]}
        />
      </div>
      <p className={shared.muted} style={{ marginTop: 8 }}>
        {t('filterHint')}
      </p>
      <UnstyledButton className={shared.action} mt={9} onClick={onReset}>
        {t('reset')}
      </UnstyledButton>
    </div>
  );
}
