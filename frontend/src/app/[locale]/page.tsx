import { Card, Container, Group, SimpleGrid, Stack, Text, Title } from '@mantine/core';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { AirbnbCta } from '@/components/AirbnbCta';
import { ButtonLink } from '@/components/ButtonLink';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import { defaultFacility, defaultFacilityCode } from '@/config/site';
import { FeatureCard, loadPage, MarkdownContent, PhotoHero, PlaceholderPhoto } from '@/features/public-site';
import { PlaceholderBanner } from '@/features/public-site/components/PlaceholderBanner';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const locale = (await params).locale as Locale;
  const t = await getTranslations({ locale, namespace: 'common' });
  const { meta } = await loadPage(locale, 'home');
  return { title: { absolute: `${t('brand')} | ${meta.title}` }, description: meta.description };
}

export default async function HomePage({ params }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const t = await getTranslations({ locale });
  const { meta, body } = await loadPage(locale, 'home');
  const photo = (id: string) => defaultFacility.photos.find((p) => p.id === id) ?? defaultFacility.photos[0];
  const previews = ['room', 'shared', 'neighborhood'].map(photo);

  return (
    <>
      <PhotoHero
        photo={photo('hero')}
        alt={photo('hero').alt[locale]}
        badge={t('publicSite.placeholderPhoto')}
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
      <Container size={1200} py={{ base: 40, md: 64 }}>
        <Stack gap={56}>
          <Stack gap='md'>
            <PlaceholderBanner show={meta.placeholder} />
            <InfoBanner tone='info' title={meta.stayType.title}>
              <Text>{meta.stayType.body}</Text>
            </InfoBanner>
          </Stack>

          <section aria-labelledby='features-title'>
            <Title id='features-title' order={2} fz={{ base: 24, md: 30 }} mb='lg'>
              {t('publicSite.featuresTitle')}
            </Title>
            <SimpleGrid cols={{ base: 1, sm: 3 }} spacing='lg'>
              {meta.features.map((f) => (
                <FeatureCard key={f.title} icon={f.icon} title={f.title} body={f.body} />
              ))}
            </SimpleGrid>
          </section>

          <section aria-labelledby='photos-title'>
            <Group justify='space-between' align='flex-end' mb='lg'>
              <Title id='photos-title' order={2} fz={{ base: 24, md: 30 }}>
                {t('publicSite.photosTitle')}
              </Title>
              <ButtonLink href={paths.public.facilities.getHref()} variant='subtle'>
                {t('cta.details')}
              </ButtonLink>
            </Group>
            <SimpleGrid cols={{ base: 1, sm: 3 }} spacing='lg'>
              {previews.map((p) => (
                <PlaceholderPhoto
                  key={p.id}
                  photo={p}
                  alt={p.alt[locale]}
                  badge={t('publicSite.placeholderPhoto')}
                  sizes='(min-width: 768px) 33vw, 100vw'
                />
              ))}
            </SimpleGrid>
          </section>

          <section>
            <MarkdownContent>{body}</MarkdownContent>
          </section>

          <SimpleGrid cols={{ base: 1, md: 3 }} spacing='lg'>
            <Card padding='lg'>
              <Stack gap='sm' h='100%'>
                <Title order={2} fz='xl'>
                  {t('publicSite.bookingTitle')}
                </Title>
                <Text style={{ flex: 1 }}>{t('publicSite.bookingBody')}</Text>
                <AirbnbCta />
              </Stack>
            </Card>
            <Card padding='lg'>
              <Stack gap='sm' h='100%'>
                <Title order={2} fz='xl'>
                  {t('home.registerTitle')}
                </Title>
                <Text style={{ flex: 1 }}>{t('home.registerBody')}</Text>
                <ButtonLink href={paths.registration.wizard.getHref(defaultFacilityCode)} variant='outline'>
                  {t('home.registerCta')}
                </ButtonLink>
              </Stack>
            </Card>
            <Card padding='lg'>
              <Stack gap='sm' h='100%'>
                <Title order={2} fz='xl'>
                  {t('publicSite.surveyTitle')}
                </Title>
                <Text style={{ flex: 1 }}>{t('publicSite.surveyBody')}</Text>
                <ButtonLink href={paths.public.survey.getHref()} variant='outline'>
                  {t('publicSite.surveyCta')}
                </ButtonLink>
              </Stack>
            </Card>
          </SimpleGrid>
        </Stack>
      </Container>
    </>
  );
}
