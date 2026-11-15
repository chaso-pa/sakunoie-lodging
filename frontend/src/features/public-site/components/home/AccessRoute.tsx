import { IconHome, IconMapPin, IconTrain } from '@tabler/icons-react';
import { FeatureIcon } from '@/features/public-site/components/FeatureCard';
import classes from '@/features/public-site/components/home/AccessRoute.module.scss';
import { TravelTime } from '@/features/public-site/components/home/TravelTime';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Access = PageFrontMatter<'home'>['access'];
type Props = {
  access: Access;
  area: string;
  /** Confirmed facts as chips (free parking, from the facility amenities). */
  facts: { icon: string; label: string }[];
  routeLabel: string;
};

/**
 * Access as a vertical timeline (Tokyo → Sakudaira → the house, with the travel times on the rail),
 * plus the other nearby station, parking and the "address after booking" note beside it.
 */
export function AccessRoute({ access, area, facts, routeLabel }: Props) {
  const { steps, legs } = access.route;
  return (
    <div className={classes.wrap}>
      <ol className={classes.route} aria-label={routeLabel}>
        {steps.map((step, i) => {
          const last = i === steps.length - 1;
          return (
            <li key={step.name} className={classes.step} data-last={last || undefined}>
              <span className={classes.node}>
                {last ? (
                  <IconHome size={20} aria-hidden stroke={1.8} />
                ) : (
                  <IconTrain size={18} aria-hidden stroke={1.8} />
                )}
              </span>
              <div className={classes.stop}>
                <span className={classes.name}>{step.name}</span>
                {step.note ? <span className={classes.note}>{step.note}</span> : null}
              </div>
              {legs[i] ? (
                <div className={classes.leg}>
                  {legs[i].travel.map((t) => (
                    <TravelTime key={t.text} travel={t} />
                  ))}
                </div>
              ) : null}
            </li>
          );
        })}
      </ol>

      <div className={classes.side}>
        {access.alternatives.map((alt) => (
          <div key={alt.from} className={classes.alt}>
            <p className={classes.altFrom}>
              <IconTrain size={18} aria-hidden stroke={1.8} />
              {alt.from}
            </p>
            <div className={classes.altTimes}>
              {alt.travel.map((t) => (
                <TravelTime key={t.text} travel={t} />
              ))}
            </div>
          </div>
        ))}
        <ul className={classes.facts}>
          {facts.map((f) => (
            <li key={f.label} className={classes.fact}>
              <FeatureIcon name={f.icon} size={18} />
              {f.label}
            </li>
          ))}
        </ul>
        <p className={classes.address}>
          <IconMapPin size={18} aria-hidden stroke={1.8} />
          <span>
            <strong className={classes.area}>{area}</strong>
            {access.body}
          </span>
        </p>
      </div>
    </div>
  );
}
