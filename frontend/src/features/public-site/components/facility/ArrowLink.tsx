import { IconArrowRight } from '@tabler/icons-react';
import type { ComponentProps, ReactNode } from 'react';
import classes from '@/features/public-site/components/facility/ArrowLink.module.scss';
import { Link } from '@/i18n/navigation';

type Props = { href: ComponentProps<typeof Link>['href']; children: ReactNode };

/** Underlined internal text link with a trailing arrow (44px tall target). */
export function ArrowLink({ href, children }: Props) {
  return (
    <Link href={href} className={classes.link}>
      {children}
      <IconArrowRight size={16} aria-hidden />
    </Link>
  );
}
