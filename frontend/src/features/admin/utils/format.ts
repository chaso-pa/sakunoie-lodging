/** `2026-12-01` → `2026/12/01` (civil dates are shown as-is, never shifted by a time zone). */
export const formatDate = (isoDate: string) => isoDate.replaceAll('-', '/');

/** UTC instant shown in the facility time zone, e.g. `2026/12/03 23:59`. */
export const formatInstant = (iso: string, timeZone: string) =>
  new Intl.DateTimeFormat('ja-JP', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).format(new Date(iso));

/** Filename from `Content-Disposition: attachment; filename="…"`; restricted to safe characters. */
export const filenameFromDisposition = (header: string | null | undefined, fallback: string) => {
  const match = header ? /filename="([^"]+)"/.exec(header) : null;
  const name = match?.[1] ?? '';
  return /^[A-Za-z0-9._-]+$/.test(name) ? name : fallback;
};
