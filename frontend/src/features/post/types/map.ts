import type { LatLng } from '@/features/post/types/place';

export type MapMarker = LatLng & {
  id: string;
  /** Read by screen readers and shown as the marker's tooltip. */
  label: string;
  emoji: string;
};

/** What both maps of the picker (Leaflet's and Google's) take, so the picker does not care which it draws. */
export type PlaceMapProps = {
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
export const defaultZoom = 10;
export const focusZoom = 16;
