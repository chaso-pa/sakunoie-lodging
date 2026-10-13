'use client';

import {
  Button,
  Collapse,
  FileButton,
  NativeSelect,
  NumberInput,
  Textarea,
  TextInput,
  UnstyledButton,
  VisuallyHidden
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useRouter } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';
import shared from '@/components/Community.module.scss';
import { paths } from '@/config/paths';
import { captionMaxLength, postPhotoLimit, tagsMaxLength, useCreatePost } from '@/features/post/api/create-post';
import { useLinkedSpot } from '@/features/post/api/get-linked-spot';
import { useWeather } from '@/features/post/api/get-weather';
import { useRegisterPlace } from '@/features/post/api/register-place';
import { PlacePicker } from '@/features/post/components/PlacePicker';
import classes from '@/features/post/components/Post.module.scss';
import { WeatherCard } from '@/features/post/components/WeatherCard';
import type { PlaceSelection } from '@/features/post/types/place';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { getApiErrorStatus, getPostPhotoProblem } from '@/lib/api-client';
import { allWeatherValues, areaValues, isKnownArea, todayIsoDate } from '@/utils/community-labels';
import {
  downscaleImage,
  ImageError,
  type ImageProblem,
  postPhotoMaxBytes,
  uploadMaxBytes
} from '@/utils/downscale-image';

/** A chosen photo, already downscaled, with the object URL of its thumbnail. */
type ComposerPhoto = { id: number; file: File; url: string };

