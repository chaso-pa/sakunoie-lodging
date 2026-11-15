import { Container } from '@mantine/core';
import { IconArrowRight } from '@tabler/icons-react';
import type { ComponentProps } from 'react';
import classes from '@/features/public-site/components/home/GuestStrip.module.scss';
import { Link } from '@/i18n/navigation';

type Href = ComponentProps<typeof Link>['href'];
type Props = { title: string; body: string; links: { href: Href; label: string }[] };

/**
 * For guests who have already booked: a quiet strip with text links (registration, survey), kept
 * visually lighter than the booking CTA so it never competes with it.
 */
export function GuestStrip({ title, body, links }: Props) {
  return (
    <aside aria-labelledby='guests-title' className={classes.strip}>
      <Container size={1200} className={classes.inner}>
        <div className={classes.text}>
          <h2 id='guests-title' className={classes.title}>
            {title}
          </h2>
          <p className={classes.body}>{body}</p>
        </div>
        <ul className={classes.links}>
          {links.map((l) => (
            <li key={l.label}>
              <Link href={l.href} className={classes.link}>
                {l.label}
                <IconArrowRight size={16} aria-hidden />
              </Link>
            </li>
          ))}
        </ul>
      </Container>
    </aside>
  );
}
