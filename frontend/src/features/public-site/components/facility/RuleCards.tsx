import { FeatureIcon } from '@/features/public-site/components/FeatureCard';
import classes from '@/features/public-site/components/facility/RuleCards.module.scss';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Props = { rules: PageFrontMatter<'facilities'>['rules']['items'] };

/**
 * House rules. Mobile: one bordered list of compact rows (icon, short title, small detail line).
 * ≥ sm: separate icon cards, three per row on wide screens.
 */
export function RuleCards({ rules }: Props) {
  return (
    <ul className={classes.list}>
      {rules.map((r) => (
        <li key={r.title} className={classes.rule}>
          <span className={classes.icon}>
            <FeatureIcon name={r.icon} size={20} />
          </span>
          <div className={classes.text}>
            <h3 className={classes.title}>{r.title}</h3>
            {r.body ? <p className={classes.body}>{r.body}</p> : null}
          </div>
        </li>
      ))}
    </ul>
  );
}
