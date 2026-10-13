import classes from '@/components/Community.module.scss';
import { paths } from '@/config/paths';

/**
 * Credit a spot's name, address and position need wherever they are shown: OpenStreetMap data is used
 * under the ODbL, and Google requires its name next to Places data shown without a Google map. Spots the
 * operator or a member wrote themselves need none. The wording is the providers' own, so it is not translated.
 */
export function SpotCredit({ source }: { source: string }) {
  if (source === 'osm') {
    return (
      <a className={classes.credit} href={paths.external.osmCopyright.getHref()} target='_blank' rel='noreferrer'>
        © OpenStreetMap contributors
      </a>
    );
  }
  if (source === 'google') {
    return (
      <span className={classes.credit} data-google translate='no'>
        Google Maps
      </span>
    );
  }
  return null;
}
