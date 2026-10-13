'use client';

import { Button, Loader, TextInput, UnstyledButton } from '@mantine/core';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';
import shared from '@/components/Community.module.scss';
import { SpotCredit } from '@/components/SpotCredit';
import { useNearbyPlaces } from '@/features/post/api/nearby-places';
import { placeQueryMaxLength, usePlaceSearch } from '@/features/post/api/search-places';
import classes from '@/features/post/components/Place.module.scss';
import { type MapRequest, type MapSpot, PlaceMapModal } from '@/features/post/components/PlaceMapModal';
import { PlaceRegisterModal, type RegisterDraft } from '@/features/post/components/PlaceRegisterModal';
import { PlaceAttribution, type PlaceRow, PlaceRows } from '@/features/post/components/PlaceRows';
import { useCurrentPosition } from '@/features/post/hooks/use-current-position';
import {
  googleSource,
  type LatLng,
  type Place,
  type PlaceSelection,
  placeKey,
  toSelection
} from '@/features/post/types/place';
import { getPlaceProblem, getRetryAfterSeconds, type PlaceProblem } from '@/features/post/utils/place-problem';
import { useCommunityLabels } from '@/hooks/use-community-labels';
import { useSpots } from '@/hooks/use-spots';
import type { SpotListItem } from '@/types/community';

const minQueryLength = 2;
const suggestionCount = 8;

const wrapButton = { root: classes.wrapButton, label: classes.wrapLabel };

type PlacePickerProps = {
  value: PlaceSelection | null;
  onChange: (value: PlaceSelection | null) => void;
};

/**
 * The composer's optional place field: search by name or address, pick on the map, look around the
 * current position, or type a place in by hand. Choosing a place here does not create anything on the
 * server; the composer registers a new place only when the post is shared.
 */
