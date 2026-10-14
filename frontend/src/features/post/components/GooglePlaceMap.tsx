'use client';

import { Loader } from '@mantine/core';
import { useLocale } from 'next-intl';
import { useEffect, useRef, useState } from 'react';
import classes from '@/features/post/components/Place.module.scss';
import { defaultZoom, focusZoom, type PlaceMapProps, sakuCenter } from '@/features/post/types/map';
import type { LatLng } from '@/features/post/types/place';
import { type GoogleMapsLibraries, googleMapId, loadGoogleMaps } from '@/features/post/utils/load-google-maps';

const toGoogle = (position: LatLng): google.maps.LatLngLiteral => ({
  lat: position.latitude,
  lng: position.longitude
});

const markerContent = (emoji: string, selected: boolean) => {
  const wrap = document.createElement('span');
  wrap.className = classes.googleMarker;
  const pin = document.createElement('span');
  pin.className = classes.marker;
  if (selected) {
    pin.dataset.selected = '';
  }
  const face = document.createElement('span');
  face.textContent = emoji;
  pin.append(face);
  wrap.append(pin);
  return wrap;
};

/**
 * Takes markers off the map. After the API refuses the key it tears the map down itself, and detaching a
 * marker then throws; that must not take the page down with it while the picker swaps in the Leaflet map.
 */
const removeMarkers = (markers: google.maps.marker.AdvancedMarkerElement[]) => {
  for (const marker of markers) {
    try {
      marker.map = null;
    } catch {
      // The map is already gone.
    }
  }
};

const meContent = () => {
  const dot = document.createElement('span');
  dot.className = `${classes.meDot} ${classes.googleCentered}`;
  return dot;
};

/**
 * The Google map of the place picker, used when a browser key is configured. It takes the same props as
 * the Leaflet map and behaves the same: tapping a marker selects it; tapping or long-pressing anywhere
 * else reports that position. Loaded through `next/dynamic` only when the map opens; each opening creates
 * one map, which is one billed map load. If the API cannot load or refuses the key, `load-google-maps`
 * reports it and the picker swaps this map for the Leaflet one.
 */
export function GooglePlaceMap({
  label,
  markers,
  selectedId,
  pin,
  me,
  focus,
  onMarkerSelect,
  onPositionSelect
}: PlaceMapProps) {
  const locale = useLocale();
  const containerRef = useRef<HTMLDivElement>(null);
  const [map, setMap] = useState<{ map: google.maps.Map; lib: GoogleMapsLibraries } | null>(null);
  const handlersRef = useRef({ onMarkerSelect, onPositionSelect });
  handlersRef.current = { onMarkerSelect, onPositionSelect };

  useEffect(() => {
    let cancelled = false;
    let listeners: google.maps.MapsEventListener[] = [];
    loadGoogleMaps(locale)
      .then((lib) => {
        const container = containerRef.current;
        if (cancelled || !container) {
          return;
        }
        const created = new lib.Map(container, {
          center: toGoogle(sakuCenter),
          zoom: defaultZoom,
          mapId: googleMapId,
          // One finger pans the map: it fills the modal, and the page behind it does not scroll.
          gestureHandling: 'greedy',
          disableDefaultUI: true,
          // Zoom buttons at the bottom right, within reach of a thumb, as on the Leaflet map.
          zoomControl: true,
          zoomControlOptions: { position: lib.ControlPosition.RIGHT_BOTTOM },
          // A tap on a shop's icon is a tap on that position, not Google's info window.
          clickableIcons: false
        });
        const report = (event: google.maps.MapMouseEvent) => {
          if (event.latLng) {
            handlersRef.current.onPositionSelect({ latitude: event.latLng.lat(), longitude: event.latLng.lng() });
          }
        };
        // A long press arrives as `contextmenu` on touch screens that send one.
        listeners = [created.addListener('click', report), created.addListener('contextmenu', report)];
        setMap({ map: created, lib });
      })
      // The failure is reported by the loader; the picker then shows the Leaflet map instead of this one.
      .catch(() => undefined);
    return () => {
      cancelled = true;
      for (const listener of listeners) {
        listener.remove();
      }
    };
  }, [locale]);

  useEffect(() => {
    if (!map) {
      return;
    }
    const drawn = markers.map((item) => {
      const selected = item.id === selectedId;
      const marker = new map.lib.AdvancedMarkerElement({
        map: map.map,
        position: toGoogle(item),
        content: markerContent(item.emoji, selected),
        title: item.label,
        gmpClickable: true,
        zIndex: selected ? 1000 : 0
      });
      marker.addEventListener('gmp-click', () => handlersRef.current.onMarkerSelect(item.id));
      return marker;
    });
    return () => removeMarkers(drawn);
  }, [map, markers, selectedId]);

  useEffect(() => {
    if (!map) {
      return;
    }
    const drawn: google.maps.marker.AdvancedMarkerElement[] = [];
    if (me) {
      drawn.push(new map.lib.AdvancedMarkerElement({ map: map.map, position: toGoogle(me), content: meContent() }));
    }
    if (pin) {
      drawn.push(
        new map.lib.AdvancedMarkerElement({
          map: map.map,
          position: toGoogle(pin),
          content: markerContent('📍', true),
          zIndex: 2000
        })
      );
    }
    return () => removeMarkers(drawn);
  }, [map, pin, me]);

  useEffect(() => {
    if (focus && map) {
      map.map.moveCamera({ center: toGoogle(focus), zoom: focus.zoom ?? focusZoom });
    }
  }, [map, focus]);

  return (
    <>
      {/* biome-ignore lint/a11y/useSemanticElements: the Maps JavaScript API draws into this element; it is a map, not a form fieldset */}
      <div ref={containerRef} className={classes.map} role='group' aria-label={label} />
      {!map && (
        <div className={classes.mapLoading}>
          <Loader />
        </div>
      )}
    </>
  );
}
