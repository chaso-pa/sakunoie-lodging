'use client';

import 'leaflet/dist/leaflet.css';

import L from 'leaflet';
import { useEffect, useRef } from 'react';
import { env } from '@/config/env';
import classes from '@/features/post/components/Place.module.scss';
import type { LatLng } from '@/features/post/types/place';

export type MapMarker = LatLng & {
  id: string;
  /** Read by screen readers and shown as the marker's tooltip. */
  label: string;
  emoji: string;
};

type PlaceMapProps = {
  label: string;
  markers: MapMarker[];
  selectedId: string | null;
  /** A position the member tapped that is not a marker. */
  pin: LatLng | null;
  /** The member's own position, when they asked for places near them. Drawn only; never sent anywhere from here. */
  me: LatLng | null;
  /** The map moves here whenever this object changes. */
  focus: (LatLng & { zoom?: number }) | null;
  onMarkerSelect: (id: string) => void;
  onPositionSelect: (position: LatLng) => void;
};

// Saku City Hall. The picker opens on the area the lodging is in, wide enough to take in the towns around it.
export const sakuCenter: LatLng = { latitude: 36.2489, longitude: 138.4769 };
const defaultZoom = 10;
const focusZoom = 16;

const escapeHtml = (text: string) => text.replace(/[&<>"']/g, (char) => `&#${char.charCodeAt(0)};`);

const markerIcon = (emoji: string, selected: boolean) =>
  L.divIcon({
    className: '',
    html: `<span class="${classes.marker}"${selected ? ' data-selected' : ''}><span>${escapeHtml(emoji)}</span></span>`,
    iconSize: [40, 40],
    iconAnchor: [20, 38]
  });

const dotIcon = (className: string, size: number) =>
  L.divIcon({ className: '', html: `<span class="${className}"></span>`, iconSize: [size, size] });

/**
 * The Leaflet map of the place picker. Leaflet needs `window`, so this file is only ever loaded through
 * `next/dynamic` with `ssr: false`. Tapping a marker selects it; tapping or long-pressing anywhere else
 * reports that position.
 */
export function PlaceMap({
  label,
  markers,
  selectedId,
  pin,
  me,
  focus,
  onMarkerSelect,
  onPositionSelect
}: PlaceMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<L.Map | null>(null);
  const markerLayerRef = useRef<L.LayerGroup | null>(null);
  const overlayLayerRef = useRef<L.LayerGroup | null>(null);
  const handlersRef = useRef({ onMarkerSelect, onPositionSelect });
  handlersRef.current = { onMarkerSelect, onPositionSelect };

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }
    const map = L.map(container, {
      center: [sakuCenter.latitude, sakuCenter.longitude],
      zoom: defaultZoom,
      zoomControl: false
    });
    // Zoom buttons at the bottom right, within reach of a thumb.
    L.control.zoom({ position: 'bottomright' }).addTo(map);
    map.attributionControl.setPrefix('<a href="https://leafletjs.com" target="_blank" rel="noreferrer">Leaflet</a>');
    L.tileLayer(env.mapTileUrl, {
      attribution: env.mapTileAttribution,
      maxZoom: env.mapTileMaxZoom,
      // The openstreetmap.org tile policy requires a Referer; keep sending one whatever the page's policy is.
      referrerPolicy: 'strict-origin-when-cross-origin'
    }).addTo(map);

    const report = (event: L.LeafletMouseEvent) =>
      handlersRef.current.onPositionSelect({ latitude: event.latlng.lat, longitude: event.latlng.lng });
    map.on('click', report);
    // A long press arrives as `contextmenu` on touch screens.
    map.on('contextmenu', report);

    markerLayerRef.current = L.layerGroup().addTo(map);
    overlayLayerRef.current = L.layerGroup().addTo(map);
    mapRef.current = map;

    // The modal animates open, so the container has no final size when the map is created.
    const observer = new ResizeObserver(() => map.invalidateSize());
    observer.observe(container);

    return () => {
      observer.disconnect();
      map.remove();
      mapRef.current = null;
    };
  }, []);

  useEffect(() => {
    const layer = markerLayerRef.current;
    if (!layer) {
      return;
    }
    layer.clearLayers();
    for (const item of markers) {
      const selected = item.id === selectedId;
      L.marker([item.latitude, item.longitude], {
        icon: markerIcon(item.emoji, selected),
        title: item.label,
        alt: item.label,
        zIndexOffset: selected ? 1000 : 0,
        // Without this a tap on a marker would also reach the map as a tap on an empty position.
        bubblingMouseEvents: false
      })
        .on('click', () => handlersRef.current.onMarkerSelect(item.id))
        .addTo(layer);
    }
  }, [markers, selectedId]);

  useEffect(() => {
    const layer = overlayLayerRef.current;
    if (!layer) {
      return;
    }
    layer.clearLayers();
    if (me) {
      L.marker([me.latitude, me.longitude], {
        icon: dotIcon(classes.meDot, 18),
        interactive: false,
        keyboard: false
      }).addTo(layer);
    }
    if (pin) {
      L.marker([pin.latitude, pin.longitude], {
        icon: markerIcon('📍', true),
        interactive: false,
        keyboard: false,
        zIndexOffset: 2000
      }).addTo(layer);
    }
  }, [pin, me]);

  useEffect(() => {
    if (focus && mapRef.current) {
      mapRef.current.setView([focus.latitude, focus.longitude], focus.zoom ?? focusZoom);
    }
  }, [focus]);

  // biome-ignore lint/a11y/useSemanticElements: Leaflet draws into this element; it is a map, not a form fieldset
  return <div ref={containerRef} className={classes.map} role='group' aria-label={label} />;
}