/** Composer: up to five photos, a caption, an optional place, and the weather filled in from the API. */
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

  const [photos, setPhotos] = useState<ComposerPhoto[]>([]);
  // Set while chosen files are being downscaled: how many of them are done.
  const [progress, setProgress] = useState<{ current: number; total: number } | null>(null);
  const [photoError, setPhotoError] = useState('');
  // Read out by screen readers when photos are added, moved or removed.
  const [photoStatus, setPhotoStatus] = useState('');
  const nextPhotoId = useRef(0);
  const resetPicker = useRef<() => void>(null);
  const pickerButton = useRef<HTMLButtonElement>(null);
  const thumbs = useRef<HTMLOListElement>(null);
  // The control to focus once the list has been drawn again: an arrow of the photo that moved, or the
  // picker after a removal.
  const focusAfterChange = useRef<{ id: number; action: 'earlier' | 'later' } | 'picker' | null>(null);
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

  // Each thumbnail's object URL is revoked when its photo is removed, and the rest when the composer
  // goes away (after sharing, or on leaving the page).
  const photosRef = useRef(photos);
  useEffect(() => {
    photosRef.current = photos;
  }, [photos]);
  useEffect(
    () => () => {
      for (const photo of photosRef.current) {
        URL.revokeObjectURL(photo.url);
      }
    },
    []
  );

  // Moving or removing a photo redraws the list, and the button that was pressed may be gone or disabled.
  // Keep the keyboard on the photo that moved (the same arrow, or the opposite one at an end); after a
  // removal hand it to the picker, which is enabled again by then.
  // biome-ignore lint/correctness/useExhaustiveDependencies: runs after every change of the list
  useEffect(() => {
    const target = focusAfterChange.current;
    focusAfterChange.current = null;
    if (target === 'picker') {
      pickerButton.current?.focus();
    } else if (target && thumbs.current) {
      const controls = thumbs.current.querySelectorAll<HTMLButtonElement>(`[data-photo="${target.id}"][data-move]`);
      const enabled = [...controls].filter((control) => !control.disabled);
      (enabled.find((control) => control.dataset.move === target.action) ?? enabled[0])?.focus();
    }
  }, [photos]);

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

  const processing = progress !== null;
  const full = photos.length >= postPhotoLimit;

  const imageProblemMessage = (problem: ImageProblem) =>
    problem === 'type'
      ? tCommon('imageType')
      : problem === 'tooLarge'
        ? tCommon('fileTooLarge')
        : tCommon('imageUnreadable');

  /**
   * Adds the chosen files in the order the picker gave them, until the post is full. A file that cannot
   * be used does not take a slot. They are downscaled one after another: one decoded phone photo in
   * memory at a time, and the page stays responsive and shows each thumbnail as soon as it is ready.
   */
  const addPhotos = async (files: File[]) => {
    // Lets the same file be picked again later, after it was removed.
    resetPicker.current?.();
    const room = postPhotoLimit - photos.length;
    if (files.length === 0 || room <= 0) {
      return;
    }
    const expected = Math.min(files.length, room);
    setPhotoError('');
    setProgress({ current: 1, total: expected });

    let totalBytes = photos.reduce((sum, photo) => sum + photo.file.size, 0);
    let added = 0;
    let leftOut = 0;
    let overTotal = false;
    const problems: ImageProblem[] = [];
    for (const [index, file] of files.entries()) {
      if (added >= room) {
        leftOut = files.length - index;
        break;
      }
      setProgress({ current: Math.min(added + 1, expected), total: expected });
      try {
        const prepared = await downscaleImage(file, postPhotoMaxBytes);
        if (totalBytes + prepared.size > uploadMaxBytes) {
          overTotal = true;
          break;
        }
        totalBytes += prepared.size;
        added += 1;
        const photo = { id: nextPhotoId.current++, file: prepared, url: URL.createObjectURL(prepared) };
        setPhotos((current) => [...current, photo]);
      } catch (error) {
        problems.push(error instanceof ImageError ? error.problem : 'unreadable');
      }
    }
    setProgress(null);

    const notices: string[] = [];
    if (leftOut > 0) {
      notices.push(t('photosTrimmed', { max: postPhotoLimit, added }));
    }
    if (overTotal) {
      notices.push(t('errors.totalTooLarge'));
    }
    if (problems.length > 0) {
      // One file chosen: just say what is wrong with it. Several: say how many were left out, and why.
      if (files.length > 1) {
        notices.push(t('photosSkipped', { count: problems.length }));
      }
      notices.push(...[...new Set(problems)].map(imageProblemMessage));
    }
    setPhotoError(notices.join(' '));
    if (added > 0) {
      setPhotoStatus(t('photosAdded', { count: added, total: photos.length + added }));
    }
  };

  const movePhoto = (index: number, action: 'earlier' | 'later') => {
    const target = action === 'earlier' ? index - 1 : index + 1;
    if (target < 0 || target >= photos.length) {
      return;
    }
    const next = [...photos];
    [next[index], next[target]] = [next[target], next[index]];
    focusAfterChange.current = { id: photos[index].id, action };
    setPhotos(next);
    setPhotoStatus(t('photoMoved', { position: target + 1, total: photos.length }));
  };

  const removePhoto = (photo: ComposerPhoto) => {
    URL.revokeObjectURL(photo.url);
    focusAfterChange.current = 'picker';
    setPhotos((current) => current.filter((item) => item.id !== photo.id));
    setPhotoError('');
    setPhotoStatus(t('photoRemoved', { count: photos.length - 1 }));
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
    const photoProblem = getPostPhotoProblem(error);
    if (photoProblem?.kind === 'tooMany') {
      setFormError(t('errors.tooManyPhotos', { max: postPhotoLimit }));
    } else if (photoProblem?.kind === 'photoTooLarge') {
      setFormError(t('errors.photoTooLarge', { index: photoProblem.index + 1 }));
    } else if (photoProblem?.kind === 'unsupportedImage') {
      setFormError(t('errors.photoUnsupported', { index: photoProblem.index + 1 }));
    } else if (photoProblem?.kind === 'totalTooLarge') {
      setFormError(t('errors.totalTooLarge'));
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
        media: photos.map((photo) => photo.file),
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
    if (photos.length === 0 && !caption.trim()) {
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
          {t('photoLabel')}{' '}
          <span className={classes.photoCount}>{t('photoCountOf', { count: photos.length, max: postPhotoLimit })}</span>
        </div>
        {photos.length === 0 ? (
          <div className={classes.preview}>
            <span>
              ＋<br />
              {progress ? t('photosProcessing', progress) : t('photoHint', { max: postPhotoLimit })}
            </span>
          </div>
        ) : (
          <ol className={classes.thumbs} ref={thumbs} aria-labelledby='post-photo-label'>
            {photos.map((photo, index) => (
              <li key={photo.id} className={classes.thumb}>
                {/* biome-ignore lint/performance/noImgElement: a local object URL preview, nothing for next/image to optimise */}
                <img src={photo.url} alt={t('previewAlt', { index: index + 1, total: photos.length })} />
                <span className={classes.thumbIndex} aria-hidden>
                  {index + 1}
                </span>
                <div className={classes.thumbControls}>
                  <UnstyledButton
                    className={classes.thumbButton}
                    data-photo={photo.id}
                    data-move='earlier'
                    aria-label={t('movePhotoEarlier', { index: index + 1 })}
                    disabled={index === 0}
                    onClick={() => movePhoto(index, 'earlier')}
                  >
                    ←
                  </UnstyledButton>
                  <UnstyledButton
                    className={classes.thumbButton}
                    data-photo={photo.id}
                    data-move='later'
                    aria-label={t('movePhotoLater', { index: index + 1 })}
                    disabled={index === photos.length - 1}
                    onClick={() => movePhoto(index, 'later')}
                  >
                    →
                  </UnstyledButton>
                  <UnstyledButton
                    className={classes.thumbButton}
                    data-danger
                    aria-label={t('removePhotoAt', { index: index + 1 })}
                    onClick={() => removePhoto(photo)}
                  >
                    ✕
                  </UnstyledButton>
                </div>
              </li>
            ))}
          </ol>
        )}
        <div className={classes.photoButtons}>
          <FileButton onChange={addPhotos} accept='image/*' multiple resetRef={resetPicker} disabled={full}>
            {(props) => (
              <Button
                {...props}
                ref={pickerButton}
                variant='light'
                size='sm'
                loading={processing}
                disabled={full}
                aria-describedby='post-photo-label'
              >
                {photos.length > 0 ? t('addPhotos') : t('choosePhoto')}
              </Button>
            )}
          </FileButton>
          <span className={classes.photoNote}>
            {progress && photos.length > 0
              ? t('photosProcessing', progress)
              : full
                ? t('photoLimitReached', { max: postPhotoLimit })
                : ''}
          </span>
        </div>
        <p className={shared.formError} role='alert'>
          {photoError}
        </p>
        <VisuallyHidden aria-live='polite'>{photoStatus}</VisuallyHidden>

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
