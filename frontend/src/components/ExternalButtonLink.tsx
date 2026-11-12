import { Button, type ButtonProps } from '@mantine/core';
import { IconExternalLink } from '@tabler/icons-react';
import type { ReactNode } from 'react';

type Props = ButtonProps & {
  href: string;
  children: ReactNode;
  /** Visually hidden suffix, e.g. "(opens in a new tab)". */
  newTabLabel: string;
};

/** Button-styled external link: new tab, no opener and no referrer. Usable from server components. */
export function ExternalButtonLink({ href, children, newTabLabel, ...props }: Props) {
  return (
    <Button
      component='a'
      href={href}
      target='_blank'
      rel='noopener noreferrer'
      rightSection={<IconExternalLink size={18} aria-hidden />}
      {...props}
    >
      {children}
      <span className='visually-hidden'>{newTabLabel}</span>
    </Button>
  );
}