export function PlacePicker({ value, onChange }: PlacePickerProps) {
  const t = useTranslations('community.compose.place');
  const tCommon = useTranslations('community.common');
  const tErrors = useTranslations('errors');
  const tApp = useTranslations('common');
  const { areaLabel } = useCommunityLabels();
  const locale = useLocale();
  const spots = useSpots();
  const nearby = useNearbyPlaces();
  const { status: locateStatus, locate } = useCurrentPosition();

  const [changing, setChanging] = useState(false);
  const [input, setInput] = useState('');
  // What the member last asked to search for. The provider is asked once per Enter or tap on the search
  // button, never per keystroke: the default provider's usage policy forbids search-as-you-type.
  const [submitted, setSubmitted] = useState('');
  const [blockedUntil, setBlockedUntil] = useState(0);
  const [blocked, setBlocked] = useState(false);
  // In memory for as long as the composer is open, to centre the map. Never stored and never posted.
  const [me, setMe] = useState<LatLng | null>(null);
  const [mapRequest, setMapRequest] = useState<MapRequest | null>(null);
  const [draft, setDraft] = useState<RegisterDraft | null>(null);
  const [searchFocused, setSearchFocused] = useState(false);

  const rootRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const changeRef = useRef<HTMLButtonElement>(null);
  const focusAfterRender = useRef<'input' | 'change' | null>(null);

  // Curated spots are matched in the browser as the member types; that costs no request.
  const keyword = input.trim();
  const search = usePlaceSearch(blocked ? '' : submitted, locale);
  const edited = keyword !== submitted;
  const submitSearch = () => setSubmitted(keyword.length >= minQueryLength ? keyword : '');

  // After a 429 nothing is sent until the wait is over, whatever is typed in the meantime.
  useEffect(() => {
    if (search.isError && getPlaceProblem(search.error) === 'rateLimited') {
      setBlockedUntil(Date.now() + getRetryAfterSeconds(search.error) * 1000);
      setBlocked(true);
    }
  }, [search.isError, search.error]);
  useEffect(() => {
    if (!blocked) {
      return;
    }
    const timer = setTimeout(() => setBlocked(false), Math.max(0, blockedUntil - Date.now()));
    return () => clearTimeout(timer);
  }, [blocked, blockedUntil]);

  // On a phone the keyboard covers the lower half of the screen. Keep the field at the top while it has
  // focus, also after the keyboard has finished opening, so the results under it stay in view.
  useEffect(() => {
    if (!searchFocused || !window.matchMedia('(pointer: coarse)').matches) {
      return;
    }
    const toTop = () => rootRef.current?.scrollIntoView({ block: 'start', behavior: 'instant' });
    toTop();
    window.visualViewport?.addEventListener('resize', toTop);
    return () => window.visualViewport?.removeEventListener('resize', toTop);
  }, [searchFocused]);

  useEffect(() => {
    if (focusAfterRender.current === 'input') {
      inputRef.current?.focus();
    } else if (focusAfterRender.current === 'change') {
      changeRef.current?.focus();
    }
    focusAfterRender.current = null;
  });

  const spotDetail = (spot: SpotListItem) =>
    spot.address || [areaLabel(spot.area), spot.category].filter(Boolean).join(' · ');
  const spotSelection = (spot: SpotListItem): PlaceSelection => ({
    kind: 'spot',
    id: spot.id,
    name: spot.name,
    originalName: spot.original_name,
    address: spotDetail(spot),
    emoji: spot.emoji,
    area: spot.area,
    source: spot.source
  });

  const select = (selection: PlaceSelection) => {
    onChange(selection);
    setChanging(false);
    setInput('');
    setSubmitted('');
    setMapRequest(null);
    setDraft(null);
    focusAfterRender.current = 'change';
  };

  // A spot whose details could not be fetched has no name to offer.
  const allSpots = (spots.data ?? []).filter((spot) => spot.details_available && spot.name);
  const selectPlace = (place: Place) => {
    const known = place.spot_id ? allSpots.find((spot) => spot.id === place.spot_id) : undefined;
    const selection = known ? spotSelection(known) : toSelection(place);
    if (selection) {
      select(selection);
    }
  };
  const mapSpots: MapSpot[] = allSpots.flatMap((spot) =>
    // Google's terms keep its places off any map but its own.
    spot.source !== googleSource && spot.latitude !== null && spot.longitude !== null
      ? [
          {
            id: spot.id,
            name: spot.name,
            detail: spotDetail(spot),
            emoji: spot.emoji,
            latitude: spot.latitude,
            longitude: spot.longitude
          }
        ]
      : []
  );

  const spotRow = (spot: SpotListItem): PlaceRow => {
    const onMap = mapSpots.find((item) => item.id === spot.id);
    return {
      key: `spot:${spot.id}`,
      name: spot.name,
      detail: spotDetail(spot),
      category: '',
      emoji: spot.emoji,
      onSelect: () => select(spotSelection(spot)),
      onShowOnMap: onMap && (() => setMapRequest({ mode: 'pick', focus: onMap }))
    };
  };
  const placeRow = (place: Place): PlaceRow => {
    const onOwnMap = place.source !== googleSource && place.latitude !== null && place.longitude !== null;
    return {
      key: `place:${placeKey(place)}`,
      name: place.name,
      originalName: place.original_name,
      detail: place.address,
      category: t(`categories.${place.category_key}`),
      distanceM: place.distance_m,
      emoji: place.emoji,
      photoUrl: place.photo_url,
      onSelect: () => selectPlace(place),
      onShowOnMap: onOwnMap ? () => setMapRequest({ mode: 'pick', focus: place }) : undefined,
      mapUrl: place.map_url
    };
  };

  const lower = keyword.toLowerCase();
  const matchingSpots = keyword
    ? allSpots.filter((spot) =>
        [spot.name, spot.original_name ?? '', spot.address ?? '', spot.area, spot.category, spot.description]
          .join(' ')
          .toLowerCase()
          .includes(lower)
      )
    : [];

  const problemText = (problem: PlaceProblem) =>
    problem === 'network'
      ? tErrors('network')
      : problem === 'rateLimited'
        ? t('errorRateLimited')
        : problem === 'notConfigured'
          ? t('errorNotConfigured')
          : problem === 'provider'
            ? t('errorProvider')
            : tErrors('generic');

  const nearMe = async () => {
    nearby.reset();
    const position = await locate();
    setMe(position);
    if (position) {
      nearby.mutate({ position, language: locale });
    }
  };

  const locateText =
    locateStatus === 'locating'
      ? t('locating')
      : locateStatus === 'denied'
        ? t('locateDenied')
        : locateStatus === 'unavailable'
          ? t('locateUnavailable')
          : locateStatus === 'timeout'
            ? t('locateTimeout')
            : locateStatus === 'unsupported'
              ? t('locateUnsupported')
              : '';
  const nearbyMessage = locateText
    ? locateText
    : nearby.isPending
      ? t('nearbySearching')
      : nearby.isError
        ? problemText(getPlaceProblem(nearby.error))
        : nearby.data
          ? nearby.data.results.length === 0
            ? t('noNearby')
            : t('nearbyFound', { count: nearby.data.results.length })
          : '';

  const searchProblem = blocked ? 'rateLimited' : search.isError ? getPlaceProblem(search.error) : null;
  // Spots matched in the browser are listed first; the same spot is not repeated among the results.
  const listedSpotIds = new Set(matchingSpots.map((spot) => spot.id));
  const selectable = (place: Place) => Boolean(place.spot_id || place.external_id);
  const results = (search.data?.results ?? []).filter(
    (place) => selectable(place) && !(place.spot_id && listedSpotIds.has(place.spot_id))
  );
  const nearbyPlaces = (nearby.data?.results ?? []).filter(selectable);
  const showSearch = !value || changing;

  const openRegister = (position: LatLng | null = null) => {
    setMapRequest(null);
    setDraft({ name: input.trim().slice(0, 100), address: '', position });
  };

  const modals = (
    <>
      <PlaceMapModal
        request={mapRequest}
        spots={mapSpots}
        places={[...results, ...nearbyPlaces]}
        me={me}
        locateStatus={locateStatus}
        nearbyPending={nearby.isPending}
        nearbyMessage={nearbyMessage}
        onNearMe={nearMe}
        onClose={() => setMapRequest(null)}
        onSelectSpot={(spot) => {
          const known = allSpots.find((item) => item.id === spot.id);
          if (known) {
            select(spotSelection(known));
          }
        }}
        onSelectPlace={selectPlace}
        onPosition={(position) => {
          setMapRequest(null);
          setDraft((current) => current && { ...current, position });
        }}
        onRegisterAt={openRegister}
      />
      <PlaceRegisterModal
        // Hidden, not closed, while the map is open over it for the position.
        draft={mapRequest?.mode === 'position' ? null : draft}
        onChange={setDraft}
        onClose={() => setDraft(null)}
        onPickPosition={() => draft && setMapRequest({ mode: 'position', position: draft.position })}
        onSubmit={select}
      />
    </>
  );

  if (!showSearch && value) {
    const name = value.kind === 'candidate' ? value.place.name : value.name;
    const address = value.kind === 'candidate' ? value.place.address : value.address;
    const originalName =
      value.kind === 'candidate' ? value.place.original_name : value.kind === 'spot' ? value.originalName : null;
    return (
      // biome-ignore lint/a11y/useSemanticElements: one labelled field of the composer, not a group of form controls
      <div className={classes.picker} role='group' aria-labelledby='post-place-label'>
        <div className={shared.fieldLabel} id='post-place-label'>
          {t('label')}
        </div>
        <div className={classes.selected}>
          <span className={shared.spotEmojiSmall} aria-hidden>
            {value.kind === 'spot' ? value.emoji : value.kind === 'candidate' ? value.place.emoji : '📍'}
          </span>
          <div className={classes.rowText}>
            <span className={classes.name} translate='no'>
              {name}
            </span>
            {originalName && (
              <span className={classes.originalName} translate='no'>
                {originalName}
              </span>
            )}
            {address && (
              <span className={classes.address} translate='no'>
                {address}
              </span>
            )}
            {value.kind !== 'manual' && (
              <SpotCredit source={value.kind === 'candidate' ? value.place.source : value.source} />
            )}
            {value.kind !== 'spot' && <span className={classes.selectedNote}>{t('newPlaceNote')}</span>}
          </div>
        </div>
        <div className={classes.selectedActions}>
          <Button
            ref={changeRef}
            variant='light'
            size='sm'
            onClick={() => {
              setChanging(true);
              focusAfterRender.current = 'input';
            }}
          >
            {t('change')}
          </Button>
          <Button
            variant='default'
            size='sm'
            onClick={() => {
              onChange(null);
              focusAfterRender.current = 'input';
            }}
          >
            {t('clear')}
          </Button>
        </div>
        {modals}
      </div>
    );
  }

  return (
    <div className={classes.picker} ref={rootRef}>
      <TextInput
        ref={inputRef}
        type='search'
        label={t('label')}
        description={t('helper')}
        inputWrapperOrder={['label', 'input', 'description']}
        placeholder={t('searchPlaceholder')}
        size='md'
        radius='sm'
        classNames={{ label: shared.fieldLabel, description: `${shared.muted} ${classes.helper}` }}
        autoComplete='off'
        enterKeyHint='search'
        maxLength={placeQueryMaxLength}
        rightSectionWidth='auto'
        rightSectionPointerEvents='all'
        rightSection={
          <Button
            variant='filled'
            size='sm'
            radius='sm'
            mr={5}
            disabled={keyword === ''}
            loading={search.isFetching}
            onClick={submitSearch}
          >
            {t('searchButton')}
          </Button>
        }
        value={input}
        onChange={(event) => setInput(event.currentTarget.value)}
        onFocus={() => setSearchFocused(true)}
        onBlur={() => setSearchFocused(false)}
        onKeyDown={(event) => {
          // Enter searches; it must not submit the post. (An Enter that confirms an IME conversion is not one.)
          if (event.key === 'Enter' && !event.nativeEvent.isComposing) {
            event.preventDefault();
            submitSearch();
          }
        }}
      />
      <div className={classes.tools}>
        <Button
          variant='light'
          size='md'
          classNames={wrapButton}
          onClick={() => setMapRequest({ mode: 'pick', focus: null })}
        >
          🗺 {t('openMap')}
        </Button>
        <Button
          variant='light'
          size='md'
          classNames={wrapButton}
          loading={locateStatus === 'locating' || nearby.isPending}
          onClick={nearMe}
        >
          📍 {t('nearMe')}
        </Button>
      </div>
      <output className={`${shared.muted} ${classes.status}`} aria-live='polite'>
        {nearbyMessage}
      </output>

      <div aria-live='polite'>
        {keyword && matchingSpots.length > 0 && (
          <>
            <div className={classes.groupTitle}>{t('communityTitle')}</div>
            <PlaceRows rows={matchingSpots.map(spotRow)} label={t('communityTitle')} />
          </>
        )}
        {keyword && edited && (
          <p className={classes.stateBox}>{keyword.length < minQueryLength ? t('minChars') : t('searchHint')}</p>
        )}
        {submitted && !edited && (
          <>
            <div className={classes.groupTitle}>{t('resultsTitle')}</div>
            {searchProblem ? (
              <div className={classes.stateBox} data-error role='alert'>
                {problemText(searchProblem)}
                {searchProblem !== 'notConfigured' && (
                  <div>
                    <Button
                      variant='light'
                      size='xs'
                      mt={8}
                      disabled={blocked}
                      loading={search.isFetching}
                      onClick={() => search.refetch()}
                    >
                      {tApp('retry')}
                    </Button>
                  </div>
                )}
              </div>
            ) : search.isPending ? (
              <output className={`${classes.stateBox} ${classes.searching}`}>
                <Loader size='xs' /> {t('searching')}
              </output>
            ) : results.length === 0 ? (
              <p className={classes.stateBox}>{t('noResults')}</p>
            ) : (
              <>
                <PlaceRows rows={results.map(placeRow)} label={t('resultsTitle')} />
                {search.data && (
                  <PlaceAttribution provider={search.data.provider} attribution={search.data.attribution} />
                )}
              </>
            )}
          </>
        )}
        {nearbyPlaces.length > 0 && nearby.data && (
          <>
            <div className={classes.groupTitle}>{t('nearbyTitle')}</div>
            <PlaceRows rows={nearbyPlaces.map(placeRow)} label={t('nearbyTitle')} />
            <PlaceAttribution provider={nearby.data.provider} attribution={nearby.data.attribution} />
          </>
        )}
        {!keyword && allSpots.length > 0 && (
          <>
            <div className={classes.groupTitle}>{t('communityTitle')}</div>
            {/* biome-ignore lint/a11y/useSemanticElements: a row of shortcut buttons, not a form fieldset */}
            <div className={shared.chips} role='group' aria-label={t('communityTitle')}>
              {allSpots.slice(0, suggestionCount).map((spot) => (
                <UnstyledButton
                  key={spot.id}
                  className={`${shared.chip} ${classes.suggestion}`}
                  onClick={() => select(spotSelection(spot))}
                >
                  {spot.emoji} {spot.name}
                </UnstyledButton>
              ))}
            </div>
          </>
        )}
      </div>

      <UnstyledButton className={classes.linkButton} onClick={() => openRegister()}>
        ✏️ {t('registerLink')}
      </UnstyledButton>
      {value && (
        <UnstyledButton
          className={classes.linkButton}
          style={{ marginLeft: 14 }}
          onClick={() => {
            setChanging(false);
            focusAfterRender.current = 'change';
          }}
        >
          {tCommon('cancel')}
        </UnstyledButton>
      )}
      {modals}
    </div>
  );
}
