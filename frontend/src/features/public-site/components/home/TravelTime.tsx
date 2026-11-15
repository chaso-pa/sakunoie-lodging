import { IconCar, IconTrain, IconWalk } from '@tabler/icons-react';
import classes from '@/features/public-site/components/home/TravelTime.module.scss';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Travel = PageFrontMatter<'home'>['around']['items'][number]['travel'];

const icons = { walk: IconWalk, car: IconCar, train: IconTrain } as const;

/** "徒歩約5分" with a walk/car/train icon. The text says the mode, so the icon is decorative. */
export function TravelTime({ travel, className }: { travel: Travel; className?: string }) {
  const Icon = icons[travel.mode];
  return (
    <span className={[classes.time, className].filter(Boolean).join(' ')} data-mode={travel.mode}>
      <Icon size={16} aria-hidden stroke={1.8} />
      {travel.text}
    </span>
  );
}
