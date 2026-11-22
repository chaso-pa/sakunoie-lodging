import { FeatureIcon } from '@/features/public-site/components/FeatureCard';
import classes from '@/features/public-site/components/facility/KeyFacts.module.scss';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Props = { label: string; facts: PageFrontMatter<'facilities'>['keyFacts'] };

/**
 * The facts a guest decides on (guests, check-in/out), as a description list. A fact with
 * `lines` shows several labelled values, e.g. "IN 15:00〜" over "OUT 〜10:00".
 */
export function KeyFacts({ label, facts }: Props) {
  return (
    <section aria-labelledby='key-facts-title'>
      <h2 id='key-facts-title' className='visually-hidden'>
        {label}
      </h2>
      <dl className={classes.grid} data-count={facts.length}>
        {facts.map((f) => (
          <div key={f.label} className={classes.item}>
            <dt className={classes.label}>
              <span className={classes.icon}>
                <FeatureIcon name={f.icon} size={20} />
              </span>
              {f.label}
            </dt>
            {'lines' in f ? (
              f.lines.map((line) => (
                <dd key={line.tag} className={classes.line}>
                  <span className={classes.tag}>{line.tag}</span>
                  <span className={classes.value}>{line.value}</span>
                </dd>
              ))
            ) : (
              <dd className={classes.value}>{f.value}</dd>
            )}
          </div>
        ))}
      </dl>
    </section>
  );
}
