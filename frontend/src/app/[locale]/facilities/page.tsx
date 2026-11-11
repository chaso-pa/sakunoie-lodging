import { Anchor, Container, Group, SimpleGrid, Stack, Title } from '@mantine/core';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { AirbnbCta } from '@/components/AirbnbCta';
import { defaultFacility } from '@/config/site';
import { AmenityChip, ContentPageHeader, loadPage, MarkdownContent, PhotoGallery } from '@/features/public-site';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { meta } = await loadPage((await params).locale as Locale, 'facilities');
  return { title: meta.title, description: meta.description };
}

export default async function FacilitiesPage({ params }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'publicSite' });
  const { meta, body } = await loadPage(locale, 'facilities');
  const facility = defaultFacility;
  const gallery = facility.photos.filter((p) => p.id !== 'hero').map((photo) => ({ photo, alt: photo.alt[locale] }));

  return (
    <Container size={1200} py={{ base: 40, md: 64 }}>
      <Stack gap='xl'>
        <ContentPageHeader title={meta.title} intro={body} placeholder={meta.placeholder} />
        <nav aria-label={t('sectionNav')}>
          <Group gap='xs' wrap='wrap'>
            {meta.sections.map((s) => (
              <Anchor key={s.id} href={`#${s.id}`} px='sm' py={10} fw={600} mih={44} display='inline-flex'>
                {s.title}
              </Anchor>
            ))}
          </Group>
        </nav>
        <PhotoGallery
          photos={gallery}
          badge={t('placeholderPhoto')}
          labels={{
            region: t('galleryLabel'),
            prev: t('galleryPrev'),
            next: t('galleryNext'),
            positionTemplate: t.raw('galleryPosition') as string
          }}
        />
        <Stack gap='lg' maw={760}>
          {meta.sections.map((s) => (
            <section key={s.id} id={s.id} aria-labelledby={`${s.id}-title`} style={{ scrollMarginTop: 80 }}>
              <Title id={`${s.id}-title`} order={2} fz={{ base: 22, md: 26 }} mb='sm'>
                {s.title}
              </Title>
              <MarkdownContent>{s.body}</MarkdownContent>
              {s.id === 'amenities' ? (
                <SimpleGrid cols={2} spacing='sm' aria-label={t('confirmedAmenities')}>
                  {facility.amenities.map((a) => (
                    <AmenityChip key={a.code} icon={a.icon} label={a.label[locale]} />
                  ))}
                </SimpleGrid>
              ) : null}
            </section>
          ))}
        </Stack>
        <div>
          <AirbnbCta size='lg' />
        </div>
      </Stack>
    </Container>
  );
}
