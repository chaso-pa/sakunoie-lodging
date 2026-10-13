'use client';

import { Button, Loader, Modal } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import dynamic from 'next/dynamic';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';
import shared from '@/components/Community.module.scss';
import { useReversePlace } from '@/features/post/api/reverse-place';
import classes from '@/features/post/components/Place.module.scss';
import type { MapMarker } from '@/features/post/components/PlaceMap';
import { PlaceAttribution } from '@/features/post/components/PlaceRows';
import type { LocateStatus } from '@/features/post/hooks/use-current-position';
import { googleSource, type LatLng, type Place, placeKey } from '@/features/post/types/place';
import { getPlaceProblem } from '@/features/post/utils/place-problem';

const PlaceMap = dynamic(() => import('@/features/post/components/PlaceMap').then((module) => module.PlaceMap), {
  ssr: false,
  loading: () => (
    <div className={classes.mapLoading}>
      <Loader />
    </div>
  )
});

/** A community spot that has a position, ready to be drawn. */
export type MapSpot = LatLng & { id: string; name: string; detail: string; emoji: string };

export type MapRequest =
  /** Choose a place: markers, or a tapped position looked up on the server. */
  | { mode: 'pick'; focus: Place | MapSpot | null }
  /** Only put a pin down, for the hand-entered place. */
  | { mode: 'position'; position: LatLng | null };

type PlaceMapModalProps = {
  /** null closes the modal. */
  request: MapRequest | null;
  spots: MapSpot[];
  places: Place[];
  me: LatLng | null;
  locateStatus: LocateStatus;
  nearbyPending: boolean;
  /** Text for the last "near me" attempt, empty when there is nothing to say. */
  nearbyMessage: string;
  onNearMe: () => void;
  onClose: () => void;
  onSelectSpot: (spot: MapSpot) => void;
  onSelectPlace: (place: Place) => void;
  onPosition: (position: LatLng) => void;
  onRegisterAt: (position: LatLng) => void;
};

const placeMarkerId = (place: Place) => `place:${placeKey(place)}`;
const spotMarkerId = (id: string) => `spot:${id}`;
const isPlace = (value: Place | MapSpot): value is Place => 'source' in value;

type DrawablePlace = Place & LatLng;
/** Google's terms keep its results off any map but its own, so those stay in the list. */
const isDrawable = (place: Place): place is DrawablePlace =>
  place.source !== googleSource && place.latitude !== null && place.longitude !== null;

const wrapButton = { root: classes.wrapButton, label: classes.wrapLabel };

type Pending = { kind: 'marker'; id: string } | { kind: 'position'; position: LatLng };

