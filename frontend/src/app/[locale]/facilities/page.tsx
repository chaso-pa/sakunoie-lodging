import { Container, Group, Paper, Stack, Text, ThemeIcon } from '@mantine/core';
import { IconMapPin } from '@tabler/icons-react';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { paths } from '@/config/paths';
import { defaultFacility, type FacilityPhoto } from '@/config/site';
import {
  AmenityGrid,
  ArrowLink,
  BookingCard,
  BookingReminder,
  ContentPageHeader,
  FacilityLayout,
  FacilitySection,
  KeyFacts,
  loadPage,
  PhotoGallery,
  PlaceholderMark,
  RuleCards,
  SectionNav,
  SpaceCards
} from '@/features/public-site';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { meta } = await loadPage((await params).locale as Locale, 'facilities');
  return { title: meta.title, description: meta.description };
}

/**
 * Facility details, "key facts first": header and gallery, the facts bar, private vs shared spaces,
 * amenities, house rules and access, with the price/booking card beside them (sticky on desktop).
 * Text comes from `content/{locale}/facilities.md`; amenities and photos from the facility JSON.
 */
export default async function FacilitiesPage({ params }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'publicSite' });
  const { meta, body } = await loadPage(locale, 'facilities');
  const facility = defaultFacility;
  const photoById = (id: string) => facility.photos.find((p) => p.id === id);
  const gallery = facility.photos.filter((p) => p.id !== 'hero').map((photo) => ({ photo, alt: photo.alt[locale] }));
  const placeholderMark = t('placeholderMark');

  const nav = [
    { id: 'spaces', title: meta.spaces.title },
    { id: 'amenities', title: meta.amenities.title },
    { id: 'rules', title: meta.rules.title },
    { id: 'access', title: meta.access.title },
    { id: 'booking', title: t('priceTitle') }
  ];

  return (
    <Container size={1200} py={{ base: 24, md: 56 }}>
      <Stack gap='clamp(1.25rem, 3vw, 2rem)'>
        <ContentPageHeader
          eyebrow={facility.area[locale]}
          title={meta.title}
          intro={body}
          placeholder={meta.placeholder}
        />
        <SectionNav label={t('sectionNav')} items={nav} />
        <PhotoGallery
          photos={gallery}
          badge={t('placeholderPhoto')}
          labels={{
            region: t('galleryLabel'),
            roleDescription: t('galleryRoleDescription'),
            prev: t('galleryPrev'),
            next: t('galleryNext'),
            positionTemplate: t.raw('galleryPosition') as string
          }}
        />

        <FacilityLayout
          facts={<KeyFacts label={t('keyFactsLabel')} facts={meta.keyFacts} />}
          booking={<BookingCard booking={meta.booking} />}
          endCta={<BookingReminder />}
        >
          <FacilitySection id='spaces' title={meta.spaces.title}>
            <SpaceCards
              spaces={meta.spaces}
              privatePhoto={photoById(meta.spaces.private.photo)}
              sharedPhotos={meta.spaces.shared.photos.map(photoById).filter((p): p is FacilityPhoto => p !== undefined)}
              locale={locale}
            />
          </FacilitySection>

          <FacilitySection id='amenities' title={meta.amenities.title}>
            <AmenityGrid amenities={facility.amenities} locale={locale} />
            {meta.amenities.note ? (
              <Text component='div' size='sm' c='dimmed' mt='sm' lh={1.7}>
                {meta.amenities.notePlaceholder ? (
                  <>
                    <PlaceholderMark label={placeholderMark} />{' '}
                  </>
                ) : null}
                {meta.amenities.note}
              </Text>
            ) : null}
          </FacilitySection>

          <FacilitySection id='rules' title={meta.rules.title}>
            <RuleCards rules={meta.rules.items} />
            <Group mt={4}>
              <ArrowLink href={paths.public.terms.getHref()}>{t('termsLink')}</ArrowLink>
            </Group>
          </FacilitySection>

          <FacilitySection
            id='access'
            title={meta.access.title}
            placeholderLabel={meta.access.placeholder ? placeholderMark : undefined}
          >
            <Paper withBorder radius={16} p={{ base: 14, sm: 'lg' }} bg='var(--sakunoie-surface)'>
              <Group gap='md' align='flex-start' wrap='nowrap'>
                <ThemeIcon variant='light' color='forest' size={44} radius='md'>
                  <IconMapPin size={24} aria-hidden stroke={1.6} />
                </ThemeIcon>
                <Stack gap={4}>
                  <Text fw={700}>{facility.area[locale]}</Text>
                  <Text c='dimmed' fz={{ base: 'sm', sm: 'md' }} lh={1.7}>
                    {meta.access.body}
                  </Text>
                </Stack>
              </Group>
            </Paper>
          </FacilitySection>
        </FacilityLayout>
      </Stack>
    </Container>
  );
}
