'use client';

import { Burger, Drawer, NavLink, Stack } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useEffect } from 'react';
import { ExternalButtonLink } from '@/components/ExternalButtonLink';
import classes from '@/components/SiteHeader.module.scss';
import { Link, usePathname } from '@/i18n/navigation';

type Props = {
  links: { href: string; label: string }[];
  labels: { open: string; title: string; nav: string };
  airbnb: { href: string; label: string; newTab: string };
};

/** Hamburger + drawer below the desktop breakpoint (DESIGN §6). */
export function SiteMobileMenu({ links, labels, airbnb }: Props) {
  const [opened, { toggle, close }] = useDisclosure(false);
  const pathname = usePathname();
  // biome-ignore lint/correctness/useExhaustiveDependencies: close the drawer after navigation
  useEffect(() => close(), [pathname]);

  return (
    <div className={classes.mobileOnly}>
      <Burger opened={opened} onClick={toggle} aria-label={labels.open} aria-expanded={opened} size='sm' p={12} />
      <Drawer opened={opened} onClose={close} title={labels.title} position='right' size='xs'>
        <nav aria-label={labels.nav}>
          <Stack gap={4}>
            {links.map((l) => (
              <NavLink
                key={l.href}
                component={Link}
                href={l.href}
                label={l.label}
                active={pathname === l.href}
                mih={44}
              />
            ))}
            <ExternalButtonLink href={airbnb.href} newTabLabel={airbnb.newTab} mt='md' fullWidth>
              {airbnb.label}
            </ExternalButtonLink>
          </Stack>
        </nav>
      </Drawer>
    </div>
  );
}
