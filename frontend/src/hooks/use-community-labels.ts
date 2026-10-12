import { useTranslations } from 'next-intl';
import { areaKeys, isKnownArea, isKnownWeather, weatherKeys } from '@/utils/community-labels';

/** Display names for the area and weather values the API stores in Japanese. */
export const useCommunityLabels = () => {
  const t = useTranslations('community');

  return {
    areaLabel: (value: string) => (isKnownArea(value) ? t(`areas.${areaKeys[value]}`) : value),
    weatherLabel: (value: string) => (isKnownWeather(value) ? t(`weather.${weatherKeys[value]}`) : value)
  };
};
