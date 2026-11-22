'use client';

import { Button, type ButtonProps } from '@mantine/core';
import type { ComponentProps } from 'react';
import { Link } from '@/i18n/navigation';

type ButtonLinkProps = ButtonProps & Pick<ComponentProps<typeof Link>, 'href'>;

/** Mantine Button rendered as a locale-aware link; usable from server components. */
export function ButtonLink({ href, ...props }: ButtonLinkProps) {
  return <Button component={Link} href={href} {...props} />;
}
