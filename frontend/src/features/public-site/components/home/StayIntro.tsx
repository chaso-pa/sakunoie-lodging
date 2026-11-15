import { FeatureIcon } from '@/features/public-site/components/FeatureCard';
import classes from '@/features/public-site/components/home/StayIntro.module.scss';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Props = { id: string; stay: PageFrontMatter<'home'>['stay'] };

/**
 * "What kind of stay": an appeal rather than a disclaimer. Heading on the left, the short paragraph
 * and the private/shared/guests chips on the right (stacked on mobile).
 */
export function StayIntro({ id, stay }: Props) {
  return (
    <div className={classes.grid}>
      <div className={classes.heading}>
        <p className={classes.eyebrow}>{stay.eyebrow}</p>
        <h2 id={`${id}-title`} className={classes.title}>
          {stay.title}
        </h2>
      </div>
      <div className={classes.body}>
        <p className={classes.text}>{stay.body}</p>
        <ul className={classes.chips}>
          {stay.chips.map((chip) => (
            <li key={chip.text} className={classes.chip}>
              <span className={classes.chipIcon}>
                <FeatureIcon name={chip.icon} size={18} />
              </span>
              {chip.text}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
