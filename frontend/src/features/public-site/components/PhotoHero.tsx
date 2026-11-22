import { Container, Stack, Text, Title } from '@mantine/core';
import { Fragment, type ReactNode } from 'react';
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

/**
 * Splits a headline after each comma (、，,) so it can wrap there with `<wbr>`. With `word-break:
 * keep-all` the CJK phrases stay whole ("佐久で、/ 暮らすように泊まる。" instead of an orphaned "る。").
 */
const phrases = (title: string) => title.split(/(?<=[、，,])/);

/** Edge-to-edge hero: horizontally sliding photos (swipe, arrows, dots), portrait crop on mobile, wide on desktop, text on a dark scrim (DESIGN §5.1). */
export function PhotoHero({ slides, labels, title, lead, actions }: Props) {
  return (
    <section className={classes.hero} aria-labelledby='hero-title'>
      <HeroSlideshow slides={slides} labels={labels} />
      <div className={classes.content}>
        <Container size={1200}>
          <Stack gap='md' maw={680}>
            <Title
              id='hero-title'
              order={1}
              c='white'
              fz={{ base: 'clamp(1.625rem, 8vw, 1.875rem)', sm: 40, md: 52 }}
              lh={1.25}
              className={classes.title}
            >
              {phrases(title).map((part, i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: static split of a fixed string
                <Fragment key={i}>
                  {i > 0 ? <wbr /> : null}
                  {part}
                </Fragment>
              ))}
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
