import { Container, Stack, Text, Title } from '@mantine/core';
import type { ReactNode } from 'react';
import { type HeroSlide, HeroSlideshow } from '@/features/public-site/components/HeroSlideshow';
import classes from '@/features/public-site/components/PhotoHero.module.scss';

type Props = {
  slides: HeroSlide[];
  labels: {
    region: string;
    roleDescription: string;
    pause: string;
    play: string;
    prev: string;
    next: string;
    goTo: string[];
    badge: string;
  };
  title: string;
  lead: string;
  actions: ReactNode;
};

/** Edge-to-edge hero: horizontally sliding photos (swipe, arrows, dots), portrait crop on mobile, wide on desktop, text on a dark scrim (DESIGN §5.1). */
export function PhotoHero({ slides, labels, title, lead, actions }: Props) {
  return (
    <section className={classes.hero} aria-labelledby='hero-title'>
      <HeroSlideshow slides={slides} labels={labels} />
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
