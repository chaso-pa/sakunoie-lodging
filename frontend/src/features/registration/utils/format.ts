/** Formats a UTC instant in the facility time zone, e.g. for the edit-link expiry. */
export const formatInstant = (iso: string, locale: string, timeZone: string) =>
  // dateStyle/timeStyle cannot be combined with timeZoneName, so the fields are listed explicitly.
  new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZone,
    timeZoneName: 'short'
  }).format(new Date(iso));

/** Formats a civil date (`YYYY-MM-DD`) without shifting it through any time zone. */
export const formatCivilDate = (date: string, locale: string) => {
  const parsed = new Date(`${date}T00:00:00Z`);
  if (Number.isNaN(parsed.getTime())) return date;
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeZone: 'UTC' }).format(parsed);
};
