'use client';

import { UnstyledButton } from '@mantine/core';
import { useTranslations } from 'next-intl';
import classes from '@/components/Community.module.scss';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { areaValues } from '@/utils/community-labels';

type AreaChipsProps = {
  /** Empty string means every area. */
  value: string;
  onChange: (value: string) => void;
};

export function AreaChips({ value, onChange }: AreaChipsProps) {
  const t = useTranslations('community');
  const { areaLabel } = useCommunityLabels();

  return (
    // biome-ignore lint/a11y/useSemanticElements: a fieldset cannot be the horizontally scrolling flex row this needs
    <div className={classes.chips} role='group' aria-label={t('common.areaFilterLabel')}>
      <UnstyledButton className={classes.chip} aria-pressed={value === ''} onClick={() => onChange('')}>
        {t('areas.all')}
      </UnstyledButton>
      {areaValues.map((area) => (
        <UnstyledButton
          key={area}
          className={classes.chip}
          aria-pressed={value === area}
          onClick={() => onChange(area)}
        >
          {areaLabel(area)}
        </UnstyledButton>
      ))}
    </div>
  );
}
