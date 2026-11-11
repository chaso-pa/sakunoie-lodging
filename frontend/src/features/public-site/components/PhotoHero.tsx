import { Badge, Container, Stack, Text, Title } from '@mantine/core';
import Image from 'next/image';
import type { ReactNode } from 'react';
import type { FacilityPhoto } from '@/config/site';
import classes from '@/features/public-site/components/PhotoHero.module.scss';

type Props = { photo: FacilityPhoto; alt: string; badge: string; title: string; lead: string; actions: ReactNode };

/** Edge-to-edge hero: portrait crop on mobile, wide on desktop, text on a dark scrim (DESIGN §5.1). */
export function PhotoHero({ photo, alt, badge, title, lead, actions }: Props) {
  return (
    <section className={classes.hero} aria-labelledby='hero-title'>
      <Image
        src={photo.src}
        alt={alt}
        fill
        priority
        unoptimized
        sizes='100vw'
        style={{ objectFit: 'cover', objectPosition: photo.focalPoint }}
      />
      <div className={classes.scrim} aria-hidden />
      {photo.placeholder ? (
        <Badge className={classes.badge} color='dark' variant='filled' radius='sm' style={{ textTransform: 'none' }}>
          {badge}
        </Badge>
      ) : null}
      <div className={classes.content}>
        <Container size={1200}>
          <Stack gap='md' maw={680}>
            <Title id='hero-title' order={1} c='white' fz={{ base: 30, sm: 40, md: 52 }} lh={1.25}>
              {title}
            </Title>
            <Text c='white' fz={{ base: 'md', md: 'lg' }}>
              {lead}
            </Text>
            {actions}
          </Stack>
        </Container>
      </div>
    </section>
  );
}
