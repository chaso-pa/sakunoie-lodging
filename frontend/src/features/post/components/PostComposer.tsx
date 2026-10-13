'use client';

import {
  Button,
  Collapse,
  FileButton,
  NativeSelect,
  NumberInput,
  Textarea,
  TextInput,
  UnstyledButton
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useRouter } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';
import shared from '@/components/Community.module.scss';
import { paths } from '@/config/paths';
import { captionMaxLength, tagsMaxLength, useCreatePost } from '@/features/post/api/create-post';
import { useLinkedSpot } from '@/features/post/api/get-linked-spot';
import { useWeather } from '@/features/post/api/get-weather';
import { useRegisterPlace } from '@/features/post/api/register-place';
import { PlacePicker } from '@/features/post/components/PlacePicker';
import classes from '@/features/post/components/Post.module.scss';
import { WeatherCard } from '@/features/post/components/WeatherCard';
import type { PlaceSelection } from '@/features/post/types/place';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { getApiErrorStatus, isUploadTooLarge } from '@/lib/api-client';
import { allWeatherValues, areaValues, isKnownArea, todayIsoDate } from '@/utils/community-labels';
import { downscaleImage, ImageError } from '@/utils/downscale-image';

/** Composer: one photo, a caption, an optional place, and the weather filled in from the API. */
export function PostComposer({ initialSpotId }: { initialSpotId?: string }) {
  const t = useTranslations('community.compose');
  const tCommon = useTranslations('community.common');
  const tErrors = useTranslations('errors');
  const { areaLabel, weatherLabel } = useCommunityLabels();
  const locale = useLocale();
  const router = useRouter();
  const linked = useLinkedSpot(initialSpotId);
  const weather = useWeather();
  const createPost = useCreatePost();
  const registerPlace = useRegisterPlace();

  const [photo, setPhoto] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [processing, setProcessing] = useState(false);
  const [photoError, setPhotoError] = useState('');
  const [caption, setCaption] = useState('');
  // undefined = untouched: the place then follows the `?spot=` link, once that spot has loaded.
  const [placeChoice, setPlaceChoice] = useState<PlaceSelection | null | undefined>(undefined);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [visitedAt, setVisitedAt] = useState(todayIsoDate);
  // null = not chosen by the member.
  const [areaChoice, setAreaChoice] = useState<string | null>(null);
  const [tags, setTags] = useState('');
  // null = "follow the automatic weather"; a value means the member chose it.
  const [weatherChoice, setWeatherChoice] = useState<string | null>(null);
  const [temperatureChoice, setTemperatureChoice] = useState<number | string | null>(null);
  const [formError, setFormError] = useState('');

  useEffect(() => {
    if (!photo) {
      setPreview(null);
      return;
    }
    const url = URL.createObjectURL(photo);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [photo]);

  // A spot whose details could not be loaded has no name to show; the link is then ignored.
  const linkedSpot = linked.data?.details_available ? linked.data : undefined;
  const place: PlaceSelection | null =
    placeChoice !== undefined
      ? placeChoice
      : linkedSpot
        ? {
            kind: 'spot',
            id: linkedSpot.id,
            name: linkedSpot.name,
            originalName: linkedSpot.original_name,
            address:
              linkedSpot.address || [areaLabel(linkedSpot.area), linkedSpot.category].filter(Boolean).join(' · '),
            emoji: linkedSpot.emoji,
            area: linkedSpot.area,
            source: linkedSpot.source
          }
        : null;

  // A spot inside the four regions brings its own area. Any other place has none unless the member picks one.
  const placeArea = place?.kind === 'spot' ? place.area : '';
  const area = placeArea || (areaChoice ?? (place ? '' : areaValues[0]));

  const auto = weather.isError ? undefined : weather.data;
  const weatherValue = weatherChoice ?? auto?.weather ?? '';
  const temperatureValue = temperatureChoice ?? auto?.temperature ?? '';
  const weatherOptions = [...new Set<string>([...allWeatherValues, ...(weatherValue ? [weatherValue] : [])])];

  const selectPhoto = async (file: File | null) => {
    if (!file) {
      return;
    }
    setPhotoError('');
    setProcessing(true);
    try {
      setPhoto(await downscaleImage(file));
    } catch (error) {
      const problem = error instanceof ImageError ? error.problem : 'unreadable';
      setPhotoError(
        problem === 'type'
          ? tCommon('imageType')
          : problem === 'tooLarge'
            ? tCommon('fileTooLarge')
            : tCommon('imageUnreadable')
      );
    } finally {
      setProcessing(false);
    }
  };

  const refreshWeather = async () => {
    const result = await weather.refetch();
    if (result.isSuccess) {
      // An explicit refresh replaces whatever was in the fields, as in the prototype.
      setWeatherChoice(null);
      setTemperatureChoice(null);
      notifications.show({ message: t('weatherUpdated') });
    }
  };

  const showFailure = (error: unknown, limitMessage: string) => {
    const status = getApiErrorStatus(error);
    if (isUploadTooLarge(error)) {
      setFormError(tCommon('fileTooLarge'));
    } else if (status === undefined) {
      setFormError(tErrors('network'));
    } else if (status === 422 || status === 400) {
      setFormError(t('errors.invalid'));
    } else if (status === 429) {
      setFormError(limitMessage);
    } else {
      setFormError(tErrors('generic'));
    }
  };

  const sharePost = (spotId: string) =>
    createPost.mutate(
      {
        caption,
        media: photo,
        spot_id: spotId,
        language: locale,
        weather: weatherValue,
        temperature: typeof temperatureValue === 'number' ? temperatureValue : null,
        visited_at: visitedAt,
        area,
        tags
      },
      {
        onSuccess: () => {
          notifications.show({ message: t('shared') });
          router.push(paths.community.feed.getHref());
        },
        onError: (error) => showFailure(error, t('errors.dailyLimit'))
      }
    );

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!photo && !caption.trim()) {
      setFormError(t('errors.empty'));
      return;
    }
    setFormError('');
    if (!place || place.kind === 'spot') {
      sharePost(place?.id ?? '');
      return;
    }
    // A place that is not a spot yet becomes one now, not when it was picked, so a place the member
    // changed their mind about is never created.
    registerPlace.mutate(
      place.kind === 'candidate'
        ? { source: place.place.source, external_id: place.place.external_id, language: locale }
        : {
            source: 'manual',
            language: locale,
            name: place.name,
            ...(place.address && { address: place.address }),
            ...(place.position && { latitude: place.position.latitude, longitude: place.position.longitude })
          },
      {
        onSuccess: (spot) => {
          // Keep the spot, so sharing again after a failed post does not register the place twice.
          setPlaceChoice({
            kind: 'spot',
            id: spot.id,
            name: spot.name || (place.kind === 'candidate' ? place.place.name : place.name),
            originalName: spot.original_name,
            address: spot.address || (place.kind === 'candidate' ? place.place.address : place.address),
            emoji: spot.emoji,
            area: spot.area,
            source: spot.source
          });
          sharePost(spot.id);
        },
        onError: (error) =>
          getApiErrorStatus(error) === 422 || getApiErrorStatus(error) === 400
            ? setFormError(t('place.errors.registerInvalid'))
            : showFailure(error, tErrors('rateLimited'))
      }
    );
  };

  return (
    <>
      <div className={shared.eyebrow}>{t('eyebrow')}</div>
      <h1 className={shared.h1}>{t('title')} 📸</h1>
      <p className={shared.muted}>{t('lead')}</p>

      <WeatherCard weather={weather} onRefresh={refreshWeather} />

      <form onSubmit={submit} noValidate>
        <div className={shared.fieldLabel} id='post-photo-label'>
          {t('photoLabel')}
        </div>
        <div className={classes.preview}>
          {preview ? (
            // biome-ignore lint/performance/noImgElement: a local object URL preview, nothing for next/image to optimise
            <img src={preview} alt={t('previewAlt')} />
          ) : (
            <span>
              ＋<br />
              {processing ? t('photoProcessing') : t('photoHint')}
            </span>
          )}
        </div>
        <div className={classes.photoButtons}>
          <FileButton onChange={selectPhoto} accept='image/*'>
            {(props) => (
              <Button {...props} variant='light' size='sm' loading={processing} aria-describedby='post-photo-label'>
                {photo ? t('changePhoto') : t('choosePhoto')}
              </Button>
            )}
          </FileButton>
          {photo && (
            <Button variant='default' size='sm' onClick={() => setPhoto(null)}>
              {t('removePhoto')}
            </Button>
          )}
        </div>
        <p className={shared.formError} role='alert'>
          {photoError}
        </p>

        <Textarea
          className={shared.field}
          label={t('captionLabel')}
          placeholder={t('captionPlaceholder')}
          size='md'
          radius='sm'
          classNames={{ label: shared.fieldLabel }}
          minRows={4}
          autosize
          maxLength={captionMaxLength}
          value={caption}
          onChange={(event) => setCaption(event.currentTarget.value)}
        />
        <PlacePicker value={place} onChange={setPlaceChoice} />

        <div className={classes.details}>
          <UnstyledButton
            className={classes.detailsToggle}
            aria-expanded={detailsOpen}
            aria-controls='post-details'
            onClick={() => setDetailsOpen((open) => !open)}
          >
            {detailsOpen ? '－' : '＋'} {t('detailsToggle')}
          </UnstyledButton>
          <Collapse expanded={detailsOpen} id='post-details'>
            <div className={shared.grid2} style={{ marginTop: 12 }}>
              <TextInput
                type='date'
                label={t('dateLabel')}
                size='md'
                radius='sm'
                classNames={{ label: shared.fieldLabel }}
                value={visitedAt}
                max={todayIsoDate()}
                onChange={(event) => setVisitedAt(event.currentTarget.value)}
              />
              <NativeSelect
                label={t('areaLabel')}
                description={placeArea ? t('areaFromSpot') : undefined}
                size='md'
                radius='sm'
                classNames={{ label: shared.fieldLabel }}
                disabled={Boolean(placeArea)}
                value={area}
                onChange={(event) => setAreaChoice(event.currentTarget.value)}
                data={[
                  ...(place && !placeArea ? [{ value: '', label: t('areaUnset') }] : []),
                  ...(isKnownArea(area) || !area ? [] : [{ value: area, label: area }]),
                  ...areaValues.map((value) => ({ value, label: areaLabel(value) }))
                ]}
              />
            </div>
            <div className={shared.grid2} style={{ marginTop: 12 }}>
              <NativeSelect
                label={t('weatherLabel')}
                size='md'
                radius='sm'
                classNames={{ label: shared.fieldLabel }}
                value={weatherValue}
                onChange={(event) => setWeatherChoice(event.currentTarget.value)}
                data={[
                  { value: '', label: weatherLabel('未設定') },
                  ...weatherOptions.map((value) => ({ value, label: weatherLabel(value) }))
                ]}
              />
              <NumberInput
                label={t('tempLabel')}
                size='md'
                radius='sm'
                classNames={{ label: shared.fieldLabel }}
                min={-30}
                max={50}
                value={temperatureValue}
                onChange={setTemperatureChoice}
              />
            </div>
            <p className={shared.muted} style={{ fontSize: 11, marginTop: 8 }}>
              {t('weatherAutoNote')}
            </p>
            <TextInput
              className={shared.field}
              label={t('tagsLabel')}
              placeholder={t('tagsPlaceholder')}
              size='md'
              radius='sm'
              classNames={{ label: shared.fieldLabel }}
              maxLength={tagsMaxLength}
              value={tags}
              onChange={(event) => setTags(event.currentTarget.value)}
            />
          </Collapse>
        </div>

        <p className={shared.formError} role='alert'>
          {formError}
        </p>
        <Button
          type='submit'
          fullWidth
          size='lg'
          loading={createPost.isPending || registerPlace.isPending}
          disabled={processing}
        >
          {t('submit')}
        </Button>
      </form>
      <p className={shared.muted} style={{ marginTop: 12 }}>
        {t('footNote')}
      </p>
    </>
  );
}