function MapSheet({
  request,
  spots,
  places,
  me,
  locateStatus,
  nearbyPending,
  nearbyMessage,
  onNearMe,
  onClose,
  onSelectSpot,
  onSelectPlace,
  onPosition,
  onRegisterAt
}: Omit<PlaceMapModalProps, 'request'> & { request: MapRequest }) {
  const t = useTranslations('community.compose.place');
  const tCommon = useTranslations('community.common');
  const tErrors = useTranslations('errors');
  const locale = useLocale();

  const focused = request.mode === 'pick' ? request.focus : null;
  const [pending, setPending] = useState<Pending | null>(() => {
    if (request.mode === 'position') {
      return request.position ? { kind: 'position', position: request.position } : null;
    }
    if (!focused) {
      return null;
    }
    return { kind: 'marker', id: isPlace(focused) ? placeMarkerId(focused) : spotMarkerId(focused.id) };
  });
  const [focus] = useState<LatLng | null>(() => {
    if (request.mode === 'position') {
      return request.position;
    }
    return focused && focused.latitude !== null && focused.longitude !== null
      ? { latitude: focused.latitude, longitude: focused.longitude }
      : null;
  });

  const drawable = useMemo(() => {
    const all = focused && isPlace(focused) ? [focused, ...places] : places;
    const seen = new Set(spots.map((spot) => spot.id));
    return all.filter(isDrawable).filter((place) => {
      // A result that is already a spot is drawn once, as that spot.
      const id = place.spot_id ?? placeMarkerId(place);
      if (seen.has(id)) {
        return false;
      }
      seen.add(id);
      return true;
    });
  }, [places, focused, spots]);
  const hiddenCount = places.filter((place) => place.source === googleSource).length;

  const markers = useMemo<MapMarker[]>(
    () =>
      request.mode === 'position'
        ? []
        : [
            ...spots.map((spot) => ({
              id: spotMarkerId(spot.id),
              label: spot.name,
              emoji: spot.emoji,
              latitude: spot.latitude,
              longitude: spot.longitude
            })),
            ...drawable.map((place) => ({
              id: placeMarkerId(place),
              label: place.name,
              emoji: place.emoji,
              latitude: place.latitude,
              longitude: place.longitude
            }))
          ],
    [request.mode, spots, drawable]
  );

  const tapped = pending?.kind === 'position' ? pending.position : null;
  const reverse = useReversePlace(request.mode === 'pick' ? tapped : null, locale);

  const pendingSpot =
    pending?.kind === 'marker' ? spots.find((spot) => spotMarkerId(spot.id) === pending.id) : undefined;
  const pendingPlace =
    pending?.kind === 'marker' ? drawable.find((place) => placeMarkerId(place) === pending.id) : undefined;
  const found = reverse.data?.result ?? null;
  const shown = pendingPlace ?? (tapped && request.mode === 'pick' ? found : null);

  const confirm = () => {
    if (pendingSpot) {
      onSelectSpot(pendingSpot);
    } else if (pendingPlace) {
      onSelectPlace(pendingPlace);
    } else if (tapped && request.mode === 'position') {
      onPosition(tapped);
    } else if (tapped && found) {
      onSelectPlace(found);
    }
  };
  const canConfirm = Boolean(pendingSpot || pendingPlace || (tapped && (request.mode === 'position' || found)));

  const reverseProblem = reverse.isError ? getPlaceProblem(reverse.error) : null;

  return (
    <div className={classes.sheet}>
      <div className={classes.mapWrap}>
        <PlaceMap
          label={t('map.label')}
          markers={markers}
          selectedId={pending?.kind === 'marker' ? pending.id : null}
          pin={tapped}
          me={me}
          focus={me ?? focus}
          onMarkerSelect={(id) => setPending({ kind: 'marker', id })}
          onPositionSelect={(position) => setPending({ kind: 'position', position })}
        />
        {request.mode === 'pick' && (
          <div className={classes.mapTools}>
            <Button variant='white' size='sm' loading={locateStatus === 'locating' || nearbyPending} onClick={onNearMe}>
              📍 {t('nearMe')}
            </Button>
          </div>
        )}
      </div>

      <div className={classes.panel}>
        <output className={classes.panelPlace} aria-live='polite'>
          {pendingSpot ? (
            <span translate='no' className={classes.panelPlace}>
              <span className={classes.name}>{pendingSpot.name}</span>
              <span className={classes.address}>{pendingSpot.detail}</span>
            </span>
          ) : tapped && request.mode === 'position' ? (
            <span className={classes.address}>{t('map.positionChosen')}</span>
          ) : tapped && reverse.isPending ? (
            <span className={`${classes.address} ${classes.searching}`}>
              <Loader size='xs' /> {t('map.lookingUp')}
            </span>
          ) : shown ? (
            <span translate='no' className={classes.panelPlace}>
              <span className={classes.name}>{shown.name}</span>
              {shown.original_name && <span className={classes.originalName}>{shown.original_name}</span>}
              {shown.address && <span className={classes.address}>{shown.address}</span>}
            </span>
          ) : tapped ? (
            <span className={classes.address}>
              {reverseProblem === null
                ? t('map.nothingHere')
                : reverseProblem === 'network'
                  ? tErrors('network')
                  : reverseProblem === 'rateLimited'
                    ? t('errorRateLimited')
                    : reverseProblem === 'notConfigured'
                      ? t('map.lookupUnavailable')
                      : t('map.lookupFailed')}
            </span>
          ) : (
            <span className={shared.muted}>{request.mode === 'position' ? t('map.positionHint') : t('map.hint')}</span>
          )}
        </output>
        {tapped && found && reverse.data && (
          <PlaceAttribution provider={reverse.data.provider} attribution={reverse.data.attribution} />
        )}
        {request.mode === 'pick' && !tapped && nearbyMessage && (
          <output className={classes.status} aria-live='polite'>
            {nearbyMessage}
          </output>
        )}
        {request.mode === 'pick' && hiddenCount > 0 && <p className={classes.status}>{t('map.googleNotOnMap')}</p>}
        {request.mode === 'pick' && tapped && !reverse.isPending && (
          <Button
            variant='light'
            fullWidth
            mt={10}
            size='md'
            classNames={wrapButton}
            onClick={() => onRegisterAt(tapped)}
          >
            ✏️ {t('map.registerHere')}
          </Button>
        )}
        <div className={classes.panelActions}>
          <Button variant='default' size='lg' classNames={wrapButton} onClick={onClose}>
            {tCommon('cancel')}
          </Button>
          <Button size='lg' classNames={wrapButton} disabled={!canConfirm} onClick={confirm}>
            {request.mode === 'position' ? t('map.confirmPosition') : t('map.confirm')}
          </Button>
        </div>
      </div>
    </div>
  );
}

/**
 * Full-screen map for choosing a place with one thumb: the map on top, the place under consideration and
 * the confirm and cancel buttons at the bottom. Everything in it is forgotten when it closes.
 */
export function PlaceMapModal({ request, onClose, ...rest }: PlaceMapModalProps) {
  const t = useTranslations('community.compose.place');
  const tCommon = useTranslations('community.common');
  const isPhone = useMediaQuery('(max-width: 520px)');

  return (
    <Modal
      opened={request !== null}
      onClose={onClose}
      title={request?.mode === 'position' ? t('map.positionTitle') : t('map.title')}
      fullScreen={isPhone}
      size={480}
      centered
      padding={0}
      closeButtonProps={{ 'aria-label': tCommon('cancel'), size: 'lg' }}
      classNames={{ content: classes.mapModal, header: classes.mapModalHeader, body: classes.mapModalBody }}
    >
      {request && <MapSheet request={request} onClose={onClose} {...rest} />}
    </Modal>
  );
}
