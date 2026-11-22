import { Box, Group } from '@mantine/core';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { AirbnbCta } from '@/components/AirbnbCta';
import { ButtonLink } from '@/components/ButtonLink';
import { airbnbListingId } from '@/config/airbnb';
import { paths } from '@/config/paths';
import { defaultFacility } from '@/config/site';
import {
  AccessRoute,
  AirbnbEmbed,
  ArrowLink,
  BookingBand,
  BookingCard,
  Highlights,
  HomeSection,
  loadPage,
  NearbyCards,
  PhotoHero,
  PlaceholderBanner,
  SpacesPreview,
  StayIntro
} from '@/features/public-site';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const locale = (await params).locale as Locale;
  const t = await getTranslations({ locale, namespace: 'common' });
  const { meta } = await loadPage(locale, 'home');
  return { title: { absolute: `${t('brand')} | ${meta.title}` }, description: meta.description };
}

const HERO_SLIDES = ['room', 'shared', 'dining'];

/**
 * Home landing page, in conversion order: hero → what kind of stay → why stay here → room and shared
 * spaces → around Saku → Airbnb listing (reviews) beside price/check-in and the primary CTA →
 * access → a quiet strip for guests who have already booked. Sections alternate canvas and surface.
 * Text comes from `content/{locale}/home.md`; price, check-in/out and the booking card from
 * `facilities.md` (one source for those facts); photos and amenities from the facility JSON.
 */
export default async function HomePage({ params }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const t = await getTranslations({ locale });
  const { meta } = await loadPage(locale, 'home');
  const { meta: facilities } = await loadPage(locale, 'facilities');
  const facility = defaultFacility;
  const photoById = (id: string) => facility.photos.find((p) => p.id === id);
  const heroSlides = HERO_SLIDES.map((id) => photoById(id) ?? facility.photos[0]);
  const checkInOut = facilities.keyFacts.find((f) => 'lines' in f);
  const parking = facility.amenities.find((a) => a.code === 'free_parking');
  const listingId = airbnbListingId(facility.airbnbUrl);

  const spaceTiles = [
    { ...meta.spaces.private, kind: 'private' as const },
    ...meta.spaces.shared.map((s) => ({ ...s, kind: 'shared' as const }))
  ].flatMap(({ photo, label, kind }) => {
    const p = photoById(photo);
    return p ? [{ photo: p, label, kind }] : [];
  });

  return (
    <>
      <PhotoHero
        slides={heroSlides.map((p) => ({
          src: p.src,
          alt: p.alt[locale],
          focalPoint: p.focalPoint,
          placeholder: p.placeholder
        }))}
        labels={{
          region: t('publicSite.heroSlidesLabel'),
          roleDescription: t('publicSite.galleryRoleDescription'),
          pause: t('publicSite.heroPause'),
          play: t('publicSite.heroPlay'),
          prev: t('publicSite.galleryPrev'),
          next: t('publicSite.galleryNext'),
          goTo: heroSlides.map((_, i) => t('publicSite.heroGoTo', { n: i + 1 })),
          badge: t('publicSite.placeholderPhoto')
        }}
        title={meta.title}
        lead={meta.lead}
        actions={
          <Group gap='sm' grow preventGrowOverflow={false} wrap='wrap'>
            <AirbnbCta size='lg' />
            <ButtonLink href={paths.public.facilities.getHref()} size='lg' variant='white' color='dark'>
              {t('cta.details')}
            </ButtonLink>
          </Group>
        }
      />

      <HomeSection id='stay' tone='canvas' title={meta.stay.title} hideHeader>
        <StayIntro id='stay' stay={meta.stay} />
      </HomeSection>

      <HomeSection id='highlights' tone='surface' title={meta.highlights.title}>
        <Highlights items={meta.highlights.items} photoById={photoById} locale={locale} />
      </HomeSection>

      <HomeSection
        id='spaces'
        tone='canvas'
        title={meta.spaces.title}
        intro={meta.spaces.body}
        aside={<ArrowLink href={paths.public.facilities.getHref()}>{t('home.spacesLink')}</ArrowLink>}
      >
        <SpacesPreview
          tiles={spaceTiles}
          badges={{ private: t('publicSite.badgePrivate'), shared: t('publicSite.badgeShared') }}
          locale={locale}
        />
        <Group mt='sm' hiddenFrom='sm'>
          <ArrowLink href={paths.public.facilities.getHref()}>{t('home.spacesLink')}</ArrowLink>
        </Group>
      </HomeSection>

      <HomeSection id='around' tone='surface' title={meta.around.title} intro={meta.around.intro}>
        <NearbyCards items={meta.around.items} />
      </HomeSection>

      <BookingBand
        airbnb={meta.airbnb}
        embed={
          listingId ? (
            <AirbnbEmbed
              listingId={listingId}
              listingUrl={facility.airbnbUrl}
              labels={{
                frameTitle: t('home.airbnbFrameTitle'),
                open: t('home.airbnbOpen'),
                newTab: t('cta.opensInNewTab')
              }}
            />
          ) : null
        }
        booking={
          <BookingCard
            booking={facilities.booking}
            checkInOut={checkInOut && 'lines' in checkInOut ? checkInOut : undefined}
          />
        }
      />

      <HomeSection id='access' tone='surface' title={meta.access.title} compact>
        <AccessRoute
          access={meta.access}
          area={facility.area[locale]}
          facts={parking ? [{ icon: parking.icon, label: parking.label[locale] }] : []}
          routeLabel={t('home.routeLabel')}
        />
        {meta.placeholder ? (
          <Box mt='xl'>
            <PlaceholderBanner show body={t('home.placeholderBody')} />
          </Box>
        ) : null}
      </HomeSection>
    </>
  );
}
