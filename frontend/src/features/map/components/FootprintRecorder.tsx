'use client';

import { Button, Checkbox, NativeSelect, NumberInput, Textarea, TextInput } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import shared from '@/components/Community.module.scss';
import { paths } from '@/config/paths';
import { useCreateGpsVisit, visitNoteMaxLength, visitPlaceMaxLength } from '@/features/map/api/create-gps-visit';
import classes from '@/features/map/components/Map.module.scss';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { getApiErrorStatus } from '@/lib/api-client';
import type { SpotListItem } from '@/types/community';
import { todayIsoDate, weatherValues } from '@/utils/community-labels';

type Position = { latitude: number; longitude: number; accuracy: number };
type LocateStatus = 'idle' | 'locating' | 'located' | 'denied' | 'unavailable' | 'timeout' | 'unsupported';

const customPlace = 'custom';

/**
 * "Record a footprint" card. The browser is asked for the position only when the member taps the
 * locate button, and the form opens whatever the answer is, so a refusal never blocks recording.
 * Both visibility switches start off.
 */
export function FootprintRecorder({ spots }: { spots: SpotListItem[] }) {
  const t = useTranslations('community.map.footprint');
  const tErrors = useTranslations('errors');
  const { weatherLabel } = useCommunityLabels();
  const router = useRouter();
  const createVisit = useCreateGpsVisit();

  const [status, setStatus] = useState<LocateStatus>('idle');
  const [position, setPosition] = useState<Position | null>(null);
  const [formOpen, setFormOpen] = useState(false);

  const [place, setPlace] = useState('');
  const [customName, setCustomName] = useState('');
  const [date, setDate] = useState(todayIsoDate);
  const [weather, setWeather] = useState('');
  const [temperature, setTemperature] = useState<number | string>('');
  const [note, setNote] = useState('');
  const [isPublic, setIsPublic] = useState(false);
  const [exactPublic, setExactPublic] = useState(false);
  const [formError, setFormError] = useState('');

  const selectedPlace = place || spots[0]?.id || customPlace;

  const locate = () => {
    if (!('geolocation' in navigator)) {
      setStatus('unsupported');
      setFormOpen(true);
      return;
    }
    setStatus('locating');
    navigator.geolocation.getCurrentPosition(
      (result) => {
        setPosition({
          latitude: result.coords.latitude,
          longitude: result.coords.longitude,
          accuracy: result.coords.accuracy
        });
        setStatus('located');
        setFormOpen(true);
      },
      (error) => {
        setPosition(null);
        setExactPublic(false);
        setStatus(
          error.code === error.PERMISSION_DENIED
            ? 'denied'
            : error.code === error.POSITION_UNAVAILABLE
              ? 'unavailable'
              : 'timeout'
        );
        setFormOpen(true);
      },
      { enableHighAccuracy: true, timeout: 10000, maximumAge: 60000 }
    );
  };

  const statusMessage = () => {
    switch (status) {
      case 'locating':
        return t('statusLocating');
      case 'located':
        return t('statusLocated');
      case 'denied':
        return t('statusDenied');
      case 'unavailable':
        return t('statusUnavailable');
      case 'timeout':
        return t('statusTimeout');
      case 'unsupported':
        return t('statusUnsupported');
      default:
        return '';
    }
  };

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    const isCustom = selectedPlace === customPlace;
    const placeName = customName.trim();
    if (isCustom && !placeName) {
      setFormError(t('errors.placeRequired'));
      return;
    }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) {
      setFormError(t('errors.dateRequired'));
      return;
    }
    setFormError('');
    createVisit.mutate(
      {
        ...(isCustom ? { place_name: placeName } : { spot_id: selectedPlace }),
        visited_date: date,
        ...(weather && { weather }),
        ...(typeof temperature === 'number' && { temperature }),
        ...(note.trim() && { note: note.trim() }),
        is_public: isPublic,
        exact_public: Boolean(position) && exactPublic,
        ...(position && { latitude: position.latitude, longitude: position.longitude })
      },
      {
        onSuccess: () => {
          notifications.show({ message: t('saved') });
          router.push(paths.community.trips.getHref());
        },
        onError: (error) => {
          const code = getApiErrorStatus(error);
          setFormError(
            code === undefined
              ? tErrors('network')
              : code === 422 || code === 400
                ? t('errors.invalid')
                : code === 429
                  ? tErrors('rateLimited')
                  : tErrors('generic')
          );
        }
      }
    );
  };

  return (
    <div className={shared.card}>
      <div className={shared.cardBody}>
        <span className={shared.pill}>📍 {t('pill')}</span>
        <h2 className={shared.h2} style={{ marginTop: 10 }}>
          {t('title')}
        </h2>
        <p className={shared.muted}>{t('body')}</p>
        <Button fullWidth mt={10} size='md' loading={status === 'locating'} onClick={locate}>
          📍 {t('locate')}
        </Button>
        <Button
          fullWidth
          mt={10}
          size='md'
          variant='light'
          aria-expanded={formOpen}
          onClick={() => setFormOpen((open) => !open)}
        >
          {t('manual')}
        </Button>
        <output className={shared.muted} style={{ display: 'block', marginTop: 10 }}>
          {statusMessage()}
        </output>
        {position && <p className={shared.weather}>{t('located', { meters: Math.round(position.accuracy || 0) })}</p>}

        {formOpen && (
          <form onSubmit={submit} noValidate>
            <h3 className={shared.h3} style={{ marginTop: 12 }}>
              {t('formTitle')}
            </h3>
            <NativeSelect
              className={shared.field}
              label={t('spotLabel')}
              size='md'
              radius='sm'
              classNames={{ label: shared.fieldLabel }}
              value={selectedPlace}
              onChange={(event) => setPlace(event.currentTarget.value)}
              data={[
                ...spots.map((spot) => ({ value: spot.id, label: `${spot.emoji} ${spot.name}` })),
                { value: customPlace, label: t('customOption') }
              ]}
            />
            {selectedPlace === customPlace && (
              <TextInput
                className={shared.field}
                label={t('customLabel')}
                placeholder={t('customPlaceholder')}
                size='md'
                radius='sm'
                classNames={{ label: shared.fieldLabel }}
                maxLength={visitPlaceMaxLength}
                value={customName}
                onChange={(event) => setCustomName(event.currentTarget.value)}
              />
            )}
            <div className={shared.grid2}>
              <TextInput
                type='date'
                label={t('dateLabel')}
                size='md'
                radius='sm'
                classNames={{ label: shared.fieldLabel }}
                required
                value={date}
                max={todayIsoDate()}
                onChange={(event) => setDate(event.currentTarget.value)}
              />
              <NativeSelect
                label={t('weatherLabel')}
                size='md'
                radius='sm'
                classNames={{ label: shared.fieldLabel }}
                value={weather}
                onChange={(event) => setWeather(event.currentTarget.value)}
                data={[
                  { value: '', label: weatherLabel('未設定') },
                  ...weatherValues.map((value) => ({ value, label: weatherLabel(value) }))
                ]}
              />
            </div>
            <NumberInput
              className={shared.field}
              label={t('tempLabel')}
              placeholder={t('tempPlaceholder')}
              size='md'
              radius='sm'
              classNames={{ label: shared.fieldLabel }}
              min={-30}
              max={50}
              value={temperature}
              onChange={setTemperature}
            />
            <Textarea
              className={shared.field}
              label={t('noteLabel')}
              placeholder={t('notePlaceholder')}
              size='md'
              radius='sm'
              classNames={{ label: shared.fieldLabel }}
              minRows={3}
              autosize
              maxLength={visitNoteMaxLength}
              value={note}
              onChange={(event) => setNote(event.currentTarget.value)}
            />
            <Checkbox
              className={classes.check}
              size='sm'
              label={t('publicLabel')}
              description={t('publicHint')}
              checked={isPublic}
              onChange={(event) => setIsPublic(event.currentTarget.checked)}
            />
            <Checkbox
              className={classes.check}
              size='sm'
              label={t('exactLabel')}
              description={position ? t('exactHint') : t('exactNoPosition')}
              checked={Boolean(position) && exactPublic}
              disabled={!position}
              onChange={(event) => setExactPublic(event.currentTarget.checked)}
            />
            <p className={shared.formError} role='alert'>
              {formError}
            </p>
            <Button type='submit' fullWidth mt={10} size='md' loading={createVisit.isPending}>
              {t('submit')}
            </Button>
          </form>
        )}
      </div>
    </div>
  );
}
