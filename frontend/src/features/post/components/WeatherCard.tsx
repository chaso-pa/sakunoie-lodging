'use client';

import { Button } from '@mantine/core';
import { useTranslations } from 'next-intl';
import type { useWeather } from '@/features/post/api/get-weather';
import classes from '@/features/post/components/Post.module.scss';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { getApiErrorStatus } from '@/lib/api-client';

type WeatherCardProps = {
  weather: ReturnType<typeof useWeather>;
  onRefresh: () => void;
};

/** Current weather around Saku City, fetched by the API. A failure is reported here and nowhere else. */
export function WeatherCard({ weather, onRefresh }: WeatherCardProps) {
  const t = useTranslations('community.compose');
  const { weatherLabel } = useCommunityLabels();

  const text = () => {
    if (weather.isFetching) {
      return t('weatherLoading');
    }
    if (weather.isError) {
      return getApiErrorStatus(weather.error) === 429 ? t('weatherRateLimited') : t('weatherFailed');
    }
    if (weather.data) {
      return t('weatherValue', { weather: weatherLabel(weather.data.weather), temp: weather.data.temperature });
    }
    return '';
  };

  return (
    <div className={classes.weatherCard}>
      <div className={classes.weatherIcon} aria-hidden>
        {weather.data && !weather.isError ? weather.data.icon : '🌤️'}
      </div>
      <div className={classes.weatherMain}>
        <div className={classes.weatherTitle}>{t('weatherCardTitle')}</div>
        <output className={classes.weatherText}>{text()}</output>
        <div className={classes.weatherSource}>
          {weather.data && !weather.isError ? t('weatherSource') : t('weatherNote')}
        </div>
      </div>
      <Button variant='light' size='xs' loading={weather.isFetching} onClick={onRefresh}>
        {t('refresh')}
      </Button>
    </div>
  );
}
